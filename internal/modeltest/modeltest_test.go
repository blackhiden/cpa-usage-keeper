package modeltest

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type clientStub struct {
	status     int
	err        error
	credential string
	model      string
	mode       Mode
	timeout    time.Duration
}

func (s *clientStub) TestModel(_ context.Context, credential, model string, mode Mode, timeout time.Duration) (int, error) {
	s.credential, s.model, s.mode, s.timeout = credential, model, mode, timeout
	return s.status, s.err
}

type settingsStub struct {
	credential string
	err        error
}

func (s settingsStub) AdminTestCredential(context.Context) (string, error) {
	return s.credential, s.err
}

func TestServiceUsesViewerCredential(t *testing.T) {
	client := &clientStub{status: 200}
	service := NewService(client, settingsStub{credential: "admin-secret"}, 5*time.Second)
	result := service.Test(context.Background(), Request{Model: "gpt-test", Mode: ModeStreaming}, "viewer-secret", true)
	if !result.Success || client.credential != "viewer-secret" || client.mode != ModeStreaming || client.timeout != 5*time.Second {
		t.Fatalf("unexpected result=%+v client=%+v", result, client)
	}
}

func TestServiceRequiresAdminCredential(t *testing.T) {
	client := &clientStub{status: 200}
	service := NewService(client, settingsStub{}, DefaultTimeout)
	result := service.Test(context.Background(), Request{Model: "gpt-test", Mode: ModeRegular}, "forged", false)
	if result.Success || result.ErrorCode != "admin_key_required" || client.credential != "" {
		t.Fatalf("unexpected result=%+v client=%+v", result, client)
	}
}

func TestServiceBoundsFailureReason(t *testing.T) {
	client := &clientStub{status: 502, err: errors.New("upstream: " + strings.Repeat("x", 500))}
	service := NewService(client, settingsStub{credential: "admin-secret"}, DefaultTimeout)
	result := service.Test(context.Background(), Request{Model: "gpt-test", Mode: ModeRegular}, "", false)
	if result.Success || result.ErrorCode != "test_failed" || len(result.Reason) > 240 {
		t.Fatalf("unexpected result=%+v", result)
	}
}

func TestValidateRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		request Request
		want    error
	}{
		{"missing model", Request{Mode: ModeRegular}, ErrInvalidModel},
		{"newline model", Request{Model: "a\nb", Mode: ModeRegular}, ErrInvalidModel},
		{"invalid mode", Request{Model: "a", Mode: "other"}, ErrInvalidMode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !errors.Is(ValidateRequest(tc.request), tc.want) {
				t.Fatalf("got %v", ValidateRequest(tc.request))
			}
		})
	}
}
