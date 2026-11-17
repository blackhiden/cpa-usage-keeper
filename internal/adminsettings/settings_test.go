package adminsettings

import (
	"context"
	"path/filepath"
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

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := repository.OpenDatabase(config.Config{SQLitePath: filepath.Join(t.TempDir(), "admin-settings.db")})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
