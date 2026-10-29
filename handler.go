package chronos

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type healthResponse struct {
	Status string `json:"status"`
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	body := errorResponse{
		Error:   "invalid_request",
		Message: message,
	}

	_ = json.NewEncoder(w).Encode(body)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := healthResponse{
		Status: "ok",
	}

	body, err := json.Marshal(response)

	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func prorateHandler(w http.ResponseWriter, r *http.Request) {
	var req ProrationRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(
			w,
			http.StatusBadRequest,
			"request body must be valid JSON",
		)
		return
	}

	if err := validate(req); err != nil {
		writeJSONError(
			w,
			http.StatusUnprocessableEntity,
			err.Error(),
		)
		return
	}

	response := calculateProration(req)

	body, err := json.Marshal(response)
	if err != nil {
		writeJSONError(
			w,
			http.StatusInternalServerError,
			"could not encode response",
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func NewRouter() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/health", healthHandler)
	mux.HandleFunc("POST /v1/prorate", prorateHandler)

	return mux
}
