package test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/service"
)

type providerRouteStub struct {
	service.UsageIdentityProvider
	providers        []service.ProviderAggregate
	keysByProvider   map[string]service.ListProviderKeysResponse
	providersRequest service.ListProvidersRequest
	keysRequest      service.ListProviderKeysRequest
}

func (s *providerRouteStub) ListProviders(_ context.Context, request service.ListProvidersRequest) ([]service.ProviderAggregate, error) {
	s.providersRequest = request
	return s.providers, nil
}

func (s *providerRouteStub) ListProviderKeys(_ context.Context, request service.ListProviderKeysRequest) (service.ListProviderKeysResponse, error) {
	s.keysRequest = request
	if result, ok := s.keysByProvider[request.Provider]; ok {
		return result, nil
	}
	return service.ListProviderKeysResponse{Provider: request.Provider, Page: request.Page, PageSize: request.PageSize}, nil
}

func TestProviderRoutesAdminReceivesUnmaskedResultsAndSort(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}
	stub := &providerRouteStub{
		providers: []service.ProviderAggregate{{Provider: "OpenAI", ProviderType: "openai", KeyCount: 2, TotalRequests: 10, SuccessCount: 9, SuccessRate: .9}},
		keysByProvider: map[string]service.ListProviderKeysResponse{
			"OpenAI": {Provider: "OpenAI", TotalCount: 1, Page: 1, PageSize: 20, TotalPages: 1, Keys: []entities.UsageIdentity{{ID: 1, Identity: "auth-index", AuthType: entities.UsageIdentityAuthTypeAIProvider, Provider: "OpenAI", Prefix: "team-prefix", BaseURL: "https://provider.example"}}},
		},
	}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{UsageIdentity: stub})
	cookie := &http.Cookie{Name: standardSessionCookieName, Value: token}

	providers := serveAPIGet(router, "/api/v1/usage/providers?sort=success_rate", cookie)
	if providers.Code != http.StatusOK || stub.providersRequest.Sort != "success_rate" || stub.providersRequest.CPAAPIKeyID != 0 || !contains(providers.Body.String(), `"provider_type":"openai"`) {
		t.Fatalf("admin provider response=%d request=%+v body=%s", providers.Code, stub.providersRequest, providers.Body.String())
	}
	keys := serveAPIGet(router, "/api/v1/usage/providers/OpenAI/keys?sort=success_rate&page=1&page_size=20", cookie)
	if keys.Code != http.StatusOK || stub.keysRequest.CPAAPIKeyID != 0 {
		t.Fatalf("admin key response=%d request=%+v body=%s", keys.Code, stub.keysRequest, keys.Body.String())
	}
	var payload struct {
		Keys []struct{ Prefix, BaseURL string } `json:"keys"`
	}
	if err := json.Unmarshal(keys.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode admin key response: %v", err)
	}
	if len(payload.Keys) != 1 || payload.Keys[0].Prefix != "team-prefix" || payload.Keys[0].BaseURL != "https://provider.example" {
		t.Fatalf("admin credentials unexpectedly masked: %s", keys.Body.String())
	}
}

func TestProviderRoutesScopeViewerAndMaskCredentials(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.CreateAPIKeyViewerWithSource(42, auth.SessionSourceStandard)
	if err != nil {
		t.Fatalf("create viewer session: %v", err)
	}
	stub := &providerRouteStub{
		providers: []service.ProviderAggregate{{Provider: "OpenAI", ProviderType: "openai", KeyCount: 1, TotalRequests: 2, SuccessCount: 1, SuccessRate: .5}},
		keysByProvider: map[string]service.ListProviderKeysResponse{
			"OpenAI": {Provider: "OpenAI", TotalCount: 1, Page: 1, PageSize: 20, TotalPages: 1, Keys: []entities.UsageIdentity{{ID: 1, Identity: "auth-index", AuthType: entities.UsageIdentityAuthTypeAIProvider, Provider: "OpenAI", Prefix: "team-prefix", BaseURL: "https://provider.example"}}},
		},
	}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		UsageIdentity: stub,
		CPAAPIKeys:    &authCPAAPIKeyStub{row: entities.CPAAPIKey{ID: 42, APIKey: "viewer-key", DisplayKey: "viewer-key"}},
	})
	cookie := &http.Cookie{Name: standardSessionCookieName, Value: token}

	providers := serveAPIGet(router, "/api/v1/usage/providers?sort=success_rate", cookie)
	if providers.Code != http.StatusOK || stub.providersRequest.CPAAPIKeyID != 42 {
		t.Fatalf("viewer provider response=%d request=%+v body=%s", providers.Code, stub.providersRequest, providers.Body.String())
	}
	keys := serveAPIGet(router, "/api/v1/usage/providers/OpenAI/keys", cookie)
	if keys.Code != http.StatusOK || stub.keysRequest.CPAAPIKeyID != 42 {
		t.Fatalf("viewer key response=%d request=%+v body=%s", keys.Code, stub.keysRequest, keys.Body.String())
	}
	if !contains(keys.Body.String(), `"prefix":"***"`) || !contains(keys.Body.String(), `"base_url":"***"`) || contains(keys.Body.String(), "team-prefix") || contains(keys.Body.String(), "provider.example") {
		t.Fatalf("viewer credential masking failed: %s", keys.Body.String())
	}

	missing := serveAPIGet(router, "/api/v1/usage/providers/Anthropic/keys", cookie)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected inaccessible provider to return 404, got %d body=%s", missing.Code, missing.Body.String())
	}
}

func TestProviderRoutesRequireAuthentication(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}, nil, "", OptionalProviders{UsageIdentity: &providerRouteStub{}})
	for _, target := range []string{"/api/v1/usage/providers", "/api/v1/usage/providers/OpenAI/keys"} {
		response := serveAPIGet(router, target)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s status=%d, want 401: %s", target, response.Code, response.Body.String())
		}
	}
}
