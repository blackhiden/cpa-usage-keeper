package modeltest

import (
	"context"
	"errors"
	"strings"
	"time"

	"cpa-usage-keeper/internal/cpa"
)

const (
	DefaultTimeout = 30 * time.Second
	MinTimeout     = 1 * time.Second
	MaxTimeout     = 120 * time.Second
)

type Mode string

const (
	ModeRegular   Mode = Mode(cpa.ModelTestModeRegular)
	ModeStreaming Mode = Mode(cpa.ModelTestModeStreaming)
)

var (
	ErrInvalidModel       = errors.New("model is invalid")
	ErrInvalidMode        = errors.New("model test mode is invalid")
	ErrInvalidTimeout     = errors.New("model test timeout is invalid")
	ErrCredentialRequired = errors.New("model test credential is required")
)

type Request struct {
	Model string `json:"model"`
	Mode  Mode   `json:"mode"`
}

type Result struct {
	Success    bool   `json:"success"`
	Mode       Mode   `json:"mode"`
	StatusCode int    `json:"status_code,omitempty"`
	ErrorCode  string `json:"error_code,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

type Client interface {
	TestModel(context.Context, string, string, cpa.ModelTestMode, time.Duration) (int, error)
}

type CredentialSource interface {
	AdminTestCredential(context.Context) (string, error)
}

type TimeoutSource interface {
	RequestTimeout(context.Context) (time.Duration, error)
}

type Service struct {
	client        Client
	settings      CredentialSource
	timeoutSource TimeoutSource
	timeout       time.Duration
}

func NewService(client Client, settings CredentialSource, timeout time.Duration) *Service {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &Service{client: client, settings: settings, timeout: timeout}
}

func (s *Service) SetTimeoutSource(source TimeoutSource) {
	if s != nil {
		s.timeoutSource = source
	}
}

func ValidateRequest(request Request) error {
	model := strings.TrimSpace(request.Model)
	if model == "" || len(model) > 512 || strings.ContainsAny(model, "\r\n") {
		return ErrInvalidModel
	}
	if request.Mode != ModeRegular && request.Mode != ModeStreaming {
		return ErrInvalidMode
	}
	return nil
}

func ValidateTimeout(timeout time.Duration) error {
	if timeout < MinTimeout || timeout > MaxTimeout {
		return ErrInvalidTimeout
	}
	return nil
}

func (s *Service) Test(ctx context.Context, request Request, viewerCredential string, isViewer bool) Result {
	result := Result{Mode: request.Mode}
	if err := ValidateRequest(request); err != nil {
		result.ErrorCode, result.Reason = "invalid_request", err.Error()
		return result
	}
	timeout := s.timeout
	if s.timeoutSource != nil {
		resolved, err := s.timeoutSource.RequestTimeout(ctx)
		if err != nil {
			result.ErrorCode, result.Reason = "settings_unavailable", "admin model test settings are unavailable"
			return result
		}
		timeout = resolved
	}
	if err := ValidateTimeout(timeout); err != nil {
		result.ErrorCode, result.Reason = "invalid_timeout", err.Error()
		return result
	}
	credential := strings.TrimSpace(viewerCredential)
	if isViewer {
		if credential == "" {
			result.ErrorCode, result.Reason = "viewer_credential_unavailable", ErrCredentialRequired.Error()
			return result
		}
	} else {
		var err error
		if s.settings == nil {
			err = ErrCredentialRequired
		} else {
			credential, err = s.settings.AdminTestCredential(ctx)
		}
		if err != nil || strings.TrimSpace(credential) == "" {
			result.ErrorCode, result.Reason = "admin_key_required", "admin test API key is not configured"
			return result
		}
	}
	statusCode, err := s.client.TestModel(ctx, credential, strings.TrimSpace(request.Model), cpa.ModelTestMode(request.Mode), timeout)
	result.StatusCode = statusCode
	if err != nil {
		result.ErrorCode, result.Reason = "test_failed", safeReason(err)
		return result
	}
	result.Success = true
	return result
}

func safeReason(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 {
		message = message[:240]
	}
	return message
}
