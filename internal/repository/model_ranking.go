package repository

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const (
	// modelRankingCandidateLimit bounds the candidate rows before the domain
	// ranks them. It is a safety bound only: every metric is computed from the
	// same candidate set, so a metric switch cannot change which models are
	// considered.
	modelRankingCandidateLimit = 1000
	// modelRankingProviderLimit bounds the provenance list per display name.
	modelRankingProviderLimit = 10
)

// modelRankingTimeRangePredicate mirrors rankingTimeRangePredicate in
// internal/ranking/aggregate.go. That helper is unexported in the ranking
// package, and importing the ranking domain would couple two protected feature
// domains, so the shape is intentionally duplicated: an index-friendly coarse
// text range widened by 24 hours, narrowed by the real epoch of every candidate
// row. Keep the two in sync if the storage timestamp format changes.
func modelRankingTimeRangePredicate(start, end time.Time) (string, []any) {
	// storageTime uses the Keeper local offset, so text order is not instant
	// order across a DST rollback. The coarse range still uses the timestamp
	// index; strftime only rechecks real time on candidate rows.
	coarseStart := timeutil.FormatStorageTime(start.Add(-24 * time.Hour))
	coarseEnd := timeutil.FormatStorageTime(end.Add(24 * time.Hour))
	epochExpression := "CAST(strftime('%s', timestamp) AS INTEGER)"
	if end.Nanosecond() == 0 {
		return "timestamp >= ? AND timestamp < ? AND " + epochExpression + " >= ? AND " + epochExpression + " < ?", []any{
			coarseStart, coarseEnd, start.Unix(), end.Unix(),
		}
	}
	return "timestamp >= ? AND timestamp < ? AND " + epochExpression + " >= ? AND (" + epochExpression + " < ? OR (" + epochExpression + " = ? AND timestamp < ?))", []any{
		coarseStart, coarseEnd, start.Unix(), end.Unix(), end.Unix(), timeutil.FormatStorageTime(end),
	}
}

