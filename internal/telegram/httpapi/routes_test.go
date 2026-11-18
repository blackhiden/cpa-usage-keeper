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
		BotToken: "123456789:AAHtest-token-value", ChatID: "-1001234567890", Configured: true,
	}})
}

func TestNotifyRouteResolvesReporterServerSide(t *testing.T) {
	notifier := &notifierStub{}
	router := testRouter(configuredService(notifier), func(*gin.Context) (string, bool) {
		return "Team Key", true
	})
	recorder := postNotify(router, `{"model":"gpt-test","mode":"streaming","status_code":502,"error_code":"test_failed","reason":"boom","reporter":"Forged"}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if notifier.calls != 1 || notifier.message.Reporter != "Team Key" {
		t.Fatalf("calls=%d message=%+v", notifier.calls, notifier.message)
	}
	if notifier.message.Model != "gpt-test" {
		t.Fatalf("unexpected model: %+v", notifier.message)
	}
}

func TestNotifyRouteRateLimited(t *testing.T) {
	notifier := &notifierStub{}
	service := configuredService(notifier)
	router := testRouter(service, func(*gin.Context) (string, bool) { return "Team Key", true })
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

func TestNotifyRouteSendFailed(t *testing.T) {
	notifier := &notifierStub{err: telegram.ErrSendFailed}
	router := testRouter(configuredService(notifier), func(*gin.Context) (string, bool) { return "Team Key", true })
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
	router := testRouter(configuredService(notifier), func(*gin.Context) (string, bool) { return "Team Key", true })
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
	router := testRouter(configuredService(notifier), func(*gin.Context) (string, bool) { return "", false })
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
