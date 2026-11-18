package adminsettings

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	"gorm.io/gorm"
)

const (
	apiKeySettingKey      = "admin.model_test.api_key"
	timeoutSettingKey     = "admin.model_test.timeout_seconds"
	telegramBotTokenKey   = "admin.telegram.bot_token"
	telegramChatIDKey     = "admin.telegram.chat_id"
	telegramThreadIDKey   = "admin.telegram.message_thread_id"
	encryptedValueType    = "encrypted"
	integerValueType      = "int"
	stringValueType       = "string"
	defaultTimeoutSeconds = int64(30)
	minTimeoutSeconds     = int64(1)
	maxTimeoutSeconds     = int64(120)
	minBotTokenLength     = 20
	maxBotTokenLength     = 200
	maxChatIDLength       = 64
	maxMessageThreadID    = int64(1 << 31)
)

var ErrProtectionKeyRequired = errors.New("admin settings protection key is required")

// TelegramSettings is the API-facing Telegram view. It never carries the token.
type TelegramSettings struct {
	BotTokenConfigured bool   `json:"bot_token_configured"`
	ChatID             string `json:"chat_id"`
	MessageThreadID    int64  `json:"message_thread_id,omitempty"`
}

type Settings struct {
	APIKeyConfigured      bool             `json:"api_key_configured"`
	RequestTimeoutSeconds int64            `json:"request_timeout_seconds"`
	Telegram              TelegramSettings `json:"telegram"`
}

// TelegramUpdate carries a partial Telegram update; nil fields keep their value
// and an explicit empty value clears the stored setting.
type TelegramUpdate struct {
	BotToken        *string
	ChatID          *string
	MessageThreadID *int64
}

type Update struct {
	APIKey                *string
	RequestTimeoutSeconds *int64
	Telegram              *TelegramUpdate
}

// TelegramConfig is the secret-bearing view consumed by the notification
// domain. It is never serialized to an API response.
type TelegramConfig struct {
	BotToken        string
	ChatID          string
	MessageThreadID int64
	Configured      bool
}

type Service struct {
	db            *gorm.DB
	protectionKey string
}

func NewService(db *gorm.DB, protectionKey string) *Service {
	return &Service{db: db, protectionKey: strings.TrimSpace(protectionKey)}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	settings, err := repository.GetAppSettings(ctx, s.db, []string{
		apiKeySettingKey, timeoutSettingKey, telegramBotTokenKey, telegramChatIDKey, telegramThreadIDKey,
	})
	if err != nil {
		return Settings{}, err
	}
	result := Settings{RequestTimeoutSeconds: defaultTimeoutSeconds}
	if setting, ok := settings[apiKeySettingKey]; ok && setting.Value != nil && strings.TrimSpace(*setting.Value) != "" {
		if _, err := s.decrypt(*setting.Value); err != nil {
			return Settings{}, fmt.Errorf("decrypt admin test key: %w", err)
		}
		result.APIKeyConfigured = true
	}
	if setting, ok := settings[timeoutSettingKey]; ok && setting.Value != nil {
		seconds, err := strconv.ParseInt(strings.TrimSpace(*setting.Value), 10, 64)
		if err != nil || seconds < minTimeoutSeconds || seconds > maxTimeoutSeconds {
			return Settings{}, fmt.Errorf("stored admin test timeout is invalid")
		}
		result.RequestTimeoutSeconds = seconds
	}
	telegram, err := s.telegramFromSettings(settings)
	if err != nil {
		return Settings{}, err
	}
	result.Telegram = TelegramSettings{
		BotTokenConfigured: telegram.BotToken != "",
		ChatID:             telegram.ChatID,
		MessageThreadID:    telegram.MessageThreadID,
	}
	return result, nil
}

