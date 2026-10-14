package test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/cpa"
	"github.com/gorilla/websocket"
)

func TestLiveFlowRelayForwardsCPAEventsToAdminClients(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/live-flow/ws" {
			t.Errorf("upstream path = %q, want /live-flow/ws", r.URL.Path)
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
	defer upstream.Close()

	client := cpa.NewClient(upstream.URL, "management-key", time.Second, false)
	sessions := auth.NewSessionManager(time.Hour)
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{LiveFlowClient: client})
	keeper := httptest.NewServer(router)
	defer keeper.Close()
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}
	headers := http.Header{}
	headers.Set("Cookie", "cpa_usage_keeper_session="+token)

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(keeper.URL, "http")+"/api/v1/live-flow/ws", headers)
	if err != nil {
		t.Fatalf("dial keeper live flow: %v response=%v", err, response)
	}
	defer func() { _ = conn.Close() }()

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read relayed live flow event: %v", err)
	}
	if messageType != websocket.TextMessage || string(payload) != `{"id":"flow-1","ts":1,"method":"POST","path":"/v1/responses","model":"gpt-5","status":200,"latency_ms":8}` {
		t.Fatalf("unexpected relayed event type=%d payload=%s", messageType, payload)
	}
}

func TestLiveFlowRelayReportsAnUpstreamHandshakeFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	client := cpa.NewClient(upstream.URL, "management-key", time.Second, false)
	router := NewRouter(nil, nil, nil, nil, AuthConfig{Enabled: false}, nil, "", OptionalProviders{LiveFlowClient: client})
	keeper := httptest.NewServer(router)
	defer keeper.Close()

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(keeper.URL, "http")+"/api/v1/live-flow/ws", nil)
	if err != nil {
		t.Fatalf("dial keeper live flow: %v response=%v", err, response)
	}
	defer func() { _ = conn.Close() }()

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read relay error frame: %v", err)
	}
	if messageType != websocket.TextMessage || !strings.Contains(string(payload), `"code":"upstream_feature_disabled"`) || !strings.Contains(string(payload), `"retryable":false`) {
		t.Fatalf("unexpected relay error frame type=%d payload=%s", messageType, payload)
	}
}

func TestLiveFlowRelayRequiresAnAdminSession(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "")
	keeper := httptest.NewServer(router)
	defer keeper.Close()

	_, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(keeper.URL, "http")+"/api/v1/live-flow/ws", nil)
	if err == nil {
		t.Fatal("unauthenticated live flow websocket unexpectedly connected")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("live flow response status = %v, want %d", response, http.StatusUnauthorized)
	}
}

func TestLiveFlowRelayAcceptsAPIKeyViewerSession(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade upstream live flow: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
		if err := conn.WriteMessage(websocket.TextMessage, []byte(`{"id":"viewer-flow","status":200}`)); err != nil {
			t.Errorf("write upstream live flow frame: %v", err)
		}
	}))
	defer upstream.Close()

	client := cpa.NewClient(upstream.URL, "management-key", time.Second, false)
	sessions := auth.NewSessionManager(time.Hour)
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{LiveFlowClient: client})
	keeper := httptest.NewServer(router)
	defer keeper.Close()

	token, _, err := sessions.CreateAPIKeyViewer(42)
	if err != nil {
		t.Fatalf("create API key viewer session: %v", err)
	}
	headers := http.Header{}
	headers.Set("Cookie", "cpa_usage_keeper_session="+token)

	conn, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(keeper.URL, "http")+"/api/v1/live-flow/ws", headers)
	if err != nil {
		t.Fatalf("API key viewer dial keeper live flow: %v response=%v", err, response)
	}
	defer func() { _ = conn.Close() }()

	messageType, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read relayed live flow event: %v", err)
	}
	if messageType != websocket.TextMessage || !strings.Contains(string(payload), `"id":"viewer-flow"`) {
		t.Fatalf("unexpected relayed event type=%d payload=%s", messageType, payload)
	}
}

