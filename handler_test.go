package chronos

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func TestNewRouter(t *testing.T) {

	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{
			name:       "health route",
			method:     http.MethodGet,
			path:       "/v1/health",
			body:       "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/v1/does-not-exist",
			body:       "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "wrong method",
			method:     http.MethodPost,
			path:       "/v1/health",
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "prorate route",
			method: http.MethodPost,
			path:   "/v1/prorate",
			body: `{
        		"subscription_id": "sub_111",
        		"old_plan": {
            		"name": "Old",
            		"price_paisa": 10000
        		},
        		"new_plan": {
            		"name": "New",
            		"price_paisa": 20000
        		},
        		"total_days": 30,
        		"used_days": 10
    		}`,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			router := NewRouter()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, wanted = %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestProrateHandlerMalformedJSON(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/prorate",
		bytes.NewBufferString(`{"subscription_id":`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body testErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v; body = %s",
			err, rec.Body.String())
	}

	if body.Error != "invalid_request" {
		t.Errorf("error = %q, want %q", body.Error, "invalid_request")
	}
}

func TestProrateHandlerWrongJSONType(t *testing.T) {
	router := NewRouter()

	payload := `{
        "subscription_id": "sub_123",
        "old_plan": {
            "name": "Basic",
            "price_paisa": 10000
        },
        "new_plan": {
            "name": "Pro",
            "price_paisa": 20000
        },
        "total_days": "thirty",
        "used_days": 10
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/prorate",
		bytes.NewBufferString(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s",
			rec.Code, http.StatusBadRequest, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body testErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v; body = %s",
			err, rec.Body.String())
	}

	if body.Error != "invalid_request" {
		t.Errorf("error = %q, want %q", body.Error, "invalid_request")
	}
}

func TestProrateHandlerValidationError(t *testing.T) {
	router := NewRouter()

	payload := `{
        "subscription_id": "",
        "old_plan": {
            "name": "Basic",
            "price_paisa": 10000
        },
        "new_plan": {
            "name": "Pro",
            "price_paisa": 20000
        },
        "total_days": 30,
        "used_days": 10
    }`

	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/prorate",
		bytes.NewBufferString(payload),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body = %s",
			rec.Code, http.StatusUnprocessableEntity, rec.Body.String())
	}

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}

	var body testErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v; body = %s",
			err, rec.Body.String())
	}

	if body.Error != "invalid_request" {
		t.Errorf("error = %q, want %q", body.Error, "invalid_request")
	}

	if body.Message != "subscription_id is required" {
		t.Errorf("message = %q, want %q",
			body.Message, "subscription_id is required")
	}
}
