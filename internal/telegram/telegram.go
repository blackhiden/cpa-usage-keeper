// Package telegram delivers model-test failure notifications to an
// administrator through the Telegram Bot API.
//
// The domain is self-contained: it imports no Gin, GORM, or settings package.
// Configuration arrives through SettingsSource and delivery through Notifier,
// so the composition root owns all wiring. Nothing in this package may return,
// log, or render the bot token.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout bounds a single Bot API call.
	DefaultTimeout = 10 * time.Second
	// MaxReasonLength mirrors the model-test failure reason bound.
	MaxReasonLength = 240
	// MaxModelLength mirrors the model-test model bound.
	MaxModelLength = 512
	// MaxErrorCodeLength bounds the upstream error code echoed in the message.
	MaxErrorCodeLength = 64
	// MaxReporterLength bounds the resolved reporter label.
	MaxReporterLength = 128
	// MaxChatIDLength bounds the configured chat identifier.
	MaxChatIDLength = 64
)

// Stable, non-secret error codes returned to HTTP callers.
const (
	CodeNotConfigured  = "telegram_not_configured"
	CodeDisabled       = "telegram_disabled"
	CodeModelMuted     = "telegram_model_muted"
	CodeSendFailed     = "telegram_send_failed"
	CodeRateLimited    = "notify_rate_limited"
	CodeInvalidRequest = "invalid_notify_request"
)

var (
	ErrNotConfigured  = errors.New("telegram is not configured")
	ErrDisabled       = errors.New("telegram notifications are disabled")
	ErrModelMuted     = errors.New("telegram model is muted")
	ErrSendFailed     = errors.New("telegram send failed")
	ErrRateLimited    = errors.New("telegram notification rate limited")
	ErrInvalidRequest = errors.New("invalid telegram notification request")
)

// Message kinds. The message model stays channel-neutral (REC-014).
const (
	KindFailure = "failure"
	KindTest    = "test"
)

// Model-test modes accepted from the failure contract.
const (
	ModeRegular   = "regular"
	ModeStreaming = "streaming"
)

// Request is the caller-supplied notification payload. It deliberately has no
// reporter and no failure fields: the reporter is resolved server-side from the
// session and the failure comes from the server-side record (REC-012).
type Request struct {
	Model string `json:"model"`
	Mode  string `json:"mode"`
}

// Message is the channel-neutral notification content.
type Message struct {
	Kind       string
	Reporter   string
	Model      string
	Mode       string
	StatusCode int
	ErrorCode  string
	Reason     string
}

// Target is the resolved delivery destination.
type Target struct {
	BotToken        string
	ChatID          string
	MessageThreadID int64
}

// Settings is the notifier-facing configuration view. Configured is false when
// the feature is inert; BotToken is only ever held in memory.
type Settings struct {
	BotToken        string
	ChatID          string
	MessageThreadID int64
	Enabled         bool
	MutedModels     []string
	Configured      bool
}

// SettingsSource supplies the current Telegram configuration.
type SettingsSource interface {
	TelegramSettings(ctx context.Context) (Settings, error)
}

// Notifier delivers a rendered message to a target.
type Notifier interface {
	SendMessage(ctx context.Context, target Target, message Message) error
}

// RateLimitError reports a suppressed notification and the wait before retry.
type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string { return ErrRateLimited.Error() }

func (e *RateLimitError) Unwrap() error { return ErrRateLimited }

// AdminReporterLabel is the reporter label used when an administrator triggers
// a notification (OQ-6 baseline: API Key Name exists only for viewer sessions).
const AdminReporterLabel = "Administrator"

// Service validates, records, rate-limits, and delivers notifications.
type Service struct {
	notifier Notifier
	settings SettingsSource
	limiter  *Limiter
	failures *FailureStore
}

// NewService builds the notification service with the default limiter and an
// in-memory failure store (REC-012).
func NewService(notifier Notifier, settings SettingsSource) *Service {
	return &Service{
		notifier: notifier,
		settings: settings,
		limiter:  NewLimiter(LimiterOptions{}),
		failures: NewFailureStore(DefaultFailureTTL),
	}
}