func (s *Service) Save(ctx context.Context, update Update) (Settings, error) {
	if update.RequestTimeoutSeconds != nil {
		seconds := *update.RequestTimeoutSeconds
		if seconds < minTimeoutSeconds || seconds > maxTimeoutSeconds {
			return Settings{}, fmt.Errorf("request timeout must be between %d and %d seconds", minTimeoutSeconds, maxTimeoutSeconds)
		}
	}
	// Validate every Telegram field before writing anything so a rejected
	// request never leaves partially applied settings behind.
	if update.Telegram != nil {
		if update.Telegram.BotToken != nil {
			if value := strings.TrimSpace(*update.Telegram.BotToken); value != "" {
				if err := validateBotToken(value); err != nil {
					return Settings{}, err
				}
			}
		}
		if update.Telegram.ChatID != nil {
			if value := strings.TrimSpace(*update.Telegram.ChatID); value != "" {
				if err := validateChatID(value); err != nil {
					return Settings{}, err
				}
			}
		}
		if update.Telegram.MessageThreadID != nil {
			if id := *update.Telegram.MessageThreadID; id != 0 {
				if err := validateMessageThreadID(id); err != nil {
					return Settings{}, err
				}
			}
		}
	}
	current, err := s.Get(ctx)
	if err != nil {
		return Settings{}, err
	}
	if update.APIKey != nil {
		value := strings.TrimSpace(*update.APIKey)
		if value == "" {
			if err := s.upsert(ctx, apiKeySettingKey, nil, encryptedValueType); err != nil {
				return Settings{}, err
			}
			current.APIKeyConfigured = false
		} else {
			encrypted, err := s.encrypt(value)
			if err != nil {
				return Settings{}, err
			}
			if err := s.upsert(ctx, apiKeySettingKey, &encrypted, encryptedValueType); err != nil {
				return Settings{}, err
			}
			current.APIKeyConfigured = true
		}
	}
	if update.RequestTimeoutSeconds != nil {
		seconds := *update.RequestTimeoutSeconds
		value := strconv.FormatInt(seconds, 10)
		if err := s.upsert(ctx, timeoutSettingKey, &value, integerValueType); err != nil {
			return Settings{}, err
		}
		current.RequestTimeoutSeconds = seconds
	}
	if update.Telegram != nil {
		if err := s.saveTelegram(ctx, update.Telegram, &current.Telegram); err != nil {
			return Settings{}, err
		}
	}
	return current, nil
}

func (s *Service) saveTelegram(ctx context.Context, update *TelegramUpdate, current *TelegramSettings) error {
	if update.BotToken != nil {
		value := strings.TrimSpace(*update.BotToken)
		if value == "" {
			if err := s.upsert(ctx, telegramBotTokenKey, nil, encryptedValueType); err != nil {
				return err
			}
			current.BotTokenConfigured = false
		} else {
			encrypted, err := s.encrypt(value)
			if err != nil {
				return err
			}
			if err := s.upsert(ctx, telegramBotTokenKey, &encrypted, encryptedValueType); err != nil {
				return err
			}
			current.BotTokenConfigured = true
		}
	}
	if update.ChatID != nil {
		value := strings.TrimSpace(*update.ChatID)
		if err := s.upsertOptional(ctx, telegramChatIDKey, value, stringValueType); err != nil {
			return err
		}
		current.ChatID = value
	}
	if update.MessageThreadID != nil {
		id := *update.MessageThreadID
		if id == 0 {
			if err := s.upsert(ctx, telegramThreadIDKey, nil, integerValueType); err != nil {
				return err
			}
			current.MessageThreadID = 0
		} else {
			value := strconv.FormatInt(id, 10)
			if err := s.upsert(ctx, telegramThreadIDKey, &value, integerValueType); err != nil {
				return err
			}
			current.MessageThreadID = id
		}
	}
	return nil
}

// TelegramConfig returns the secret-bearing configuration used by the
// notification domain. It is never marshalled into an API response.
func (s *Service) TelegramConfig(ctx context.Context) (TelegramConfig, error) {
	settings, err := repository.GetAppSettings(ctx, s.db, []string{telegramBotTokenKey, telegramChatIDKey, telegramThreadIDKey})
	if err != nil {
		return TelegramConfig{}, err
	}
	return s.telegramFromSettings(settings)
}

func (s *Service) telegramFromSettings(settings map[string]entities.AppSetting) (TelegramConfig, error) {
	config := TelegramConfig{}
	if setting, ok := settings[telegramBotTokenKey]; ok && setting.Value != nil && strings.TrimSpace(*setting.Value) != "" {
		token, err := s.decrypt(*setting.Value)
		if err != nil {
			return TelegramConfig{}, fmt.Errorf("decrypt telegram bot token: %w", err)
		}
		config.BotToken = token
	}
	if setting, ok := settings[telegramChatIDKey]; ok && setting.Value != nil {
		config.ChatID = strings.TrimSpace(*setting.Value)
	}
	if setting, ok := settings[telegramThreadIDKey]; ok && setting.Value != nil && strings.TrimSpace(*setting.Value) != "" {
		id, err := strconv.ParseInt(strings.TrimSpace(*setting.Value), 10, 64)
		if err != nil || id <= 0 || id > maxMessageThreadID {
			return TelegramConfig{}, fmt.Errorf("stored telegram topic id is invalid")
		}
		config.MessageThreadID = id
	}
	config.Configured = config.BotToken != "" && config.ChatID != ""
	return config, nil
}

