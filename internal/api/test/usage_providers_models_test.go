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

type providerModelsStub struct {
	service.UsageIdentityProvider
	models  map[string]service.ListProviderModelsResponse
	request service.ListProviderModelsRequest
}

func (s *providerModelsStub) ListProviderModels(_ context.Context, request service.ListProviderModelsRequest) (service.ListProviderModelsResponse, error) {
	s.request = request
	if result, ok := s.models[request.Provider]; ok {
		return result, nil
	}
	return service.ListProviderModelsResponse{Provider: request.Provider, Page: request.Page, PageSize: request.PageSize}, nil
}

func TestProviderModelsRouteAdminSeesAllModels(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	lastUsed := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)
	stub := &providerModelsStub{
		models: map[string]service.ListProviderModelsResponse{
			"OpenAI": {
				Provider: "OpenAI", TotalCount: 2, Page: 1, PageSize: 20, TotalPages: 1,
				Models: []service.ProviderModel{
					{Model: "gpt-4", TotalRequests: 100, SuccessCount: 95, FailureCount: 5, SuccessRate: 0.95, InputTokens: 1000, CacheReadTokens: 200, TotalTokens: 1500, LastUsedAt: &lastUsed},
					{Model: "gpt-3.5-turbo", TotalRequests: 50, SuccessCount: 48, FailureCount: 2, SuccessRate: 0.96, InputTokens: 500, CacheReadTokens: 50, TotalTokens: 700, LastUsedAt: &lastUsed},
				},
			},
		},
	}

	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{UsageIdentity: stub})
	cookie := &http.Cookie{Name: standardSessionCookieName, Value: token}

	response := serveAPIGet(router, "/api/v1/usage/providers/OpenAI/models?sort=success_rate&page=1&page_size=20", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	if stub.request.Sort != "success_rate" || stub.request.Page != 1 || stub.request.PageSize != 20 || stub.request.CPAAPIKeyID != 0 {
		t.Fatalf("unexpected request params: %+v", stub.request)
	}

	var payload struct {
		Provider   string `json:"provider"`
		TotalCount int64  `json:"total_count"`
		Models     []struct {
			Model           string  `json:"model"`
			TotalRequests   int64   `json:"total_requests"`
			SuccessRate     float64 `json:"success_rate"`
			InputTokens     int64   `json:"input_tokens"`
			CacheReadTokens int64   `json:"cache_read_tokens"`
		} `json:"models"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Provider != "OpenAI" || payload.TotalCount != 2 || len(payload.Models) != 2 {
		t.Fatalf("unexpected response structure: %+v", payload)
	}
	if payload.Models[0].Model != "gpt-4" || payload.Models[0].TotalRequests != 100 || payload.Models[0].SuccessRate != 0.95 {
		t.Fatalf("unexpected first model: %+v", payload.Models[0])
	}
}

func TestProviderModelsRouteViewerSeesScoped(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.CreateAPIKeyViewerWithSource(42, auth.SessionSourceStandard)
	if err != nil {
		t.Fatalf("create viewer session: %v", err)
	}

	stub := &providerModelsStub{
		models: map[string]service.ListProviderModelsResponse{
			"Claude": {
				Provider: "Claude", TotalCount: 1, Page: 1, PageSize: 20, TotalPages: 1,
				Models: []service.ProviderModel{
					{Model: "claude-3-opus", TotalRequests: 20, SuccessCount: 19, FailureCount: 1, SuccessRate: 0.95, InputTokens: 200, CacheReadTokens: 50, TotalTokens: 300},
				},
			},
		},
	}

	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		UsageIdentity: stub,
		CPAAPIKeys:    &authCPAAPIKeyStub{row: entities.CPAAPIKey{ID: 42, APIKey: "viewer-key", DisplayKey: "viewer-key"}},
	})
	cookie := &http.Cookie{Name: standardSessionCookieName, Value: token}

	response := serveAPIGet(router, "/api/v1/usage/providers/Claude/models", cookie)
	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}

	if stub.request.CPAAPIKeyID != 42 {
		t.Fatalf("expected viewer to be scoped to API key 42, got %d", stub.request.CPAAPIKeyID)
	}
}

func TestProviderModelsRouteReturns404ForNoModels(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	stub := &providerModelsStub{models: map[string]service.ListProviderModelsResponse{}}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{UsageIdentity: stub})
	cookie := &http.Cookie{Name: standardSessionCookieName, Value: token}

	response := serveAPIGet(router, "/api/v1/usage/providers/NonExistent/models", cookie)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for provider with no models, got %d: %s", response.Code, response.Body.String())
	}
}

func TestProviderModelsRouteRequiresAuthentication(t *testing.T) {
	stub := &providerModelsStub{models: map[string]service.ListProviderModelsResponse{}}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, nil, "", OptionalProviders{UsageIdentity: stub})

	response := serveAPIGet(router, "/api/v1/usage/providers/OpenAI/models")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
}
