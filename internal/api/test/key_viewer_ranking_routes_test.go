package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/api"
	"cpa-usage-keeper/internal/auth"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/service"
)

type keyViewerRankingKeyStub struct {
	row entities.CPAAPIKey
}

func (s *keyViewerRankingKeyStub) ListCPAAPIKeys(context.Context) ([]entities.CPAAPIKey, error) {
	return []entities.CPAAPIKey{s.row}, nil
}

func (s *keyViewerRankingKeyStub) FindActiveCPAAPIKeyByValue(context.Context, string) (entities.CPAAPIKey, error) {
	return s.row, nil
}

func (s *keyViewerRankingKeyStub) FindActiveCPAAPIKeyByID(_ context.Context, id int64) (entities.CPAAPIKey, error) {
	if id != s.row.ID {
		return entities.CPAAPIKey{}, service.ErrInvalidID
	}
	return s.row, nil
}

func (s *keyViewerRankingKeyStub) UpdateCPAAPIKeyAlias(context.Context, int64, string) (entities.CPAAPIKey, error) {
	return s.row, nil
}

func newKeyViewerRankingRouter(t *testing.T, localEnabled bool) (*auth.SessionManager, string, *rankingRouteProviderStub, *adminLocalRankingProviderStub, http.Handler) {
	t.Helper()
	sessions := auth.NewSessionManager(time.Hour)
	viewerToken, _, err := sessions.CreateAPIKeyViewer(42)
	if err != nil {
		t.Fatalf("create viewer session: %v", err)
	}
	community := &rankingRouteProviderStub{}
	local := &adminLocalRankingProviderStub{}
	keyProvider := &keyViewerRankingKeyStub{row: entities.CPAAPIKey{ID: 42, APIKey: "sk-viewer123456", KeyAlias: "Viewer"}}
	config := AuthConfig{
		Enabled:                         true,
		LoginPassword:                   "secret",
		SessionTTL:                      time.Hour,
		APIKeyViewerLocalRankingEnabled: localEnabled,
		LocalRankingAvailable:           true, // Provider is available in tests
	}
	router := NewRouter(nil, nil, nil, nil, config, NewAuthHandler(config, sessions), "", OptionalProviders{
		CPAAPIKeys:   keyProvider,
		Ranking:      community,
		LocalRanking: local,
	})
	return sessions, viewerToken, community, local, router
}

func viewerRankingRequest(method, target, token string) *http.Request {
	request := httptest.NewRequest(method, target, nil)
	request.AddCookie(&http.Cookie{Name: "cpa_usage_keeper_session", Value: token})
	return request
}

func TestKeyViewerCommunityRankingRouteIsReadOnly(t *testing.T) {
	// Community ranking is disabled - routes are no longer registered.
	// This test is kept to document the intentional removal.
	t.Skip("Community ranking is disabled - routes not registered")
}

func TestKeyViewerLocalRankingRouteIsAlwaysAvailableAndReadOnly(t *testing.T) {
	// Local ranking is now always available for viewers when the provider is available.
	// The deprecated config flag is ignored for route registration.
	for _, tc := range []struct {
		name                  string
		configFlag            bool // The deprecated flag value (now ignored for route registration)
		wantStatus, wantCalls int
	}{
		{"always available even with flag false", false, http.StatusOK, 1},
		{"always available with flag true", true, http.StatusOK, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, viewerToken, _, local, router := newKeyViewerRankingRouter(t, tc.configFlag)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, viewerRankingRequest(http.MethodGet, "/api/v1/key-ranking/local/leaderboards?period=today&metric=overall", viewerToken))
			if response.Code != tc.wantStatus || local.calls != tc.wantCalls {
				t.Fatalf("local ranking status=%d body=%s calls=%d; want %d/%d", response.Code, response.Body.String(), local.calls, tc.wantStatus, tc.wantCalls)
			}
			mutation := httptest.NewRecorder()
			router.ServeHTTP(mutation, viewerRankingRequest(http.MethodPatch, "/api/v1/key-ranking/local/profiles/42", viewerToken))
			if mutation.Code != http.StatusNotFound || local.calls != tc.wantCalls {
				t.Fatalf("viewer local profile mutation was exposed: status=%d body=%s calls=%d", mutation.Code, mutation.Body.String(), local.calls)
			}
			// The session API now reports whether the local ranking provider is available.
			// Since this test passes a non-nil LocalRanking provider, it should always be true.
			session := httptest.NewRecorder()
			router.ServeHTTP(session, viewerRankingRequest(http.MethodGet, "/api/v1/auth/session", viewerToken))
			if session.Code != http.StatusOK {
				t.Fatalf("viewer session request failed: status=%d body=%s", session.Code, session.Body.String())
			}
			if !strings.Contains(session.Body.String(), `"local_ranking_enabled":true`) {
				t.Fatalf("viewer session missing expected local_ranking_enabled:true in response: %s", session.Body.String())
			}
		})
	}
}
