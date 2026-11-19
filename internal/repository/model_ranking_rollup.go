package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"
)

const (
	modelRankingRollupBatchSize  = 1000
	modelRankingRollupInsertSize = 100
)

const modelRankingEventProjectionColumns = "id, api_group_key, provider, model, model_alias, timestamp, failed, generate, latency_ms, ttft_ms, input_tokens, cache_read_tokens, total_tokens, cost_usd, cost_available"

func EnsureModelRankingRollupCheckpoint(ctx context.Context, db *gorm.DB, now time.Time) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	row := entities.ModelRankingAggregationCheckpoint{Name: "model_ranking", CreatedAt: timeutil.NormalizeStorageTime(now), UpdatedAt: timeutil.NormalizeStorageTime(now)}
	if err := db.Clauses(dbresolver.Write).WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return fmt.Errorf("ensure model ranking checkpoint: %w", err)
	}
	return nil
}

type ModelRankingRollupRow struct {
	Model                string
	Provider             string
	BucketKey            int64
	RequestCount         int64
	SuccessCount         int64
	FailureCount         int64
	InputTokens          int64
	CacheReadTokens      int64
	TotalTokens          int64
	TTFTSumMS            int64
	TTFTSampleCount      int64
	LatencySumMS         int64
	LatencySampleCount   int64
	CostUSD              float64
	UnavailableCostCount int64
	Peak5MRequestCount   int64
	Peak5MTotalTokens    int64
}

type modelRankingRollupKey struct {
	kind, period, scope, model, provider string
	bucket                               int64
}

type modelRankingRollupAccum struct {
	row entities.ModelRankingPeriodStat
}

func ModelRankingRollupCheckpoint(ctx context.Context, db *gorm.DB) (int64, error) {
	if db == nil {
		return 0, fmt.Errorf("database is nil")
	}
	var checkpoint entities.ModelRankingAggregationCheckpoint
	err := db.Clauses(dbresolver.Read).WithContext(ctx).Where("name = ?", "model_ranking").Take(&checkpoint).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return 0, fmt.Errorf("load model ranking checkpoint: %w", err)
	}
	return checkpoint.LastAggregatedUsageEventID, nil
}

// AggregateModelRankingPage commits all model rollup rows and the matching
// cursor together. Replaying a page after failure cannot double count it.
func AggregateModelRankingPage(ctx context.Context, db *gorm.DB, now time.Time) (int, bool, error) {
	if db == nil {
		return 0, false, fmt.Errorf("database is nil")
	}
	if err := EnsureModelRankingRollupCheckpoint(ctx, db, now); err != nil {
		return 0, false, err
	}
	var rows []entities.UsageEvent
	var checkpoint entities.ModelRankingAggregationCheckpoint
	if err := db.Clauses(dbresolver.Read).WithContext(ctx).Where("name = ?", "model_ranking").Take(&checkpoint).Error; err != nil {
		return 0, false, fmt.Errorf("load model ranking checkpoint: %w", err)
	}
	if err := db.Clauses(dbresolver.Read).WithContext(ctx).Model(&entities.UsageEvent{}).
		Select(modelRankingEventProjectionColumns).
		Where("id > ?", checkpoint.LastAggregatedUsageEventID).
		Order("id ASC").Limit(modelRankingRollupBatchSize).Find(&rows).Error; err != nil {
		return 0, false, fmt.Errorf("load model ranking aggregation page: %w", err)
	}
	if len(rows) == 0 {
		return 0, true, nil
	}
	next := rows[len(rows)-1].ID
	if err := ApplyModelRankingAggregationPage(ctx, db, checkpoint.LastAggregatedUsageEventID, next, rows, now); err != nil {
		return 0, false, err
	}
	return len(rows), len(rows) < modelRankingRollupBatchSize, nil
}

func ApplyModelRankingAggregationPage(ctx context.Context, db *gorm.DB, expectedCursor, nextCursor int64, events []entities.UsageEvent, now time.Time) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if nextCursor <= expectedCursor {
		return fmt.Errorf("invalid model ranking cursor advance %d -> %d", expectedCursor, nextCursor)
	}
	prepared := buildModelRankingRollupRows(events, timeutil.NormalizeStorageTime(now))
	return ApplyModelRankingRollupRows(ctx, db, expectedCursor, nextCursor, prepared, now)
}

func LoadModelRankingAggregationCursor(ctx context.Context, db *gorm.DB) (int64, error) {
	if err := EnsureModelRankingRollupCheckpoint(ctx, db, time.Now()); err != nil {
		return 0, err
	}
	var row entities.ModelRankingAggregationCheckpoint
	if err := db.Clauses(dbresolver.Read).WithContext(ctx).Where("name = ?", "model_ranking").Take(&row).Error; err != nil {
		return 0, fmt.Errorf("load model ranking checkpoint: %w", err)
	}
	return row.LastAggregatedUsageEventID, nil
}

