package migration

import (
	"gorm.io/gorm"
)

// migrateAddCPAAPIKeyName 为 cpa_api_keys 表添加 key_name 列，存储从 CPA /v0/management/auth-api-keys 同步的 API Key 名称。
func migrateAddCPAAPIKeyName(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE cpa_api_keys ADD COLUMN key_name TEXT NOT NULL DEFAULT ''`).Error; err != nil {
			return err
		}
		return nil
	})
}
