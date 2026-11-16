package chronos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
)

type idempotencyEntry struct {
	fingerprint string
	response    ProrationResponse
	done        chan struct{}
}

type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]*idempotencyEntry
}

func newIdempotencyStore() *idempotencyStore {
	return &idempotencyStore{
		entries: make(map[string]*idempotencyEntry),
	}
}

// requestFingerprint creates a stable fingerprint from the decoded request.
// JSON marshaling a struct gives a consistent field order.
func requestFingerprint(req ProrationRequest) (string, error) {
	encoded, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// execute either performs the operation once or reuses its completed result.
//
// conflict is true when the key already belongs to a different request.
// A request sharing a key and fingerprint waits for the original execution.
func (s *idempotencyStore) execute(
	ctx context.Context,
	key string,
	fingerprint string,
	operation func() ProrationResponse,
) (response ProrationResponse, conflict bool, err error) {
	s.mu.Lock()

	existing, found := s.entries[key]

	if found {
		if existing.fingerprint != fingerprint {
			s.mu.Unlock()
			return ProrationResponse{}, true, nil
		}

		done := existing.done
		s.mu.Unlock()

		select {
		case <-done:
			return existing.response, false, nil
		case <-ctx.Done():
			return ProrationResponse{}, false, ctx.Err()
		}
	}

	entry := &idempotencyEntry{
		fingerprint: fingerprint,
		done:        make(chan struct{}),
	}

	s.entries[key] = entry
	s.mu.Unlock()

	// Only the request that claimed the key executes the operation.
	result := operation()

	s.mu.Lock()
	entry.response = result
	close(entry.done)
	s.mu.Unlock()

	return result, false, nil
}