// SetLimiter replaces the default limiter; nil disables rate limiting.
func (s *Service) SetLimiter(limiter *Limiter) {
	if s != nil {
		s.limiter = limiter
	}
}

// SetFailureStore replaces the default failure store.
func (s *Service) SetFailureStore(store *FailureStore) {
	if s != nil && store != nil {
		s.failures = store
	}
}

// RecordFailure stores the server-authored failure for a session scope so the
// notify endpoint does not need to trust the browser payload.
func (s *Service) RecordFailure(scope string, failure Failure) {
	if s == nil || s.failures == nil {
		return
	}
	s.failures.Record(scope, failure)
}

// LookupFailure returns the recorded failure for a session scope and model.
func (s *Service) LookupFailure(scope, model string) (Failure, bool) {
	if s == nil || s.failures == nil {
		return Failure{}, false
	}
	return s.failures.Lookup(scope, model)
}

// NotifyInput is the server-authoritative notification request.
type NotifyInput struct {
	Reporter string
	Scope    string
	Model    string
	Mode     string
}

// Notify resolves the recorded failure, applies the configuration, mute, and
// rate-limit rules, and delivers the message. The returned error wraps one of
// the package sentinels and never contains the bot token.
func (s *Service) Notify(ctx context.Context, input NotifyInput) error {
	if s == nil || s.notifier == nil || s.settings == nil {
		return ErrNotConfigured
	}
	model := strings.TrimSpace(input.Model)
	if model == "" || len(model) > MaxModelLength || strings.ContainsAny(model, "\r\n") {
		return fmt.Errorf("%w: model is invalid", ErrInvalidRequest)
	}
	settings, err := s.settings.TelegramSettings(ctx)
	if err != nil {
		return ErrNotConfigured
	}
	target, err := resolveTarget(settings, true)
	if err != nil {
		return err
	}
	failure := Failure{Model: model, Mode: strings.TrimSpace(input.Mode)}
	if recorded, ok := s.LookupFailure(input.Scope, model); ok {
		failure = recorded
		if failure.Mode == "" {
			failure.Mode = strings.TrimSpace(input.Mode)
		}
	}
	if isMutedModel(settings.MutedModels, failure.Model) {
		return ErrModelMuted
	}
	message, err := BuildMessage(input.Reporter, failure)
	if err != nil {
		return err
	}
	return s.deliver(ctx, target, message)
}

// SendTestMessage delivers the fixed configuration-check message (REC-011).
// It requires a configured target but intentionally ignores the enable toggle:
// the administrator explicitly asked for a test.
func (s *Service) SendTestMessage(ctx context.Context) error {
	if s == nil || s.notifier == nil || s.settings == nil {
		return ErrNotConfigured
	}
	settings, err := s.settings.TelegramSettings(ctx)
	if err != nil {
		return ErrNotConfigured
	}
	target, err := resolveTarget(settings, false)
	if err != nil {
		return err
	}
	return s.deliver(ctx, target, Message{Kind: KindTest, Reporter: AdminReporterLabel})
}

func (s *Service) deliver(ctx context.Context, target Target, message Message) error {
	if s.limiter != nil {
		if allowed, retryAfter := s.limiter.Allow(message.Reporter, limiterKey(message)); !allowed {
			return &RateLimitError{RetryAfter: retryAfter}
		}
	}
	if err := s.notifier.SendMessage(ctx, target, message); err != nil {
		return fmt.Errorf("%w: %s", ErrSendFailed, redactToken(err.Error(), target.BotToken))
	}
	return nil
}

// limiterKey keeps the test action out of the failure cooldown bucket.
func limiterKey(message Message) string {
	if message.Kind == KindTest {
		return "test"
	}
	return message.Model
}

// resolveTarget validates the configuration. requireEnabled enforces the
// administrator toggle for failure notifications.
func resolveTarget(settings Settings, requireEnabled bool) (Target, error) {
	target := Target{
		BotToken:        strings.TrimSpace(settings.BotToken),
		ChatID:          strings.TrimSpace(settings.ChatID),
		MessageThreadID: settings.MessageThreadID,
	}
	if !settings.Configured || target.BotToken == "" || target.ChatID == "" {
		return Target{}, ErrNotConfigured
	}
	if requireEnabled && !settings.Enabled {
		return Target{}, ErrDisabled
	}
	return target, nil
}

