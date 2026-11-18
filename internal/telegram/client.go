package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	botAPIBaseURL    = "https://api.telegram.org"
	maxResponseBytes = 64 * 1024
	maxDescription   = 200
)

// BotAPIClient sends messages through the Telegram Bot API.
type BotAPIClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewBotAPIClient builds a sender with a fixed timeout. A nil client yields the
// default bounded client.
func NewBotAPIClient(httpClient *http.Client) *BotAPIClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultTimeout}
	}
	return &BotAPIClient{httpClient: httpClient, baseURL: botAPIBaseURL}
}

// SetBaseURL overrides the Bot API origin. It exists for tests and for
// deployments that front the Bot API with a proxy.
func (c *BotAPIClient) SetBaseURL(baseURL string) {
	if c == nil {
		return
	}
	if trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/"); trimmed != "" {
		c.baseURL = trimmed
	}
}

type sendMessagePayload struct {
	ChatID            string `json:"chat_id"`
	Text              string `json:"text"`
	MessageThreadID   int64  `json:"message_thread_id,omitempty"`
	DisableWebPreview bool   `json:"disable_web_page_preview"`
}

// SendMessage posts one plain-text message. Errors never contain the token.
func (c *BotAPIClient) SendMessage(ctx context.Context, target Target, message Message) error {
	if c == nil || c.httpClient == nil {
		return fmt.Errorf("%w: client is not configured", ErrSendFailed)
	}
	token := strings.TrimSpace(target.BotToken)
	chatID := strings.TrimSpace(target.ChatID)
	if token == "" || chatID == "" {
		return fmt.Errorf("%w: target is incomplete", ErrSendFailed)
	}
	body, err := json.Marshal(sendMessagePayload{
		ChatID:            chatID,
		Text:              Render(message),
		MessageThreadID:   target.MessageThreadID,
		DisableWebPreview: true,
	})
	if err != nil {
		return fmt.Errorf("%w: encode request", ErrSendFailed)
	}
	endpoint := strings.TrimRight(c.baseURL, "/") + "/bot" + token + "/sendMessage"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		// The URL carries the token, so the underlying error is never surfaced.
		return fmt.Errorf("%w: create request", ErrSendFailed)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrSendFailed, redactToken(err.Error(), token))
	}
	defer response.Body.Close()
	payload, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if readErr != nil {
		return fmt.Errorf("%w: read response", ErrSendFailed)
	}
	if len(payload) > maxResponseBytes {
		return fmt.Errorf("%w: response exceeds %d bytes", ErrSendFailed, maxResponseBytes)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%w: %s", ErrSendFailed, describeFailure(payload, token, response.StatusCode))
	}
	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(payload, &result); err == nil && !result.OK {
		return fmt.Errorf("%w: %s", ErrSendFailed, describeFailure(payload, token, response.StatusCode))
	}
	return nil
}

// describeFailure extracts a bounded, token-free description from a Bot API
// error payload.
func describeFailure(payload []byte, token string, statusCode int) string {
	var body struct {
		Description string `json:"description"`
	}
	_ = json.Unmarshal(payload, &body)
	description := sanitizeText(redactToken(body.Description, token), maxDescription)
	if description == "" {
		return fmt.Sprintf("bot API returned status %d", statusCode)
	}
	return description
}
