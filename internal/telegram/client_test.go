package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBotAPIClientSendsMessage(t *testing.T) {
	var gotPath string
	var gotBody sendMessagePayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1}}`))
	}))
	defer server.Close()

	client := NewBotAPIClient(server.Client())
	client.SetBaseURL(server.URL)
	target := Target{BotToken: "123456789:AAHtest-token-value", ChatID: "-1001234567890", MessageThreadID: 7}
	err := client.SendMessage(context.Background(), target, Message{Reporter: "Team Key", Model: "gpt-test", Reason: "boom"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/bot"+target.BotToken+"/sendMessage" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotBody.ChatID != target.ChatID || gotBody.MessageThreadID != 7 || !gotBody.DisableWebPreview {
		t.Fatalf("unexpected payload: %+v", gotBody)
	}
	if !strings.Contains(gotBody.Text, "Reporter: Team Key") || !strings.Contains(gotBody.Text, "Model: gpt-test") {
		t.Fatalf("unexpected text: %q", gotBody.Text)
	}
}

func TestBotAPIClientNon2xxDoesNotLeakToken(t *testing.T) {
	token := "123456789:AAHtest-token-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"ok":false,"description":"Unauthorized bot ` + token + `"}`))
	}))
	defer server.Close()

	client := NewBotAPIClient(server.Client())
	client.SetBaseURL(server.URL)
	err := client.SendMessage(context.Background(), Target{BotToken: token, ChatID: "1"}, Message{Reporter: "Team Key", Model: "gpt-test"})
	if !errors.Is(err, ErrSendFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked the bot token: %v", err)
	}
	if !strings.Contains(err.Error(), "Unauthorized bot") {
		t.Fatalf("error lost the Bot API description: %v", err)
	}
}

func TestBotAPIClientTransportErrorDoesNotLeakToken(t *testing.T) {
	token := "123456789:AAHtest-token-value"
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := NewBotAPIClient(server.Client())
	client.SetBaseURL(server.URL)
	server.Close()

	err := client.SendMessage(context.Background(), Target{BotToken: token, ChatID: "1"}, Message{Reporter: "Team Key", Model: "gpt-test"})
	if !errors.Is(err, ErrSendFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), token) {
		t.Fatalf("error leaked the bot token: %v", err)
	}
}

func TestBotAPIClientRejectsIncompleteTarget(t *testing.T) {
	client := NewBotAPIClient(nil)
	if err := client.SendMessage(context.Background(), Target{ChatID: "1"}, Message{Reporter: "Team Key", Model: "gpt-test"}); !errors.Is(err, ErrSendFailed) {
		t.Fatalf("unexpected error: %v", err)
	}
}
