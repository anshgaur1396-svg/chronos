package chronos

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status = %d, want = %d", rec.Code, http.StatusOK)
	}

	var got healthResponse
	err := json.Unmarshal(rec.Body.Bytes(), &got)

	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got.Status != "ok" {
		t.Errorf("got = %q want = %q", got.Status, "ok")
	}
}
