package test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/service"
)

// ownedUsageEventsStub reports ownership for the viewer's own events so the
// request-log route proceeds past the ownership check.
type ownedUsageEventsStub struct {
	*usageEventsStub
}

func (s *ownedUsageEventsStub) VerifyUsageEventOwnership(context.Context, int64, string) (bool, error) {
	return true, nil
}

// TestAPIKeyViewerRequestLogFullContentPersistence guards the fork contract for owning viewers.
func TestAPIKeyViewerRequestLogFullContentPersistence(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.CreateAPIKeyViewerWithSource(42, auth.SessionSourceStandard)
	if err != nil {
		t.Fatalf("create API key viewer session: %v", err)
	}

	expectedSections := []service.RequestLogSection{
		{Title: "REQUEST INFO", Content: "URL: /v1/chat/completions\nMethod: POST"},
		{Title: "HEADERS", Content: "Authorization: Bearer test-token"},
		{Title: "REQUEST BODY", Content: `{"model":"gpt-5.4-mini","messages":[{"role":"user","content":"hello"}]}`},
		{Title: "API REQUEST", Content: `{"model":"gpt-5.4-mini"}`},
		{Title: "API RESPONSE", Content: `{"choices":[{"message":{"content":"hi"}}]}`},
		{Title: "RESPONSE", Content: `{"choices":[{"message":{"content":"hi"}}]}`},
	}
	provider := &ownedUsageEventsStub{usageEventsStub: &usageEventsStub{}}
	requestLogProvider := &requestLogProviderStub{response: service.RequestLogResponse{
		EventID:     42,
		RequestID:   "req-log-42",
		Filename:    "v1-chat-completions-req-log-42.log",
		Available:   true,
		Previewable: true,
		Sections:    expectedSections,
	}}
	keyProvider := &authCPAAPIKeyStub{row: entities.CPAAPIKey{
		ID: 42, APIKey: "provider-a", DisplayKey: "provider-a",
	}}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, provider, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		RequestLogs: requestLogProvider,
		CPAAPIKeys:  keyProvider,
		Status:      StatusRouteConfig{CPARequestLogAccessEnabled: true},
	})

	resp := serveAPIGet(router, "/api/v1/key-events/42/request-log", &http.Cookie{Name: standardSessionCookieName, Value: token})

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", resp.Code, resp.Body.String())
	}
	var payload struct {
		EventID     string                      `json:"event_id"`
		RequestID   string                      `json:"request_id"`
		Previewable bool                        `json:"previewable"`
		Sections    []service.RequestLogSection `json:"sections"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode request log response: %v", err)
	}
	if payload.EventID != "42" || payload.RequestID != "req-log-42" || !payload.Previewable {
		t.Fatalf("unexpected request log metadata: %+v", payload)
	}
	if !reflect.DeepEqual(payload.Sections, expectedSections) {
		t.Fatalf("request log sections were altered: got %#v want %#v", payload.Sections, expectedSections)
	}
	if requestLogProvider.calls != 1 || requestLogProvider.eventID != 42 {
		t.Fatalf("expected provider call with event id 42, got calls=%d eventID=%d", requestLogProvider.calls, requestLogProvider.eventID)
	}
}

func TestAPIKeyViewerRequestLogRejectsCrossKeyAccess(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.CreateAPIKeyViewerWithSource(42, auth.SessionSourceStandard)
	if err != nil {
		t.Fatalf("create API key viewer session: %v", err)
	}

	// VerifyUsageEventOwnership returns false, so the log must not be fetched.
	provider := &usageEventsStub{}
	requestLogProvider := &requestLogProviderStub{}
	keyProvider := &authCPAAPIKeyStub{row: entities.CPAAPIKey{
		ID: 42, APIKey: "provider-a", DisplayKey: "provider-a",
	}}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, provider, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		RequestLogs: requestLogProvider,
		CPAAPIKeys:  keyProvider,
		Status:      StatusRouteConfig{CPARequestLogAccessEnabled: true},
	})

	resp := serveAPIGet(router, "/api/v1/key-events/42/request-log", &http.Cookie{Name: standardSessionCookieName, Value: token})

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d body=%s", resp.Code, resp.Body.String())
	}
	if requestLogProvider.calls != 0 {
		t.Fatalf("expected no provider call for cross-key access, got %d", requestLogProvider.calls)
	}
}

func TestAPIKeyViewerRequestLogRejectsDisabledAccess(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.CreateAPIKeyViewerWithSource(42, auth.SessionSourceStandard)
	if err != nil {
		t.Fatalf("create API key viewer session: %v", err)
	}

	provider := &ownedUsageEventsStub{usageEventsStub: &usageEventsStub{}}
	requestLogProvider := &requestLogProviderStub{}
	keyProvider := &authCPAAPIKeyStub{row: entities.CPAAPIKey{
		ID: 42, APIKey: "provider-a", DisplayKey: "provider-a",
	}}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, provider, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		RequestLogs: requestLogProvider,
		CPAAPIKeys:  keyProvider,
	})

	resp := serveAPIGet(router, "/api/v1/key-events/42/request-log", &http.Cookie{Name: standardSessionCookieName, Value: token})

	if resp.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", resp.Code, resp.Body.String())
	}
	if requestLogProvider.calls != 0 {
		t.Fatalf("expected no provider call when access is disabled, got %d", requestLogProvider.calls)
	}
}
