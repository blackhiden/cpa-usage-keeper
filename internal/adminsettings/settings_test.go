package adminsettings

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/repository"
	"gorm.io/gorm"
)

func TestAdminSettingsRoundTripMasksAndEncryptsKey(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	key := "admin-test-secret"
	seconds := int64(45)
	settings, err := service.Save(context.Background(), Update{APIKey: &key, RequestTimeoutSeconds: &seconds})
	if err != nil || !settings.APIKeyConfigured || settings.RequestTimeoutSeconds != 45 {
		t.Fatalf("save result=%+v err=%v", settings, err)
	}
	loaded, err := service.Get(context.Background())
	if err != nil || !loaded.APIKeyConfigured || loaded.RequestTimeoutSeconds != 45 {
		t.Fatalf("get result=%+v err=%v", loaded, err)
	}
	stored, found, err := repository.GetAppSetting(context.Background(), db, apiKeySettingKey)
	if err != nil || !found || stored.Value == nil || *stored.Value == key {
		t.Fatalf("stored setting leaked key: found=%v setting=%+v err=%v", found, stored, err)
	}
	credential, err := service.AdminTestCredential(context.Background())
	if err != nil || credential != key {
		t.Fatalf("credential=%q err=%v", credential, err)
	}
}

func TestAdminSettingsBlankKeyClears(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	key := "admin-test-secret"
	if _, err := service.Save(context.Background(), Update{APIKey: &key}); err != nil {
		t.Fatal(err)
	}
	blank := ""
	settings, err := service.Save(context.Background(), Update{APIKey: &blank})
	if err != nil || settings.APIKeyConfigured {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestAdminSettingsRejectsInvalidTimeoutWithoutChangingValue(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	valid := int64(10)
	if _, err := service.Save(context.Background(), Update{RequestTimeoutSeconds: &valid}); err != nil {
		t.Fatal(err)
	}
	invalid := int64(121)
	if _, err := service.Save(context.Background(), Update{RequestTimeoutSeconds: &invalid}); err == nil {
		t.Fatal("expected invalid timeout")
	}
	settings, err := service.Get(context.Background())
	if err != nil || settings.RequestTimeoutSeconds != valid {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func TestAdminSettingsTelegramRoundTripKeepsTokenSecret(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	token := "123456789:AAHtest-token-value"
	chatID := "-1001234567890"
	threadID := int64(42)
	settings, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{
		BotToken: &token, ChatID: &chatID, MessageThreadID: &threadID,
	}})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if !settings.Telegram.BotTokenConfigured || settings.Telegram.ChatID != chatID || settings.Telegram.MessageThreadID != threadID {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	payload, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), token) {
		t.Fatalf("settings payload leaked the token: %s", payload)
	}
	stored, found, err := repository.GetAppSetting(context.Background(), db, telegramBotTokenKey)
	if err != nil || !found || stored.Value == nil || *stored.Value == token {
		t.Fatalf("stored telegram token is not encrypted: found=%v setting=%+v err=%v", found, stored, err)
	}
	config, err := service.TelegramConfig(context.Background())
	if err != nil || !config.Configured || config.BotToken != token || config.ChatID != chatID || config.MessageThreadID != threadID {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestAdminSettingsTelegramPartialUpdateKeepsOtherFields(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	token := "123456789:AAHtest-token-value"
	chatID := "-1001234567890"
	if _, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{BotToken: &token, ChatID: &chatID}}); err != nil {
		t.Fatal(err)
	}
	threadID := int64(7)
	settings, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{MessageThreadID: &threadID}})
	if err != nil {
		t.Fatal(err)
	}
	if !settings.Telegram.BotTokenConfigured || settings.Telegram.ChatID != chatID || settings.Telegram.MessageThreadID != 7 {
		t.Fatalf("partial update dropped fields: %+v", settings)
	}
}

func TestAdminSettingsTelegramClearingMakesFeatureInert(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	token := "123456789:AAHtest-token-value"
	chatID := "-1001234567890"
	if _, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{BotToken: &token, ChatID: &chatID}}); err != nil {
		t.Fatal(err)
	}
	blank := ""
	if _, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{BotToken: &blank}}); err != nil {
		t.Fatal(err)
	}
	config, err := service.TelegramConfig(context.Background())
	if err != nil || config.Configured || config.BotToken != "" {
		t.Fatalf("config=%+v err=%v", config, err)
	}
}

func TestAdminSettingsTelegramRejectsInvalidValues(t *testing.T) {
	db := openTestDatabase(t)
	service := NewService(db, "management-secret")
	valid := "123456789:AAHtest-token-value"
	for _, tc := range []struct {
		name   string
		update TelegramUpdate
	}{
		{"token without colon", TelegramUpdate{BotToken: stringPtr("123456789AAHtest")}},
		{"token with whitespace", TelegramUpdate{BotToken: stringPtr("123456789:AAH test")}},
		{"token with non-numeric id", TelegramUpdate{BotToken: stringPtr("abc:AAHtest-token-value")}},
		{"chat id with letters", TelegramUpdate{ChatID: stringPtr("-1001234abc")}},
		{"chat username too short", TelegramUpdate{ChatID: stringPtr("@abc")}},
		{"topic id negative", TelegramUpdate{MessageThreadID: int64Ptr(-5)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := service.Save(context.Background(), Update{Telegram: &tc.update}); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	settings, err := service.Save(context.Background(), Update{Telegram: &TelegramUpdate{BotToken: &valid, ChatID: stringPtr("@team_chat")}})
	if err != nil || !settings.Telegram.BotTokenConfigured || settings.Telegram.ChatID != "@team_chat" {
		t.Fatalf("settings=%+v err=%v", settings, err)
	}
}

func stringPtr(value string) *string { return &value }

func int64Ptr(value int64) *int64 { return &value }

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := repository.OpenDatabase(config.Config{SQLitePath: filepath.Join(t.TempDir(), "admin-settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
