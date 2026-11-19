package entities

import "time"

// ModelRankingPeriodStat stores one provider contribution per five-minute bucket.
type ModelRankingPeriodStat struct {
	ID                   int64     `gorm:"primaryKey"`
	PeriodKind           string    `gorm:"type:text;not null;uniqueIndex:uniq_model_ranking_period,priority:1;index:idx_model_ranking_period_scope,priority:1"`
	PeriodKey            string    `gorm:"type:text;not null;uniqueIndex:uniq_model_ranking_period,priority:2;index:idx_model_ranking_period_scope,priority:2"`
	ScopeKey             string    `gorm:"type:text;not null;default:'';uniqueIndex:uniq_model_ranking_period,priority:3;index:idx_model_ranking_period_scope,priority:3"`
	Model                string    `gorm:"type:text;not null;uniqueIndex:uniq_model_ranking_period,priority:4"`
	Provider             string    `gorm:"type:text;not null;default:'';uniqueIndex:uniq_model_ranking_period,priority:5"`
	BucketKey            int64     `gorm:"not null;uniqueIndex:uniq_model_ranking_period,priority:6"`
	RequestCount         int64     `gorm:"not null;default:0"`
	SuccessCount         int64     `gorm:"not null;default:0"`
	FailureCount         int64     `gorm:"not null;default:0"`
	InputTokens          int64     `gorm:"not null;default:0"`
	CacheReadTokens      int64     `gorm:"not null;default:0"`
	TotalTokens          int64     `gorm:"not null;default:0"`
	TTFTSumMS            int64     `gorm:"not null;default:0"`
	TTFTSampleCount      int64     `gorm:"not null;default:0"`
	LatencySumMS         int64     `gorm:"not null;default:0"`
	LatencySampleCount   int64     `gorm:"not null;default:0"`
	CostUSD              float64   `gorm:"column:cost_usd;type:real;not null;default:0"`
	UnavailableCostCount int64     `gorm:"not null;default:0"`
	CreatedAt            time.Time `gorm:"serializer:storageTime;not null"`
	UpdatedAt            time.Time `gorm:"serializer:storageTime;not null"`
}

func (ModelRankingPeriodStat) TableName() string { return "model_ranking_period_stats" }

type ModelRankingAggregationCheckpoint struct {
	Name                       string    `gorm:"type:text;primaryKey"`
	LastAggregatedUsageEventID int64     `gorm:"not null;default:0"`
	CreatedAt                  time.Time `gorm:"serializer:storageTime;not null"`
	UpdatedAt                  time.Time `gorm:"serializer:storageTime;not null"`
}

func (ModelRankingAggregationCheckpoint) TableName() string {
	return "model_ranking_aggregation_checkpoints"
}
