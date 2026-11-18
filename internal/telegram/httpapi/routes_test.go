package telegramhttpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cpa-usage-keeper/internal/telegram"
	"github.com/gin-gonic/gin"
)

type notifierStub struct {
	calls   int
	message telegram.Message
	err     error
}

func (s *notifierStub) SendMessage(_ context.Context, _ telegram.Target, message telegram.Message) error {
	s.calls++
	s.message = message
	return s.err
}

type settingsStub struct {
	settings telegram.Settings
}

func (s settingsStub) TelegramSettings(context.Context) (telegram.Settings, error) {
	return s.settings, nil
}

func testRouter(service *telegram.Service, resolve ReporterResolver) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterRoutes(router, service, resolve)
	return router
}

func postNotify(router *gin.Engine, body string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/model-test/notify", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	return recorder
}

func configuredService(notifier *notifierStub) *telegram.Service {
	return telegram.NewService(notifier, settingsStub{settings: telegram.Settings{
		BotToken: "123456789:AAHtest-token-value", ChatID: "-1001234567890", Enabled: true, Configured: true,
	}})
}

// resolverFor returns a fixed server-side reporter and scope.
func resolverFor(reporter, scope string) ReporterResolver {
	return func(*gin.Context) (string, string, bool) { return reporter, scope, true }
}

func recordFailure(service *telegram.Service, scope string) {
	service.RecordFailure(scope, telegram.Failure{
		Model: "gpt-test", Mode: "streaming", StatusCode: 502, ErrorCode: "test_failed", Reason: "upstream timeout",
	})
}

func TestNotifyRouteResolvesReporterAndFailureServerSide(t *testing.T) {
	notifier := &notifierStub{}
	service := configuredService(notifier)
	recordFailure(service, "scope-team")
	router := testRouter(service, resolverFor("Team Key", "scope-team"))
	// Forged reporter/reason fields in the body must be ignored entirely.
	recorder := postNotify(router, `{"model":"gpt-test","mode":"streaming","reason":"forged","reporter":"Forged","status_code":401}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if notifier.calls != 1 || notifier.message.Reporter != "Team Key" {
		t.Fatalf("calls=%d message=%+v", notifier.calls, notifier.message)
	}
	if notifier.message.Reason != "upstream timeout" || notifier.message.StatusCode != 502 {
		t.Fatalf("client fields influenced the message: %+v", notifier.message)
	}
}

func TestNotifyRouteRateLimited(t *testing.T) {
	notifier := &notifierStub{}
	service := configuredService(notifier)
	recordFailure(service, "scope-team")
	router := testRouter(service, resolverFor("Team Key", "scope-team"))
	body := `{"model":"gpt-test","mode":"regular"}`
	if recorder := postNotify(router, body); recorder.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	recorder := postNotify(router, body)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), telegram.CodeRateLimited) {
		t.Fatalf("missing stable code: %s", recorder.Body.String())
	}
	if recorder.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After header")
	}
	if notifier.calls != 1 {
		t.Fatalf("unexpected send count: %d", notifier.calls)
	}
}

func TestNotifyRouteDisabledAndMuted(t *testing.T) {
	notifier := &notifierStub{}
	settings := settingsStub{settings: telegram.Settings{
		BotToken: "123456789:AAHtest-token-value", ChatID: "-1001234567890", Enabled: false, Configured: true,
	}}
	router := testRouter(telegram.NewService(notifier, settings), resolverFor("Team Key", "scope-team"))
	recorder := postNotify(router, `{"model":"gpt-test"}`)
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), telegram.CodeDisabled) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if notifier.calls != 0 {
		t.Fatalf("unexpected send: %d", notifier.calls)
	}

	settings.settings.Enabled = true
	settings.settings.MutedModels = []string{"gpt-test"}
	router = testRouter(telegram.NewService(notifier, settings), resolverFor("Team Key", "scope-team"))
	recorder = postNotify(router, `{"model":"gpt-test"}`)
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), telegram.CodeModelMuted) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestNotifyRouteSendFailed(t *testing.T) {
	notifier := &notifierStub{err: telegram.ErrSendFailed}
	service := configuredService(notifier)
	recordFailure(service, "scope-team")
	router := testRouter(service, resolverFor("Team Key", "scope-team"))
	recorder := postNotify(router, `{"model":"gpt-test"}`)
	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "123456789:AAHtest-token-value") {
		t.Fatalf("response leaked the token: %s", recorder.Body.String())
	}
}

func TestNotifyRouteRejectsInvalidPayload(t *testing.T) {
	notifier := &notifierStub{}
	router := testRouter(configuredService(notifier), resolverFor("Team Key", "scope-team"))
	for _, body := range []string{`not json`, `{}`, `{"model":"a","mode":"other"}`} {
		recorder := postNotify(router, body)
		if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), telegram.CodeInvalidRequest) {
			t.Fatalf("body=%s status=%d response=%s", body, recorder.Code, recorder.Body.String())
		}
	}
	if notifier.calls != 0 {
		t.Fatalf("unexpected send: %d", notifier.calls)
	}
}

func TestNotifyRouteRequiresResolvedReporter(t *testing.T) {
	notifier := &notifierStub{}
	router := testRouter(configuredService(notifier), func(*gin.Context) (string, string, bool) { return "", "", false })
	recorder := postNotify(router, `{"model":"gpt-test"}`)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if notifier.calls != 0 {
		t.Fatalf("unexpected send: %d", notifier.calls)
	}
}

func TestNotifyRouteUnavailableWithoutService(t *testing.T) {
	router := testRouter(nil, nil)
	recorder := postNotify(router, `{"model":"gpt-test"}`)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestTelegramTestMessageRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	notifier := &notifierStub{}
	router := gin.New()
	RegisterAdminRoutes(router, configuredService(notifier))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/telegram/test-message", nil))
	if recorder.Code != http.StatusOK || notifier.calls != 1 || notifier.message.Kind != telegram.KindTest {
		t.Fatalf("status=%d calls=%d message=%+v", recorder.Code, notifier.calls, notifier.message)
	}
}

func TestTelegramTestMessageRouteUnconfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	notifier := &notifierStub{}
	router := gin.New()
	RegisterAdminRoutes(router, telegram.NewService(notifier, settingsStub{}))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/telegram/test-message", nil))
	if recorder.Code != http.StatusConflict || !strings.Contains(recorder.Body.String(), telegram.CodeNotConfigured) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if notifier.calls != 0 {
		t.Fatalf("unexpected send: %d", notifier.calls)
	}
}
