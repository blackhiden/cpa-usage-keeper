package telegram

import (
	"strings"
	"sync"
	"time"
)

const (
	// DefaultFailureTTL bounds how long a recorded failure can be reported.
	DefaultFailureTTL = 10 * time.Minute
	// defaultMaxFailures bounds the in-memory record table.
	defaultMaxFailures = 4096
)

// Failure is the server-authored model-test failure payload (REC-012). The
// browser never supplies any of these fields.
type Failure struct {
	Model      string
	Mode       string
	StatusCode int
	ErrorCode  string
	Reason     string
}

type storedFailure struct {
	failure    Failure
	recordedAt time.Time
}

// FailureStore keeps the last failed test per session scope and model so the
// notify endpoint can author the message without trusting the client payload.
// It is in-memory only: state is lost on restart and is per-process, which is
// acceptable for the single-instance deployment.
type FailureStore struct {
	mu      sync.Mutex
	ttl     time.Duration
	max     int
	now     func() time.Time
	entries map[string]storedFailure
}

// NewFailureStore builds a bounded store; a zero or negative TTL uses the
// default, and a non-positive max uses the default bound.
func NewFailureStore(ttl time.Duration) *FailureStore {
	if ttl <= 0 {
		ttl = DefaultFailureTTL
	}
	return &FailureStore{ttl: ttl, max: defaultMaxFailures, now: time.Now, entries: make(map[string]storedFailure)}
}

func failureKey(scope, model string) string {
	return strings.TrimSpace(scope) + "\x00" + strings.TrimSpace(model)
}

// Record stores the failure for the scope and model, replacing any previous
// entry and pruning expired rows.
func (s *FailureStore) Record(scope string, failure Failure) {
	if s == nil || strings.TrimSpace(scope) == "" || strings.TrimSpace(failure.Model) == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	s.pruneLocked(now)
	key := failureKey(scope, failure.Model)
	if _, exists := s.entries[key]; !exists {
		s.makeCapacityLocked()
	}
	s.entries[key] = storedFailure{failure: failure, recordedAt: now}
}

// Lookup returns a non-expired recorded failure for the scope and model.
func (s *FailureStore) Lookup(scope, model string) (Failure, bool) {
	if s == nil || strings.TrimSpace(scope) == "" || strings.TrimSpace(model) == "" {
		return Failure{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[failureKey(scope, model)]
	if !ok || s.now().Sub(entry.recordedAt) >= s.ttl {
		if ok {
			delete(s.entries, failureKey(scope, model))
		}
		return Failure{}, false
	}
	return entry.failure, true
}

func (s *FailureStore) pruneLocked(now time.Time) {
	for key, entry := range s.entries {
		if now.Sub(entry.recordedAt) >= s.ttl {
			delete(s.entries, key)
		}
	}
}

func (s *FailureStore) makeCapacityLocked() {
	if len(s.entries) < s.max {
		return
	}
	var oldestKey string
	var oldestAt time.Time
	for key, entry := range s.entries {
		if oldestKey == "" || entry.recordedAt.Before(oldestAt) {
			oldestKey, oldestAt = key, entry.recordedAt
		}
	}
	delete(s.entries, oldestKey)
}
