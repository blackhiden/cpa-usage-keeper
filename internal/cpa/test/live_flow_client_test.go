package cpa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cpa-usage-keeper/internal/cpa"
	"github.com/gorilla/websocket"
)

func TestDialLiveFlowUsesCPAWebSocketPathAndOptionalClientKey(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live-flow/ws" {
			t.Errorf("path = %q, want /live-flow/ws", r.URL.Path)
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer live-flow-client-key" {
			t.Errorf("Authorization = %q, want live flow client key", authorization)
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade upstream live flow: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"id":"flow-1","ts":1,"method":"POST","path":"/v1/responses","model":"gpt-5","status":200,"latency_ms":8}`)); err != nil {
			t.Errorf("write upstream live flow frame: %v", err)
		}
	}))
	defer server.Close()

	client := cpa.NewClient(server.URL, "management-key", time.Second, false)
	client.SetLiveFlowAPIKey(" live-flow-client-key ")
	conn, _, err := client.DialLiveFlow(context.Background())
	if err != nil {
		t.Fatalf("DialLiveFlow returned error: %v", err)
	}
	defer func() { _ = conn.Close() }()

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read live flow frame: %v", err)
	}
	if messageType != websocket.TextMessage || string(payload) != `{"id":"flow-1","ts":1,"method":"POST","path":"/v1/responses","model":"gpt-5","status":200,"latency_ms":8}` {
		t.Fatalf("unexpected live flow frame type=%d payload=%s", messageType, payload)
	}
}