func (s *Service) AdminTestCredential(ctx context.Context) (string, error) {
	setting, found, err := repository.GetAppSetting(ctx, s.db, apiKeySettingKey)
	if err != nil {
		return "", err
	}
	if !found || setting.Value == nil || strings.TrimSpace(*setting.Value) == "" {
		return "", nil
	}
	return s.decrypt(*setting.Value)
}

func (s *Service) RequestTimeout(ctx context.Context) (time.Duration, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return 0, err
	}
	return time.Duration(settings.RequestTimeoutSeconds) * time.Second, nil
}

func (s *Service) upsertOptional(ctx context.Context, key, value, valueType string) error {
	if value == "" {
		return s.upsert(ctx, key, nil, valueType)
	}
	return s.upsert(ctx, key, &value, valueType)
}

// validateBotToken accepts the Bot API `<numeric id>:<secret>` shape.
func validateBotToken(token string) error {
	if len(token) < minBotTokenLength || len(token) > maxBotTokenLength {
		return fmt.Errorf("bot token must be between %d and %d characters", minBotTokenLength, maxBotTokenLength)
	}
	if strings.Count(token, ":") != 1 {
		return errors.New("bot token must contain exactly one ':'")
	}
	parts := strings.SplitN(token, ":", 2)
	if parts[0] == "" || parts[1] == "" {
		return errors.New("bot token is malformed")
	}
	for _, r := range token {
		if r <= 0x20 || r == 0x7f {
			return errors.New("bot token must not contain whitespace or control characters")
		}
	}
	for _, r := range parts[0] {
		if r < '0' || r > '9' {
			return errors.New("bot token id must be numeric")
		}
	}
	return nil
}

// validateChatID accepts a numeric chat id (optionally negative) or @username.
func validateChatID(chatID string) error {
	if len(chatID) > maxChatIDLength {
		return fmt.Errorf("chat id must be at most %d characters", maxChatIDLength)
	}
	if username, ok := strings.CutPrefix(chatID, "@"); ok {
		if len(username) < 5 || len(username) > 32 {
			return errors.New("chat username must be between 5 and 32 characters")
		}
		for _, r := range username {
			if r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				continue
			}
			return errors.New("chat username contains invalid characters")
		}
		return nil
	}
	digits := strings.TrimPrefix(chatID, "-")
	if digits == "" || len(digits) > 20 {
		return errors.New("chat id must be a numeric id or @username")
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return errors.New("chat id must be a numeric id or @username")
		}
	}
	return nil
}

// validateMessageThreadID accepts a positive Telegram forum topic id.
func validateMessageThreadID(id int64) error {
	if id <= 0 || id > maxMessageThreadID {
		return errors.New("topic id must be a positive integer")
	}
	return nil
}

func (s *Service) upsert(ctx context.Context, key string, value *string, valueType string) error {
	now := time.Now()
	_, err := repository.UpsertAppSetting(ctx, s.db, entities.AppSetting{SettingKey: key, Value: value, ValueType: valueType, CreatedAt: now, UpdatedAt: now})
	return err
}

func (s *Service) cipher() (cipher.AEAD, error) {
	if s == nil || s.protectionKey == "" {
		return nil, ErrProtectionKeyRequired
	}
	digest := sha256.Sum256([]byte("cpa-usage-keeper/admin-model-test/" + s.protectionKey))
	block, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func (s *Service) encrypt(value string) (string, error) {
	aead, err := s.cipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (s *Service) decrypt(value string) (string, error) {
	aead, err := s.cipher()
	if err != nil {
		return "", err
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil || len(data) < aead.NonceSize() {
		return "", fmt.Errorf("invalid encrypted setting")
	}
	nonce, ciphertext := data[:aead.NonceSize()], data[aead.NonceSize():]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted setting")
	}
	return string(plaintext), nil
}
