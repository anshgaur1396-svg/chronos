package chronos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIdempotencyStoreExecutesOnlyOnce(t *testing.T) {
	store := newIdempotencyStore()

	var executions atomic.Int32

	operation := func() ProrationResponse {
		executions.Add(1)
		return ProrationResponse{
			SubscriptionID:   "sub_123",
			AdjustmentPaisa:  500,
			AdjustmentStatus: statusOwes,
		}
	}

	fingerprint := "fingerprint-123"

	first, conflict, err := store.execute(
		context.Background(),
		"key-123",
		fingerprint,
		operation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conflict {
		t.Fatal("unexpected conflict")
	}

	second, conflict, err := store.execute(
		context.Background(),
		"key-123",
		fingerprint,
		operation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conflict {
		t.Fatal("unexpected conflict")
	}

	if executions.Load() != 1 {
		t.Fatalf("expected 1 execution, got %d", executions.Load())
	}

	if first != second {
		t.Fatalf("expected replayed response to match: first=%+v second=%+v", first, second)
	}
}

func TestIdempotencyStoreRejectsDifferentFingerprint(t *testing.T) {
	store := newIdempotencyStore()

	operation := func() ProrationResponse {
		return ProrationResponse{SubscriptionID: "sub_123"}
	}

	_, conflict, err := store.execute(
		context.Background(),
		"key-123",
		"fingerprint-A",
		operation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if conflict {
		t.Fatal("first request must not conflict")
	}

	_, conflict, err = store.execute(
		context.Background(),
		"key-123",
		"fingerprint-B",
		operation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !conflict {
		t.Fatal("expected conflict for a different fingerprint")
	}
}

func TestIdempotencyStoreConcurrentRequestsExecuteOnce(t *testing.T) {
	store := newIdempotencyStore()

	var executions atomic.Int32

	start := make(chan struct{})
	var wg sync.WaitGroup

	const requestCount = 20

	responses := make([]ProrationResponse, requestCount)
	errors := make([]error, requestCount)
	conflicts := make([]bool, requestCount)

	operation := func() ProrationResponse {
		executions.Add(1)
		time.Sleep(50 * time.Millisecond)

		return ProrationResponse{
			SubscriptionID:   "sub_concurrent",
			AdjustmentPaisa:  750,
			AdjustmentStatus: statusOwes,
		}
	}

	for i := 0; i < requestCount; i++ {
		wg.Add(1)

		go func(index int) {
			defer wg.Done()
			<-start

			responses[index], conflicts[index], errors[index] = store.execute(
				context.Background(),
				"shared-key",
				"shared-fingerprint",
				operation,
			)
		}(i)
	}

	close(start)
	wg.Wait()

	if executions.Load() != 1 {
		t.Fatalf("expected exactly 1 execution, got %d", executions.Load())
	}

	for i := 0; i < requestCount; i++ {
		if errors[i] != nil {
			t.Fatalf("request %d returned error: %v", i, errors[i])
		}
		if conflicts[i] {
			t.Fatalf("request %d unexpectedly conflicted", i)
		}
		if responses[i].SubscriptionID != "sub_concurrent" {
			t.Fatalf("request %d received unexpected response: %+v", i, responses[i])
		}
	}
}

func TestProrateHandlerIdempotencyConflict(t *testing.T) {
	router := NewRouter()

	firstBody := `{
		"subscription_id": "sub_123",
		"old_plan": {"name": "Basic", "price_paisa": 10000},
		"new_plan": {"name": "Pro", "price_paisa": 20000},
		"total_days": 30,
		"used_days": 10
	}`

	secondBody := `{
		"subscription_id": "sub_123",
		"old_plan": {"name": "Basic", "price_paisa": 10000},
		"new_plan": {"name": "Pro", "price_paisa": 30000},
		"total_days": 30,
		"used_days": 10
	}`

	sendRequest := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPost,
			"/v1/prorate",
			strings.NewReader(body),
		)
		req.Header.Set("Idempotency-Key", "sub-123-key")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	first := sendRequest(firstBody)
	if first.Code != http.StatusOK {
		t.Fatalf("expected first request status 200, got %d: %s", first.Code, first.Body.String())
	}

	// Same key and same payload must replay successfully.
	replay := sendRequest(firstBody)
	if replay.Code != http.StatusOK {
		t.Fatalf("expected replay status 200, got %d: %s", replay.Code, replay.Body.String())
	}

	if first.Body.String() != replay.Body.String() {
		t.Fatalf("expected identical replay response; first=%s replay=%s", first.Body.String(), replay.Body.String())
	}

	// Same key but a different payload must conflict.
	conflict := sendRequest(secondBody)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("expected conflict status 409, got %d: %s", conflict.Code, conflict.Body.String())
	}
}

func TestInvalidRequestDoesNotReserveIdempotencyKey(t *testing.T) {
	router := NewRouter()

	invalidBody := `{
		"subscription_id": "",
		"old_plan": {"name": "Basic", "price_paisa": 10000},
		"new_plan": {"name": "Pro", "price_paisa": 20000},
		"total_days": 30,
		"used_days": 10
	}`

	validBody := `{
		"subscription_id": "sub_123",
		"old_plan": {"name": "Basic", "price_paisa": 10000},
		"new_plan": {"name": "Pro", "price_paisa": 20000},
		"total_days": 30,
		"used_days": 10
	}`

	sendRequest := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPost,
			"/v1/prorate",
			strings.NewReader(body),
		)
		req.Header.Set("Idempotency-Key", "K")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		return rec
	}

	// First request: invalid domain data.
	first := sendRequest(invalidBody)

	if first.Code != http.StatusUnprocessableEntity {
		t.Fatalf(
			"expected first request status 422, got %d: %s",
			first.Code,
			first.Body.String(),
		)
	}

	// Second request: valid data, but reuses the same key.
	second := sendRequest(validBody)

	if second.Code != http.StatusOK {
		t.Fatalf(
			"expected second request status 200, got %d: %s",
			second.Code,
			second.Body.String(),
		)
	}
}

func TestIdempotencyFingerprintUsesDecodedRequest(t *testing.T) {
	router := NewRouter()

	firstBody := `{
        "subscription_id": "sub_123",
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

	secondBody := `{"used_days":10,"total_days":30,"new_plan":{"price_paisa":20000,"name":"Pro"},"old_plan":{"price_paisa":10000,"name":"Basic"},"subscription_id":"sub_123"}`

	sendRequest := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(
			http.MethodPost,
			"/v1/prorate",
			strings.NewReader(body),
		)
		req.Header.Set("Idempotency-Key", "decoded-request-test")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	first := sendRequest(firstBody)
	if first.Code != http.StatusOK {
		t.Fatalf("first request: expected status %d, got %d; body: %s",
			http.StatusOK, first.Code, first.Body.String())
	}

	second := sendRequest(secondBody)
	if second.Code != http.StatusOK {
		t.Fatalf("second request: expected status %d, got %d; body: %s",
			http.StatusOK, second.Code, second.Body.String())
	}

	if first.Body.String() != second.Body.String() {
		t.Fatalf(
			"same logical request produced different responses:\nfirst:  %s\nsecond: %s",
			first.Body.String(),
			second.Body.String(),
		)
	}
}
