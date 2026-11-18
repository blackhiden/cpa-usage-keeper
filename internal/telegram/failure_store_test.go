package telegram

import (
	"testing"
	"time"
)

func TestFailureStoreRecordsAndExpires(t *testing.T) {
	now := time.Unix(1700000000, 0)
	store := NewFailureStore(time.Minute)
	store.now = func() time.Time { return now }
	store.Record("scope-team", Failure{Model: "gpt-test", StatusCode: 502, ErrorCode: "test_failed", Reason: "boom"})
	failure, ok := store.Lookup("scope-team", "gpt-test")
	if !ok || failure.Reason != "boom" || failure.StatusCode != 502 {
		t.Fatalf("lookup=%+v ok=%v", failure, ok)
	}
	now = now.Add(2 * time.Minute)
	if _, ok := store.Lookup("scope-team", "gpt-test"); ok {
		t.Fatal("expired failure should not be returned")
	}
}

func TestFailureStoreIsolatesScopesAndModels(t *testing.T) {
	store := NewFailureStore(DefaultFailureTTL)
	store.Record("scope-a", Failure{Model: "gpt-test", Reason: "a"})
	store.Record("scope-b", Failure{Model: "gpt-test", Reason: "b"})
	store.Record("scope-a", Failure{Model: "other-model", Reason: "c"})
	if failure, ok := store.Lookup("scope-a", "gpt-test"); !ok || failure.Reason != "a" {
		t.Fatalf("scope-a lookup=%+v ok=%v", failure, ok)
	}
	if failure, ok := store.Lookup("scope-b", "gpt-test"); !ok || failure.Reason != "b" {
		t.Fatalf("scope-b lookup=%+v ok=%v", failure, ok)
	}
	if failure, ok := store.Lookup("scope-a", "other-model"); !ok || failure.Reason != "c" {
		t.Fatalf("other model lookup=%+v ok=%v", failure, ok)
	}
	if _, ok := store.Lookup("scope-a", "missing-model"); ok {
		t.Fatal("missing model should not resolve")
	}
}

func TestFailureStoreIgnoresEmptyKeys(t *testing.T) {
	store := NewFailureStore(DefaultFailureTTL)
	store.Record(" ", Failure{Model: "gpt-test", Reason: "x"})
	store.Record("scope-a", Failure{Model: "  ", Reason: "x"})
	if _, ok := store.Lookup("", "gpt-test"); ok {
		t.Fatal("empty scope should not resolve")
	}
	if _, ok := store.Lookup("scope-a", ""); ok {
		t.Fatal("empty model should not resolve")
	}
}
