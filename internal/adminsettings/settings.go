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
	encryptedValueType    = "encrypted"
	integerValueType      = "int"
	defaultTimeoutSeconds = int64(30)
	minTimeoutSeconds     = int64(1)
	maxTimeoutSeconds     = int64(120)
)

var ErrProtectionKeyRequired = errors.New("admin settings protection key is required")

type Settings struct {
	APIKeyConfigured      bool  `json:"api_key_configured"`
	RequestTimeoutSeconds int64 `json:"request_timeout_seconds"`
}

type Update struct {
	APIKey                *string
	RequestTimeoutSeconds *int64
}

type Service struct {
	db            *gorm.DB
	protectionKey string
}

func NewService(db *gorm.DB, protectionKey string) *Service {
	return &Service{db: db, protectionKey: strings.TrimSpace(protectionKey)}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	settings, err := repository.GetAppSettings(ctx, s.db, []string{apiKeySettingKey, timeoutSettingKey})
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
	return result, nil
}

func (s *Service) Save(ctx context.Context, update Update) (Settings, error) {
	if update.RequestTimeoutSeconds != nil {
		seconds := *update.RequestTimeoutSeconds
		if seconds < minTimeoutSeconds || seconds > maxTimeoutSeconds {
			return Settings{}, fmt.Errorf("request timeout must be between %d and %d seconds", minTimeoutSeconds, maxTimeoutSeconds)
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
	return current, nil
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