func ApplyModelRankingRollupRows(ctx context.Context, db *gorm.DB, expectedCursor, nextCursor int64, prepared []entities.ModelRankingPeriodStat, now time.Time) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if err := EnsureModelRankingRollupCheckpoint(ctx, db, now); err != nil {
		return err
	}
	if err := db.Clauses(dbresolver.Write).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for start := 0; start < len(prepared); start += modelRankingRollupInsertSize {
			end := min(start+modelRankingRollupInsertSize, len(prepared))
			batch := prepared[start:end]
			updates := map[string]any{
				"request_count":          gorm.Expr("request_count + excluded.request_count"),
				"success_count":          gorm.Expr("success_count + excluded.success_count"),
				"failure_count":          gorm.Expr("failure_count + excluded.failure_count"),
				"input_tokens":           gorm.Expr("input_tokens + excluded.input_tokens"),
				"cache_read_tokens":      gorm.Expr("cache_read_tokens + excluded.cache_read_tokens"),
				"total_tokens":           gorm.Expr("total_tokens + excluded.total_tokens"),
				"ttft_sum_ms":            gorm.Expr("ttft_sum_ms + excluded.ttft_sum_ms"),
				"ttft_sample_count":      gorm.Expr("ttft_sample_count + excluded.ttft_sample_count"),
				"latency_sum_ms":         gorm.Expr("latency_sum_ms + excluded.latency_sum_ms"),
				"latency_sample_count":   gorm.Expr("latency_sample_count + excluded.latency_sample_count"),
				"cost_usd":               gorm.Expr("cost_usd + excluded.cost_usd"),
				"unavailable_cost_count": gorm.Expr("unavailable_cost_count + excluded.unavailable_cost_count"),
				"updated_at":             timeutil.FormatStorageTime(now),
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "period_kind"}, {Name: "period_key"}, {Name: "scope_key"}, {Name: "model"}, {Name: "provider"}, {Name: "bucket_key"}},
				DoUpdates: clause.Assignments(updates),
			}).Create(&batch).Error; err != nil {
				return fmt.Errorf("upsert model ranking rollups: %w", err)
			}
		}
		result := tx.Model(&entities.ModelRankingAggregationCheckpoint{}).
			Where("name = ? AND last_aggregated_usage_event_id = ?", "model_ranking", expectedCursor).
			Updates(map[string]any{"last_aggregated_usage_event_id": nextCursor, "updated_at": timeutil.FormatStorageTime(now)})
		if result.Error != nil {
			return fmt.Errorf("advance model ranking checkpoint: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("model ranking checkpoint cursor conflict")
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func buildModelRankingRollupRows(events []entities.UsageEvent, now time.Time) []entities.ModelRankingPeriodStat {
	values := make(map[modelRankingRollupKey]*modelRankingRollupAccum)
	for _, event := range events {
		model := strings.TrimSpace(event.Model)
		if event.ModelAlias != nil && strings.TrimSpace(*event.ModelAlias) != "" {
			model = strings.TrimSpace(*event.ModelAlias)
		}
		if model == "" {
			continue
		}
		provider := strings.TrimSpace(event.Provider)
		local := event.Timestamp.In(time.Local)
		day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
		week := day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
		month := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, time.Local)
		periods := []struct{ kind, key string }{{"day", day.Format("2006-01-02")}, {"week", isoModelWeekKey(week)}, {"month", month.Format("2006-01")}}
		scope := strings.TrimSpace(event.APIGroupKey)
		bucket := event.Timestamp.Unix() / 300
		for _, period := range periods {
			k := modelRankingRollupKey{period.kind, period.key, "", model, provider, bucket}
			addModelRankingRollup(values, k, event, now)
			if scope != "" {
				k.scope = scope
				addModelRankingRollup(values, k, event, now)
			}
		}
	}
	out := make([]entities.ModelRankingPeriodStat, 0, len(values))
	for _, value := range values {
		out = append(out, value.row)
	}
	return out
}

func addModelRankingRollup(values map[modelRankingRollupKey]*modelRankingRollupAccum, key modelRankingRollupKey, event entities.UsageEvent, now time.Time) {
	entry := values[key]
	if entry == nil {
		entry = &modelRankingRollupAccum{row: entities.ModelRankingPeriodStat{PeriodKind: key.kind, PeriodKey: key.period, ScopeKey: key.scope, Model: key.model, Provider: key.provider, BucketKey: key.bucket, CreatedAt: now, UpdatedAt: now}}
		values[key] = entry
	}
	r := &entry.row
	r.RequestCount++
	if event.Failed {
		r.FailureCount++
	} else {
		r.SuccessCount++
	}
	r.InputTokens += event.InputTokens
	r.CacheReadTokens += event.CacheReadTokens
	r.TotalTokens += event.TotalTokens
	if !event.Failed && event.TTFTMS != nil && *event.TTFTMS > 0 && event.LatencyMS > 0 {
		r.TTFTSumMS += *event.TTFTMS
		r.TTFTSampleCount++
		r.LatencySumMS += event.LatencyMS
		r.LatencySampleCount++
	}
	if event.CostUSD != nil && event.CostAvailable != nil && *event.CostAvailable {
		r.CostUSD += *event.CostUSD
	} else {
		r.UnavailableCostCount++
	}
}

func ListModelRankingRollups(ctx context.Context, db *gorm.DB, kind, periodKey, scope, provider string) ([]ModelRankingRollupRow, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	var rows []ModelRankingRollupRow
	query := db.Clauses(dbresolver.Read).WithContext(ctx).Model(&entities.ModelRankingPeriodStat{}).
		Select("model, provider, bucket_key, request_count, success_count, failure_count, input_tokens, cache_read_tokens, total_tokens, ttft_sum_ms, ttft_sample_count, latency_sum_ms, latency_sample_count, cost_usd, unavailable_cost_count").
		Where("period_kind = ? AND period_key = ? AND scope_key = ?", kind, periodKey, scope)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.Order("model ASC, bucket_key ASC, provider ASC").Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list model ranking rollups: %w", err)
	}
	return rows, nil
}

func ModelRankingScopeForAPIKeyID(apiKeyID int64, apiGroupKey string) (string, error) {
	if apiKeyID <= 0 || strings.TrimSpace(apiGroupKey) == "" {
		return "", fmt.Errorf("invalid API key scope")
	}
	return strings.TrimSpace(apiGroupKey), nil
}

func isoModelWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
