package test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/modelranking"
	modelrankinghttpapi "cpa-usage-keeper/internal/modelranking/httpapi"
)

type modelRankingProviderStub struct {
	calls       int
	lastOptions modelranking.LeaderboardOptions
	board       modelranking.Response
	err         error
}

func (s *modelRankingProviderStub) Leaderboard(_ context.Context, period modelranking.Period, metric modelranking.Metric) (modelranking.Response, error) {
	s.calls++
	if s.err != nil {
		return modelranking.Response{}, s.err
	}
	board := s.board
	board.Period = period
	board.Metric = metric
	if board.Entries == nil {
		board.Entries = []modelranking.Entry{}
	}
	return board, nil
}

func (s *modelRankingProviderStub) LeaderboardWithOptions(ctx context.Context, period modelranking.Period, metric modelranking.Metric, options modelranking.LeaderboardOptions) (modelranking.Response, error) {
	s.lastOptions = options
	return s.Leaderboard(ctx, period, metric)
}

func newModelRankingRouter(t *testing.T, provider modelrankinghttpapi.ScopedProvider) (string, string, http.Handler) {
	t.Helper()
	sessions := auth.NewSessionManager(time.Hour)
	adminToken, _, err := sessions.Create()
	if err != nil {
		t.Fatal(err)
	}
	viewerToken, _, err := sessions.CreateAPIKeyViewer(42)
	if err != nil {
		t.Fatal(err)
	}
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		ModelRanking: provider,
		CPAAPIKeys:   &keyViewerRankingKeyStub{row: entities.CPAAPIKey{ID: 42, APIKey: "sk-viewer123456", KeyAlias: "Viewer"}},
	})
	return adminToken, viewerToken, router
}

func modelRankingRequest(target, token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	return request
}

func TestModelRankingRouteIsAdminOnly(t *testing.T) {
	stub := &modelRankingProviderStub{board: modelranking.Response{PeriodKey: "2026-11-16", Entries: []modelranking.Entry{{Rank: 1, Model: "gpt-5", Providers: []string{"openai", "azure"}, TotalTokens: 100, Value: 100}}}}
	adminToken, viewerToken, router := newModelRankingRouter(t, stub)
	target := "/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens"
	viewer := httptest.NewRecorder()
	router.ServeHTTP(viewer, modelRankingRequest(target, viewerToken))
	if viewer.Code != http.StatusForbidden || stub.calls != 0 {
		t.Fatalf("viewer status=%d body=%s calls=%d", viewer.Code, viewer.Body.String(), stub.calls)
	}
	admin := httptest.NewRecorder()
	router.ServeHTTP(admin, modelRankingRequest(target, adminToken))
	if admin.Code != http.StatusOK || stub.calls != 1 {
		t.Fatalf("admin status=%d body=%s", admin.Code, admin.Body.String())
	}
}

func TestKeyModelRankingRouteUsesSessionScope(t *testing.T) {
	stub := &modelRankingProviderStub{}
	_, viewerToken, router := newModelRankingRouter(t, stub)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, modelRankingRequest("/api/v1/key-model-ranking/leaderboards?period=today&metric=total_tokens", viewerToken))
	if response.Code != http.StatusOK {
		t.Fatalf("viewer model ranking status=%d body=%s", response.Code, response.Body.String())
	}
	if stub.lastOptions.ScopeKey != "sk-viewer123456" {
		t.Fatalf("provider scope=%q", stub.lastOptions.ScopeKey)
	}
	response = httptest.NewRecorder()
	router.ServeHTTP(response, modelRankingRequest("/api/v1/key-model-ranking/leaderboards?period=today&metric=total_tokens&scope_key=other", viewerToken))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("scope override status=%d", response.Code)
	}
}

func TestKeyModelRankingRouteScopesFromAuthenticatedViewer(t *testing.T) {
	stub := &modelRankingProviderStub{}
	_, viewerToken, router := newModelRankingRouter(t, stub)
	request := httptest.NewRecorder()
	router.ServeHTTP(request, modelRankingRequest("/api/v1/key-model-ranking/leaderboards?period=today&metric=total_tokens", viewerToken))
	if request.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", request.Code, request.Body.String())
	}
	if stub.lastOptions.ScopeKey != "sk-viewer123456" {
		t.Fatalf("scope=%q", stub.lastOptions.ScopeKey)
	}
	global := httptest.NewRecorder()
	router.ServeHTTP(global, modelRankingRequest("/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens", viewerToken))
	if global.Code != http.StatusForbidden {
		t.Fatalf("global viewer route status=%d", global.Code)
	}
	attack := httptest.NewRecorder()
	router.ServeHTTP(attack, modelRankingRequest("/api/v1/key-model-ranking/leaderboards?period=today&metric=total_tokens&scope_key=other", viewerToken))
	if attack.Code != http.StatusBadRequest {
		t.Fatalf("client scope override status=%d", attack.Code)
	}
}

func TestModelRankingRouteValidatesQueryParameters(t *testing.T) {
	stub := &modelRankingProviderStub{}
	adminToken, _, router := newModelRankingRouter(t, stub)
	for _, target := range []string{"/api/v1/model-ranking/leaderboards", "/api/v1/model-ranking/leaderboards?period=today", "/api/v1/model-ranking/leaderboards?metric=total_tokens", "/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens&extra=1", "/api/v1/model-ranking/leaderboards?period=bad&metric=total_tokens", "/api/v1/model-ranking/leaderboards?period=today&metric=bad"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, modelRankingRequest(target, adminToken))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", target, response.Code)
		}
	}
}

func TestModelRankingRouteHandlesUnavailableAndFailure(t *testing.T) {
	adminToken, _, router := newModelRankingRouter(t, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, modelRankingRequest("/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens", adminToken))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
	stub := &modelRankingProviderStub{err: errors.New("failed")}
	adminToken, _, router = newModelRankingRouter(t, stub)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, modelRankingRequest("/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens", adminToken))
	if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "model_ranking_failed") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
