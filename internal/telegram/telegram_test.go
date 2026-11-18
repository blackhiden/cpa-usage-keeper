package telegram

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type notifierStub struct {
	calls   int
	target  Target
	message Message
	err     error
}

func (s *notifierStub) SendMessage(_ context.Context, target Target, message Message) error {
	s.calls++
	s.target = target
	s.message = message
	return s.err
}

type settingsStub struct {
	settings Settings
	err      error
}

func (s settingsStub) TelegramSettings(context.Context) (Settings, error) {
	return s.settings, s.err
}

func configuredSettings() Settings {
	return Settings{BotToken: "123456789:AAHtest-token-value", ChatID: "-1001234567890", MessageThreadID: 42, Enabled: true, Configured: true}
}

func recordedFailure() Failure {
	return Failure{Model: "gpt-test", Mode: ModeStreaming, StatusCode: 502, ErrorCode: "test_failed", Reason: "upstream timeout"}
}

func TestBuildMessageRedactsAndBoundsReason(t *testing.T) {
	reason := "upstream rejected sk-abcdefghijklmnop with Authorization: Bearer abcdefghijklmnop\n" + strings.Repeat("x", 400)
	message, err := BuildMessage("Team Key", Failure{Model: "gpt-test", Mode: ModeStreaming, StatusCode: 502, ErrorCode: "test_failed", Reason: reason})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(message.Reason, "sk-abcdefghijklmnop") || strings.Contains(message.Reason, "abcdefghijklmnop") {
		t.Fatalf("reason leaked a credential: %q", message.Reason)
	}
	if strings.ContainsAny(message.Reason, "\r\n") {
		t.Fatalf("reason kept control characters: %q", message.Reason)
	}
	if len(message.Reason) > MaxReasonLength {
		t.Fatalf("reason length %d exceeds %d", len(message.Reason), MaxReasonLength)
	}
	if message.Reporter != "Team Key" || message.Model != "gpt-test" || message.StatusCode != 502 {
		t.Fatalf("unexpected message: %+v", message)
	}
}

func TestBuildMessageRejectsInvalidPayload(t *testing.T) {
	for _, tc := range []struct {
		name     string
		reporter string
		failure  Failure
	}{
		{"missing model", "Team Key", Failure{Mode: ModeRegular}},
		{"newline model", "Team Key", Failure{Model: "a\nb", Mode: ModeRegular}},
		{"invalid mode", "Team Key", Failure{Model: "a", Mode: "other"}},
		{"invalid status", "Team Key", Failure{Model: "a", StatusCode: 700}},
		{"missing reporter", "  ", Failure{Model: "a"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildMessage(tc.reporter, tc.failure); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestRenderIncludesReporterModelAndReason(t *testing.T) {
	rendered := Render(Message{Reporter: "Team Key", Model: "gpt-test", Mode: ModeRegular, StatusCode: 500, ErrorCode: "test_failed", Reason: "boom"})
	for _, want := range []string{"Reporter: Team Key", "Model: gpt-test", "Status: 500", "Error: test_failed", "Reason: boom"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered message missing %q: %q", want, rendered)
		}
	}
}

func TestNotifySendsRecordedFailure(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: configuredSettings()})
	service.RecordFailure("scope-team", recordedFailure())
	// The client only supplies model and mode; every failure field is server-authored.
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "gpt-test", Mode: ModeStreaming})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.calls != 1 {
		t.Fatalf("expected one send, got %d", notifier.calls)
	}
	if notifier.target.ChatID != "-1001234567890" || notifier.target.MessageThreadID != 42 {
		t.Fatalf("unexpected target: %+v", notifier.target)
	}
	if notifier.message.Reporter != "Team Key" || notifier.message.Model != "gpt-test" || notifier.message.Reason != "upstream timeout" {
		t.Fatalf("unexpected message: %+v", notifier.message)
	}
}

func TestNotifyIgnoresFailureFromAnotherScope(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: configuredSettings()})
	service.RecordFailure("scope-team", recordedFailure())
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "other-scope", Model: "gpt-test", Mode: ModeStreaming})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.message.Reason != "" || notifier.message.StatusCode != 0 {
		t.Fatalf("cross-scope failure leaked: %+v", notifier.message)
	}
}

func TestNotifyDisabled(t *testing.T) {
	notifier := &notifierStub{}
	settings := configuredSettings()
	settings.Enabled = false
	service := NewService(notifier, settingsStub{settings: settings})
	service.RecordFailure("scope-team", recordedFailure())
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "gpt-test"})
	if !errors.Is(err, ErrDisabled) || notifier.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
}