// isMutedModel reports whether the model is on the administrator mute list.
func isMutedModel(muted []string, model string) bool {
	for _, entry := range muted {
		if strings.EqualFold(strings.TrimSpace(entry), model) {
			return true
		}
	}
	return false
}

// BuildMessage validates and normalizes the server-authored failure into a
// message. No field originates from the browser.
func BuildMessage(reporter string, failure Failure) (Message, error) {
	model := strings.TrimSpace(failure.Model)
	if model == "" || len(model) > MaxModelLength || strings.ContainsAny(model, "\r\n") {
		return Message{}, fmt.Errorf("%w: model is invalid", ErrInvalidRequest)
	}
	mode := strings.TrimSpace(failure.Mode)
	if mode != "" && mode != ModeRegular && mode != ModeStreaming {
		return Message{}, fmt.Errorf("%w: mode is invalid", ErrInvalidRequest)
	}
	if failure.StatusCode < 0 || failure.StatusCode > 599 {
		return Message{}, fmt.Errorf("%w: status code is invalid", ErrInvalidRequest)
	}
	label := sanitizeText(reporter, MaxReporterLength)
	if label == "" {
		return Message{}, fmt.Errorf("%w: reporter is required", ErrInvalidRequest)
	}
	return Message{
		Kind:       KindFailure,
		Reporter:   label,
		Model:      model,
		Mode:       mode,
		StatusCode: failure.StatusCode,
		ErrorCode:  sanitizeText(failure.ErrorCode, MaxErrorCodeLength),
		Reason:     sanitizeReason(failure.Reason),
	}, nil
}

// Render produces the plain-text message body sent to Telegram.
func Render(message Message) string {
	if message.Kind == KindTest {
		return "CPA Usage Keeper test message\nTelegram notifications are configured correctly."
	}
	var builder strings.Builder
	builder.WriteString("Model test failure\n")
	builder.WriteString("Reporter: " + message.Reporter + "\n")
	builder.WriteString("Model: " + message.Model + "\n")
	if message.Mode != "" {
		builder.WriteString("Mode: " + message.Mode + "\n")
	}
	if message.StatusCode > 0 {
		builder.WriteString("Status: " + strconv.Itoa(message.StatusCode) + "\n")
	}
	if message.ErrorCode != "" {
		builder.WriteString("Error: " + message.ErrorCode + "\n")
	}
	if message.Reason != "" {
		builder.WriteString("Reason: " + message.Reason)
	}
	return strings.TrimRight(builder.String(), "\n")
}

var keyLikePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bsk-[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`(?i)\b(bearer)\s+[A-Za-z0-9._~+/=-]{8,}`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|bot[_-]?token|secret|password)\b\s*[:=]\s*\S{4,}`),
}

// sanitizeReason strips control characters, redacts key-like substrings, and
// bounds the length so a provider error cannot smuggle a credential or an
// unbounded payload into the administrator chat.
func sanitizeReason(value string) string {
	cleaned := sanitizeText(value, MaxReasonLength*4)
	for _, pattern := range keyLikePatterns {
		cleaned = pattern.ReplaceAllString(cleaned, "${1}[redacted]")
	}
	cleaned = strings.ReplaceAll(cleaned, "[redacted][redacted]", "[redacted]")
	return sanitizeText(cleaned, MaxReasonLength)
}

// sanitizeText collapses whitespace and control characters and bounds length.
func sanitizeText(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastSpace := false
	for _, r := range trimmed {
		if r < 0x20 || r == 0x7f || r == ' ' || r == '\t' {
			if !lastSpace {
				builder.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		builder.WriteRune(r)
		lastSpace = false
	}
	result := strings.TrimSpace(builder.String())
	if limit > 0 && len(result) > limit {
		result = strings.TrimSpace(result[:limit])
	}
	return result
}

// redactToken removes any occurrence of the token from free-form text.
func redactToken(text, token string) string {
	if token == "" {
		return text
	}
	return strings.ReplaceAll(text, token, "[redacted]")
}
