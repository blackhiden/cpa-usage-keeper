package service

import (
	"fmt"
	"time"

	"cpa-usage-keeper/internal/cpa/dto/response"
	"cpa-usage-keeper/internal/repository"
	"gorm.io/gorm"
)

// syncManagementAPIKeys 同步 CPA 管理 API key 清单，优先使用带名称的 /v0/management/auth-api-keys，失败时回退到原 /v0/management/api-keys。
func syncManagementAPIKeys(db *gorm.DB, authResult *response.AuthAPIKeysResult, authErr error, legacyResult *response.ManagementAPIKeysResult, legacyErr error, now time.Time) error {
	// 优先尝试新的带名称的接口
	if authErr == nil && authResult != nil {
		if db == nil {
			return fmt.Errorf("database is nil")
		}
		// 转换为 repository 需要的格式
		items := make([]struct{ Key, Name string }, len(authResult.Payload))
		for i, item := range authResult.Payload {
			items[i] = struct{ Key, Name string }{Key: item.Key, Name: item.Name}
		}
		if err := repository.SyncCPAAPIKeysWithNames(db, items, now); err != nil {
			return fmt.Errorf("sync auth api keys: %w", err)
		}
		return nil
	}

	// 新接口失败时，回退到原接口
	if legacyErr != nil {
		return fmt.Errorf("fetch management api keys: %w", legacyErr)
	}
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if legacyResult == nil {
		return fmt.Errorf("fetch management api keys: empty response")
	}
	if err := repository.SyncCPAAPIKeys(db, legacyResult.Payload.APIKeys, now); err != nil {
		return fmt.Errorf("sync management api keys: %w", err)
	}
	return nil
}
