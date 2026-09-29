package chronos

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewRouter(t *testing.T) {

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "health route",
			method:     http.MethodGet,
			path:       "/v1/health",
			wantStatus: http.StatusOK,
		},
		{
			name:       "unknown route",
			method:     http.MethodGet,
			path:       "/v1/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name: "wrong method",
			method: http.MethodPost,
			path: "/v1/health",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router := NewRouter()
			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, wanted = %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