// ListModelRankingRows aggregates usage events grouped by display name for the
// half-open [window.Start, window.End) window. The display name is the alias
// when present, otherwise the real model name, so several real names sharing an
// alias collapse into one row. It reads from the reader pool, includes every
// usage event (matching the API-key ranking), never queries
// usage_events_archive, and returns providers[] per row.
func ListModelRankingRows(ctx context.Context, db *gorm.DB, window dto.ModelRankingWindow) ([]dto.ModelRankingRow, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	if !window.End.After(window.Start) {
		return []dto.ModelRankingRow{}, nil
	}

	predicate, arguments := modelRankingTimeRangePredicate(window.Start, window.End)
	displayName := "COALESCE(NULLIF(TRIM(event.model_alias), ''), event.model)"
	query := fmt.Sprintf(`
WITH ranged_events AS (
	SELECT
		%s AS display_name,
		event.provider AS provider,
		event.timestamp AS timestamp,
		event.failed AS failed,
		event.ttft_ms AS ttft_ms,
		event.latency_ms AS latency_ms,
		event.input_tokens AS input_tokens,
		event.cache_read_tokens AS cache_read_tokens,
		event.total_tokens AS total_tokens
	FROM usage_events AS event
	WHERE %s AND TRIM(%s) <> ''
), five_minute_buckets AS (
	SELECT
		display_name,
		CAST(strftime('%%s', timestamp) AS INTEGER) / 300 AS bucket_key,
		COUNT(*) AS request_count,
		SUM(CASE WHEN failed = 0 THEN 1 ELSE 0 END) AS success_count,
		SUM(CASE WHEN failed <> 0 THEN 1 ELSE 0 END) AS failure_count,
		SUM(input_tokens) AS input_tokens,
		SUM(cache_read_tokens) AS cache_read_tokens,
		SUM(total_tokens) AS total_tokens,
		SUM(CASE WHEN failed = 0 AND ttft_ms IS NOT NULL AND ttft_ms > 0 AND latency_ms > 0 THEN ttft_ms ELSE 0 END) AS ttft_sum_ms,
		SUM(CASE WHEN failed = 0 AND ttft_ms IS NOT NULL AND ttft_ms > 0 AND latency_ms > 0 THEN 1 ELSE 0 END) AS ttft_sample_count,
		SUM(CASE WHEN failed = 0 AND ttft_ms IS NOT NULL AND ttft_ms > 0 AND latency_ms > 0 THEN latency_ms ELSE 0 END) AS latency_sum_ms,
		SUM(CASE WHEN failed = 0 AND ttft_ms IS NOT NULL AND ttft_ms > 0 AND latency_ms > 0 THEN 1 ELSE 0 END) AS latency_sample_count
	FROM ranged_events
	GROUP BY display_name, bucket_key
)
SELECT
	display_name AS model,
	COALESCE(SUM(request_count), 0) AS request_count,
	COALESCE(SUM(success_count), 0) AS success_count,
	COALESCE(SUM(failure_count), 0) AS failure_count,
	COALESCE(SUM(input_tokens), 0) AS input_tokens,
	COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
	COALESCE(SUM(total_tokens), 0) AS total_tokens,
	COALESCE(SUM(ttft_sum_ms), 0) AS ttft_sum_ms,
	COALESCE(SUM(ttft_sample_count), 0) AS ttft_sample_count,
	COALESCE(SUM(latency_sum_ms), 0) AS latency_sum_ms,
	COALESCE(SUM(latency_sample_count), 0) AS latency_sample_count,
	COALESCE(MAX(request_count), 0) AS peak_5m_request_count,
	COALESCE(MAX(total_tokens), 0) AS peak_5m_total_tokens
FROM five_minute_buckets
GROUP BY display_name
ORDER BY model ASC
LIMIT %d`, displayName, predicate, displayName, modelRankingCandidateLimit)

	type aggregateRow struct {
		Model              string `gorm:"column:model"`
		RequestCount       int64  `gorm:"column:request_count"`
		SuccessCount       int64  `gorm:"column:success_count"`
		FailureCount       int64  `gorm:"column:failure_count"`
		InputTokens        int64  `gorm:"column:input_tokens"`
		CacheReadTokens    int64  `gorm:"column:cache_read_tokens"`
		TotalTokens        int64  `gorm:"column:total_tokens"`
		TTFTSumMS          int64  `gorm:"column:ttft_sum_ms"`
		TTFTSampleCount    int64  `gorm:"column:ttft_sample_count"`
		LatencySumMS       int64  `gorm:"column:latency_sum_ms"`
		LatencySampleCount int64  `gorm:"column:latency_sample_count"`
		Peak5MRequestCount int64  `gorm:"column:peak_5m_request_count"`
		Peak5MTotalTokens  int64  `gorm:"column:peak_5m_total_tokens"`
	}
	var aggregates []aggregateRow
	if err := db.WithContext(ctx).Clauses(dbresolver.Read).Raw(query, arguments...).Scan(&aggregates).Error; err != nil {
		return nil, fmt.Errorf("aggregate model ranking window: %w", err)
	}
	if len(aggregates) == 0 {
		return []dto.ModelRankingRow{}, nil
	}

	rows := make([]dto.ModelRankingRow, 0, len(aggregates))
	for _, aggregate := range aggregates {
		rows = append(rows, dto.ModelRankingRow{
			Model:              aggregate.Model,
			RequestCount:       aggregate.RequestCount,
			SuccessCount:       aggregate.SuccessCount,
			FailureCount:       aggregate.FailureCount,
			InputTokens:        aggregate.InputTokens,
			CacheReadTokens:    aggregate.CacheReadTokens,
			TotalTokens:        aggregate.TotalTokens,
			TTFTSumMS:          aggregate.TTFTSumMS,
			TTFTSampleCount:    aggregate.TTFTSampleCount,
			LatencySumMS:       aggregate.LatencySumMS,
			LatencySampleCount: aggregate.LatencySampleCount,
			Peak5MRequestCount: aggregate.Peak5MRequestCount,
			Peak5MTotalTokens:  aggregate.Peak5MTotalTokens,
		})
	}

	if err := fillModelRankingProviders(ctx, rows, db, predicate, arguments, displayName); err != nil {
		return nil, err
	}
	return rows, nil
}

// fillModelRankingProviders records the distinct providers that contributed to
// each display name so the UI can explain a collapsed alias row without
// splitting the ranking.
func fillModelRankingProviders(ctx context.Context, rows []dto.ModelRankingRow, db *gorm.DB, predicate string, arguments []any, displayName string) error {
	if len(rows) == 0 {
		return nil
	}
	query := fmt.Sprintf(`
SELECT
	%s AS model,
	TRIM(event.provider) AS provider
FROM usage_events AS event
WHERE %s AND TRIM(%s) <> '' AND TRIM(event.provider) <> ''
GROUP BY model, TRIM(event.provider)
ORDER BY model ASC, provider ASC`, displayName, predicate, displayName)

	type providerRow struct {
		Model    string `gorm:"column:model"`
		Provider string `gorm:"column:provider"`
	}
	var providers []providerRow
	if err := db.WithContext(ctx).Clauses(dbresolver.Read).Raw(query, arguments...).Scan(&providers).Error; err != nil {
		return fmt.Errorf("load model ranking providers: %w", err)
	}

	byModel := make(map[string][]string, len(rows))
	for _, provider := range providers {
		name := strings.TrimSpace(provider.Provider)
		if name == "" {
			continue
		}
		byModel[provider.Model] = append(byModel[provider.Model], name)
	}
	for index := range rows {
		names := byModel[rows[index].Model]
		sort.Strings(names)
		unique := names[:0]
		for _, name := range names {
			if len(unique) == 0 || unique[len(unique)-1] != name {
				unique = append(unique, name)
			}
		}
		if len(unique) > modelRankingProviderLimit {
			unique = unique[:modelRankingProviderLimit]
		}
		rows[index].Providers = unique
	}
	return nil
}