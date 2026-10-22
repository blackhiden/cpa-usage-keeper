package migration

import (
	"fmt"

	"cpa-usage-keeper/internal/entities"
	"gorm.io/gorm"
)

func restoreLocalRankingMigration(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("database is nil")
	}
	if err := tx.AutoMigrate(&entities.LocalRankingPeriodStat{}); err != nil {
		return fmt.Errorf("restore local ranking storage: %w", err)
	}
	if tx.Migrator().HasColumn(&entities.CPAAPIKey{}, "LocalRankingAvatarID") {
		return nil
	}
	if err := tx.Migrator().AddColumn(&entities.CPAAPIKey{}, "LocalRankingAvatarID"); err != nil {
		return fmt.Errorf("restore CPA API key local ranking avatar: %w", err)
	}
	return nil
}
