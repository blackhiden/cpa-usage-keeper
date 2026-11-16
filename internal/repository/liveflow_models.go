package repository

import (
	"context"
	"fmt"

	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

// liveFlowModelLimit bounds the per-request aggregation so a month of events
// never returns an unbounded model list.
const liveFlowModelLimit = 60

// ListLiveFlowModels aggregates generation usage events grouped by model for
// the half-open [window.Start, window.End) window. It reads from the reader
// pool, filters to generation traffic (matching the Live Flow relay contract),
// excludes empty model names, and returns at most liveFlowModelLimit rows
// ordered by request count descending then model ascending.
func ListLiveFlowModels(ctx context.Context, db *gorm.DB, window dto.LiveFlowModelWindow) ([]dto.LiveFlowModel, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	if window.End.Before(window.Start) || window.End.Equal(window.Start) {
		return []dto.LiveFlowModel{}, nil
	}

	start := timeutil.FormatStorageTime(window.Start)
	end := timeutil.FormatStorageTime(window.End)

	rows := make([]dto.LiveFlowModel, 0, liveFlowModelLimit)
	type aggregateRow struct {
		Model        string
		RequestCount int64
		SuccessCount int64
		FailureCount int64
		TotalTokens  int64
		LastSeenAt   string
	}
	var aggregates []aggregateRow
	// Group by the display name: the alias when present, otherwise the real
	// model. Different real names that share one alias collapse into a single
	// block instead of appearing once per prefix.
	displayName := "COALESCE(NULLIF(TRIM(event.model_alias), ''), event.model)"
	query := db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select(displayName+" AS model, COUNT(event.id) AS request_count, "+
			"SUM(CASE WHEN event.failed THEN 0 ELSE 1 END) AS success_count, "+
			"SUM(CASE WHEN event.failed THEN 1 ELSE 0 END) AS failure_count, "+
			"SUM(event.total_tokens) AS total_tokens, MAX(event.timestamp) AS last_seen_at").
		Where("event.timestamp >= ? AND event.timestamp < ?", start, end).
		Where("event.model <> ''").
		Where("event.generate IS NULL OR event.generate = ?", true).
		Group(displayName).
		Order("request_count DESC").
		Order(displayName + " ASC").
		Limit(liveFlowModelLimit)

	if err := query.Scan(&aggregates).Error; err != nil {
		return nil, fmt.Errorf("list live flow models: %w", err)
	}
	for _, aggregate := range aggregates {
		lastSeen, err := timeutil.ParseStorageTime(aggregate.LastSeenAt)
		if err != nil {
			return nil, fmt.Errorf("parse live flow model last seen %q: %w", aggregate.LastSeenAt, err)
		}
		rows = append(rows, dto.LiveFlowModel{
			Model:        aggregate.Model,
			RequestCount: aggregate.RequestCount,
			SuccessCount: aggregate.SuccessCount,
			FailureCount: aggregate.FailureCount,
			TotalTokens:  aggregate.TotalTokens,
			LastSeenAt:   lastSeen,
		})
	}

	if err := fillLiveFlowModelSources(ctx, rows, db, start, end, displayName); err != nil {
		return nil, err
	}
	if err := fillLiveFlowModelLatest(ctx, rows, db, start, end); err != nil {
		return nil, err
	}
	return rows, nil
}

// fillLiveFlowModelSources records which real model names are grouped under each
// display name so the live stream can be mapped onto the same blocks.
func fillLiveFlowModelSources(ctx context.Context, rows []dto.LiveFlowModel, db *gorm.DB, start, end, displayName string) error {
	if len(rows) == 0 {
		return nil
	}
	type sourceRow struct {
		Model  string
		Source string
	}
	var sources []sourceRow
	query := db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select(displayName + " AS model, event.model AS source").
		Where("event.timestamp >= ? AND event.timestamp < ?", start, end).
		Where("event.model <> ''").
		Where("event.generate IS NULL OR event.generate = ?", true).
		Group(displayName + ", event.model").
		Order(displayName + " ASC").
		Order("event.model ASC")
	if err := query.Scan(&sources).Error; err != nil {
		return fmt.Errorf("load live flow model sources: %w", err)
	}

	byDisplay := make(map[string][]string, len(rows))
	for _, source := range sources {
		byDisplay[source.Model] = append(byDisplay[source.Model], source.Source)
	}
	for index := range rows {
		rows[index].SourceModels = byDisplay[rows[index].Model]
	}
	return nil
}

// fillLiveFlowModelLatest attaches the newest event's latency and status to
// each aggregated model. SQLite cannot express argmax portably, so the latest
// event per returned model is fetched in one bounded query and resolved in Go.
func fillLiveFlowModelLatest(ctx context.Context, rows []dto.LiveFlowModel, db *gorm.DB, start, end string) error {
	if len(rows) == 0 {
		return nil
	}
	models := make([]string, 0, len(rows))
	for _, row := range rows {
		models = append(models, row.Model)
	}

	type latestRow struct {
		Model      string
		LatencyMS  int64
		StatusCode *int
	}
	var latest []latestRow
	query := db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select("event.model AS model, event.latency_ms AS latency_ms, event.status_code AS status_code").
		Where("event.model IN ?", models).
		Where("event.timestamp >= ? AND event.timestamp < ?", start, end).
		Where("event.generate IS NULL OR event.generate = ?", true).
		Order("event.timestamp DESC").
		Order("event.id DESC")
	if err := query.Scan(&latest).Error; err != nil {
		return fmt.Errorf("load latest live flow model events: %w", err)
	}

	seen := make(map[string]bool, len(models))
	for _, row := range latest {
		if seen[row.Model] {
			continue
		}
		seen[row.Model] = true
		for index := range rows {
			if rows[index].Model == row.Model {
				rows[index].LastLatencyMS = row.LatencyMS
				if row.StatusCode != nil {
					rows[index].LastStatus = *row.StatusCode
				}
				break
			}
		}
	}
	return nil
}
