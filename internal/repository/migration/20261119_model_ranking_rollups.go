package migration

import (
	"fmt"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func modelRankingRollupsMigration(tx *gorm.DB) error {
	if tx == nil {
		return fmt.Errorf("database is nil")
	}
	if err := tx.AutoMigrate(&entities.ModelRankingPeriodStat{}, &entities.ModelRankingAggregationCheckpoint{}); err != nil {
		return fmt.Errorf("create model ranking rollup schema: %w", err)
	}
	now := timeutil.NormalizeStorageTime(time.Now())
	checkpoint := entities.ModelRankingAggregationCheckpoint{Name: "model_ranking", CreatedAt: now, UpdatedAt: now}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&checkpoint).Error; err != nil {
		return fmt.Errorf("create model ranking checkpoint: %w", err)
	}
	return nil
}
