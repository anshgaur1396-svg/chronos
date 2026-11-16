package chronos

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const maxRequestBodySize = 1 << 20 // 1 MiB

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func writeJSONError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   code,
		Message: message,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, healthResponse{
		Status: "ok",
	})
}

func decodeRequest(
	w http.ResponseWriter,
	r *http.Request,
	req *ProrationRequest,
) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(req); err != nil {
		return err
	}

	var extra any
	err := decoder.Decode(&extra)

	if !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}

	return nil
}

func handleProrate(
	w http.ResponseWriter,
	r *http.Request,
	store *idempotencyStore,
) {
	var req ProrationRequest

	// 1. Decode the request.
	if err := decodeRequest(w, r, &req); err != nil {
		var maxBytesError *http.MaxBytesError

		if errors.As(err, &maxBytesError) {
			writeJSONError(
				w,
				http.StatusRequestEntityTooLarge,
				"request_too_large",
				"request body exceeds the 1 MiB limit",
			)
			return
		}

		writeJSONError(
			w,
			http.StatusBadRequest,
			"invalid_request",
			"request body must contain exactly one valid JSON object",
		)
		return
	}

	// 2. Validate before touching the idempotency store.
	if err := validate(req); err != nil {
		writeJSONError(
			w,
			http.StatusUnprocessableEntity,
			"invalid_request",
			err.Error(),
		)
		return
	}

	// 3. Without an Idempotency-Key, process normally.
	key := r.Header.Get("Idempotency-Key")

	if key == "" {
		writeJSON(w, http.StatusOK, calculateProration(req))
		return
	}

	// 4. Fingerprint the validated request.
	fingerprint, err := requestFingerprint(req)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"could not fingerprint request",
		)
		return
	}

	// 5. Execute once, wait for an existing execution, or detect a conflict.
	response, conflict, err := store.execute(
		r.Context(),
		key,
		fingerprint,
		func() ProrationResponse {
			return calculateProration(req)
		},
	)

	if err != nil {
		// The client disconnected or cancelled while waiting.
		if r.Context().Err() != nil {
			return
		}

		writeJSONError(
			w,
			http.StatusInternalServerError,
			"internal_error",
			"could not process request",
		)
		return
	}

	if conflict {
		writeJSONError(
			w,
			http.StatusConflict,
			"idempotency_conflict",
			"Idempotency-Key was already used with a different request",
		)
		return
	}

	// 6. Return the original or newly calculated response.
	writeJSON(w, http.StatusOK, response)
}

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()
	store := newIdempotencyStore()

	mux.HandleFunc("GET /v1/health", healthHandler)
	mux.HandleFunc("POST /v1/prorate", func(w http.ResponseWriter, r *http.Request) {
		handleProrate(w, r, store)
	})

	return mux
}