func TestNotifyMutedModel(t *testing.T) {
	notifier := &notifierStub{}
	settings := configuredSettings()
	settings.MutedModels = []string{"gpt-test"}
	service := NewService(notifier, settingsStub{settings: settings})
	service.RecordFailure("scope-team", recordedFailure())
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "GPT-TEST"})
	if !errors.Is(err, ErrModelMuted) || notifier.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
}

func TestSendTestMessageUsesConfiguredTarget(t *testing.T) {
	notifier := &notifierStub{}
	settings := configuredSettings()
	settings.Enabled = false
	service := NewService(notifier, settingsStub{settings: settings})
	if err := service.SendTestMessage(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if notifier.calls != 1 || notifier.message.Kind != KindTest {
		t.Fatalf("calls=%d message=%+v", notifier.calls, notifier.message)
	}
	if rendered := Render(notifier.message); !strings.Contains(rendered, "test message") {
		t.Fatalf("unexpected test body: %q", rendered)
	}
}

func TestSendTestMessageRequiresConfiguration(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: Settings{}})
	if err := service.SendTestMessage(context.Background()); !errors.Is(err, ErrNotConfigured) || notifier.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
}

func TestNotifyNotConfigured(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: Settings{}})
	service.RecordFailure("scope-team", recordedFailure())
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "gpt-test"})
	if !errors.Is(err, ErrNotConfigured) || notifier.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
}

func TestNotifyRateLimited(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: configuredSettings()})
	service.RecordFailure("scope-team", recordedFailure())
	input := NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "gpt-test", Mode: ModeRegular}
	if err := service.Notify(context.Background(), input); err != nil {
		t.Fatalf("first notify failed: %v", err)
	}
	err := service.Notify(context.Background(), input)
	if !errors.Is(err, ErrRateLimited) || notifier.calls != 1 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
	var rateLimit *RateLimitError
	if !errors.As(err, &rateLimit) || rateLimit.RetryAfter <= 0 {
		t.Fatalf("expected retry hint, got %+v", err)
	}
}

func TestNotifySendFailureNeverLeaksToken(t *testing.T) {
	settings := configuredSettings()
	notifier := &notifierStub{err: errors.New("post https://api.telegram.org/bot" + settings.BotToken + "/sendMessage: connection refused")}
	service := NewService(notifier, settingsStub{settings: settings})
	service.RecordFailure("scope-team", recordedFailure())
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team", Model: "gpt-test"})
	if !errors.Is(err, ErrSendFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), settings.BotToken) {
		t.Fatalf("error leaked the bot token: %v", err)
	}
}

func TestNotifyRejectsInvalidRequestBeforeSending(t *testing.T) {
	notifier := &notifierStub{}
	service := NewService(notifier, settingsStub{settings: configuredSettings()})
	err := service.Notify(context.Background(), NotifyInput{Reporter: "Team Key", Scope: "scope-team"})
	if !errors.Is(err, ErrInvalidRequest) || notifier.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, notifier.calls)
	}
}

func TestLimiterCooldownIsPerReporterAndModel(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := NewLimiter(LimiterOptions{Cooldown: time.Minute, HourlyLimit: 10, Now: func() time.Time { return now }})
	if allowed, _ := limiter.Allow("Team Key", "gpt-test"); !allowed {
		t.Fatal("first notification should be allowed")
	}
	if allowed, wait := limiter.Allow("Team Key", "gpt-test"); allowed || wait <= 0 {
		t.Fatalf("cooldown not applied: allowed=%v wait=%v", allowed, wait)
	}
	if allowed, _ := limiter.Allow("Team Key", "other-model"); !allowed {
		t.Fatal("a different model should not be throttled")
	}
	if allowed, _ := limiter.Allow("Other Key", "gpt-test"); !allowed {
		t.Fatal("a different reporter should not be throttled")
	}
	now = now.Add(time.Minute)
	if allowed, _ := limiter.Allow("Team Key", "gpt-test"); !allowed {
		t.Fatal("cooldown should expire")
	}
}

func TestLimiterHourlyCap(t *testing.T) {
	now := time.Unix(1700000000, 0)
	limiter := NewLimiter(LimiterOptions{Cooldown: time.Second, HourlyLimit: 2, Now: func() time.Time { return now }})
	if allowed, _ := limiter.Allow("Team Key", "a"); !allowed {
		t.Fatal("first should be allowed")
	}
	now = now.Add(2 * time.Second)
	if allowed, _ := limiter.Allow("Team Key", "b"); !allowed {
		t.Fatal("second should be allowed")
	}
	now = now.Add(2 * time.Second)
	if allowed, wait := limiter.Allow("Team Key", "c"); allowed || wait <= 0 {
		t.Fatalf("hourly cap not applied: allowed=%v wait=%v", allowed, wait)
	}
	now = now.Add(time.Hour)
	if allowed, _ := limiter.Allow("Team Key", "c"); !allowed {
		t.Fatal("hourly window should reset")
	}
}
