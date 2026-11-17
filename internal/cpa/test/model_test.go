package cpa

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cpa-usage-keeper/internal/cpa"
)

func TestTestModelRegularRequestUsesViewerCredentialAndBoundedProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer viewer-secret" {
			t.Fatalf("unexpected authorization %q", r.Header.Get("Authorization"))
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["model"] != "gpt-test" || payload["max_tokens"] != float64(16) || payload["stream"] != false {
			t.Fatalf("unexpected payload %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	defer server.Close()
	client := cpa.NewClient(server.URL, "management-secret", time.Second, false)
	result, err := client.TestModel(context.Background(), "viewer-secret", "gpt-test", cpa.ModelTestModeRegular, time.Second)
	if err != nil || result.StatusCode != http.StatusOK {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestTestModelStreamingRequiresDone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client := cpa.NewClient(server.URL, "management-secret", time.Second, false)
	_, err := client.TestModel(context.Background(), "viewer-secret", "gpt-test", cpa.ModelTestModeStreaming, time.Second)
	if err != nil {
		t.Fatalf("streaming test failed: %v", err)
	}
}

func TestTestModelRejectsOversizedRegularResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("x", 70*1024))
	}))
	defer server.Close()
	client := cpa.NewClient(server.URL, "management-secret", time.Second, false)
	_, err := client.TestModel(context.Background(), "viewer-secret", "gpt-test", cpa.ModelTestModeRegular, time.Second)
	if err == nil {
		t.Fatal("expected oversized response error")
	}
}
