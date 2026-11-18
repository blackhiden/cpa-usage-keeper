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
	"cpa-usage-keeper/internal/modelranking"
	modelrankinghttpapi "cpa-usage-keeper/internal/modelranking/httpapi"
)

type modelRankingProviderStub struct {
	calls int
	board modelranking.Response
	err   error
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

func newModelRankingRouter(t *testing.T, provider modelrankinghttpapi.Provider) (string, string, http.Handler) {
	t.Helper()
	sessions := auth.NewSessionManager(time.Hour)
	adminToken, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}
	viewerToken, _, err := sessions.CreateAPIKeyViewer(42)
	if err != nil {
		t.Fatalf("create viewer session: %v", err)
	}
	config := AuthConfig{
		Enabled:       true,
		LoginPassword: "secret",
		SessionTTL:    time.Hour,
	}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		ModelRanking: provider,
	})
	return adminToken, viewerToken, router
}

func modelRankingRequest(target, token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, target, nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	return request
}

func TestModelRankingRouteIsAdminOnly(t *testing.T) {
	stub := &modelRankingProviderStub{board: modelranking.Response{
		PeriodKey: "2026-11-16",
		Entries: []modelranking.Entry{{
			Rank: 1, Model: "gpt-5", Providers: []string{"openai", "azure"}, TotalTokens: 100, Value: 100,
		}},
	}}
	adminToken, viewerToken, router := newModelRankingRouter(t, stub)
	target := "/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens"

	viewer := httptest.NewRecorder()
	router.ServeHTTP(viewer, modelRankingRequest(target, viewerToken))
	if viewer.Code != http.StatusForbidden || stub.calls != 0 {
		t.Fatalf("viewer reached model ranking: status=%d body=%s calls=%d", viewer.Code, viewer.Body.String(), stub.calls)
	}

	admin := httptest.NewRecorder()
	router.ServeHTTP(admin, modelRankingRequest(target, adminToken))
	if admin.Code != http.StatusOK || stub.calls != 1 {
		t.Fatalf("admin could not reach model ranking: status=%d body=%s calls=%d", admin.Code, admin.Body.String(), stub.calls)
	}
	if got := admin.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
	body := admin.Body.String()
	for _, want := range []string{`"period":"today"`, `"metric":"total_tokens"`, `"model":"gpt-5"`, `"openai"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("response missing %s: %s", want, body)
		}
	}
}

func TestModelRankingRouteValidatesQueryParameters(t *testing.T) {
	stub := &modelRankingProviderStub{}
	adminToken, _, router := newModelRankingRouter(t, stub)
	for _, target := range []string{
		"/api/v1/model-ranking/leaderboards",
		"/api/v1/model-ranking/leaderboards?period=today",
		"/api/v1/model-ranking/leaderboards?metric=total_tokens",
		"/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens&extra=1",
		"/api/v1/model-ranking/leaderboards?period=today&period=yesterday&metric=total_tokens",
		"/api/v1/model-ranking/leaderboards?period=bogus&metric=total_tokens",
		"/api/v1/model-ranking/leaderboards?period=today&metric=overall",
		"/api/v1/model-ranking/leaderboards?period=today&metric=bogus",
	} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, modelRankingRequest(target, adminToken))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d, want 400 (body=%s)", target, response.Code, response.Body.String())
		}
		if !strings.Contains(response.Body.String(), "invalid_model_ranking_selection") {
			t.Fatalf("%s error body = %s, want invalid_model_ranking_selection", target, response.Body.String())
		}
	}
	if stub.calls != 0 {
		t.Fatalf("invalid requests reached the provider: calls=%d", stub.calls)
	}
}

func TestModelRankingRouteReturnsServiceUnavailableWhenUnconfigured(t *testing.T) {
	adminToken, _, router := newModelRankingRouter(t, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, modelRankingRequest("/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens", adminToken))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 (body=%s)", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "model_ranking_unavailable") {
		t.Fatalf("error body = %s, want model_ranking_unavailable", response.Body.String())
	}
}

func TestModelRankingRouteMapsProviderErrors(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{"invalid selection", modelranking.ErrInvalidSelection, http.StatusBadRequest, "invalid_model_ranking_selection"},
		{"aggregation failure", errors.New("boom"), http.StatusInternalServerError, "model_ranking_failed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			stub := &modelRankingProviderStub{err: tt.err}
			adminToken, _, router := newModelRankingRouter(t, stub)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, modelRankingRequest("/api/v1/model-ranking/leaderboards?period=today&metric=total_tokens", adminToken))
			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body=%s)", response.Code, tt.wantStatus, response.Body.String())
			}
			if !strings.Contains(response.Body.String(), tt.wantBody) {
				t.Fatalf("error body = %s, want %s", response.Body.String(), tt.wantBody)
			}
		})
	}
}