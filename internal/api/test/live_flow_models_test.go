package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/liveflow"

	"github.com/gin-gonic/gin"
)

type liveFlowModelsProviderStub struct {
	response liveflow.ModelsResponse
	err      error
	calls    int
	periods  []liveflow.Period
}

func (s *liveFlowModelsProviderStub) Models(_ context.Context, period liveflow.Period) (liveflow.ModelsResponse, error) {
	s.calls++
	s.periods = append(s.periods, period)
	if s.err != nil {
		return liveflow.ModelsResponse{}, s.err
	}
	response := s.response
	response.Period = period
	return response, nil
}

func newLiveFlowModelsRouter(t *testing.T, provider *liveFlowModelsProviderStub) (*gin.Engine, *auth.SessionManager) {
	t.Helper()
	sessions := auth.NewSessionManager(time.Hour)
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{LiveFlowModels: provider})
	return router, sessions
}

func TestLiveFlowModelsAllowsAdmin(t *testing.T) {
	provider := &liveFlowModelsProviderStub{response: liveflow.ModelsResponse{TotalModels: 1}}
	router, sessions := newLiveFlowModelsRouter(t, provider)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models?period=current_week", nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if len(provider.periods) != 1 || provider.periods[0] != liveflow.PeriodCurrentWeek {
		t.Fatalf("periods = %v, want current_week", provider.periods)
	}
}

func TestLiveFlowModelsAllowsAPIKeyViewer(t *testing.T) {
	provider := &liveFlowModelsProviderStub{response: liveflow.ModelsResponse{}}
	router, sessions := newLiveFlowModelsRouter(t, provider)
	token, _, err := sessions.CreateAPIKeyViewer(42)
	if err != nil {
		t.Fatalf("create API key viewer session: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models", nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestLiveFlowModelsRequiresAuthentication(t *testing.T) {
	provider := &liveFlowModelsProviderStub{}
	router, _ := newLiveFlowModelsRouter(t, provider)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
}

func TestLiveFlowModelsRejectsUnknownPeriod(t *testing.T) {
	provider := &liveFlowModelsProviderStub{}
	router, sessions := newLiveFlowModelsRouter(t, provider)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models?period=bogus", nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if provider.calls != 0 {
		t.Fatalf("provider calls = %d, want 0", provider.calls)
	}
}

func TestLiveFlowModelsDefaultsToToday(t *testing.T) {
	provider := &liveFlowModelsProviderStub{}
	router, sessions := newLiveFlowModelsRouter(t, provider)
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models", nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(provider.periods) != 1 || provider.periods[0] != liveflow.PeriodToday {
		t.Fatalf("periods = %v, want today", provider.periods)
	}
}

func TestLiveFlowModelsUnavailableWhenProviderMissing(t *testing.T) {
	sessions := auth.NewSessionManager(time.Hour)
	config := AuthConfig{Enabled: true, LoginPassword: "secret", SessionTTL: time.Hour}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "")
	token, _, err := sessions.Create()
	if err != nil {
		t.Fatalf("create admin session: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/live-flow/models", nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
