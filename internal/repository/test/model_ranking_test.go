package test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"

	"gorm.io/gorm"
)

func openModelRankingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "modelranking.db")})
	if err != nil {
		t.Fatalf("open database pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatalf("load reader pool: %v", err)
	}
	t.Cleanup(func() {
		_ = readerSQL.Close()
	})
	return db
}

func createModelRankingEvent(t *testing.T, db *gorm.DB, event entities.UsageEvent) {
	t.Helper()
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}
}

func modelRankingAlias(value string) *string { return &value }

func TestListModelRankingRowsCollapsesAliasAcrossProviders(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	ttftOne, ttftTwo := int64(100), int64(200)
	createModelRankingEvent(t, db, entities.UsageEvent{
		EventKey: "alias-1", Model: "openai/gpt-5", ModelAlias: modelRankingAlias("gpt-5"), Provider: "openai",
		Timestamp: now, Failed: false, TotalTokens: 100, InputTokens: 30, CacheReadTokens: 10,
		TTFTMS: &ttftOne, LatencyMS: 1000,
	})
	createModelRankingEvent(t, db, entities.UsageEvent{
		EventKey: "alias-2", Model: "azure/gpt-5", ModelAlias: modelRankingAlias("gpt-5"), Provider: "azure",
		Timestamp: now.Add(time.Minute), Failed: false, TotalTokens: 50, InputTokens: 20, CacheReadTokens: 5,
		TTFTMS: &ttftTwo, LatencyMS: 2000,
	})

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{
		Start: now.Add(-time.Hour), End: now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1: %+v", len(rows), rows)
	}
	row := rows[0]
	if row.Model != "gpt-5" {
		t.Fatalf("model = %q, want gpt-5", row.Model)
	}
	if row.RequestCount != 2 || row.SuccessCount != 2 || row.FailureCount != 0 {
		t.Fatalf("counters mismatch: %+v", row)
	}
	if row.TotalTokens != 150 || row.InputTokens != 50 || row.CacheReadTokens != 15 {
		t.Fatalf("token totals mismatch: %+v", row)
	}
	if row.TTFTSumMS != 300 || row.TTFTSampleCount != 2 || row.LatencySumMS != 3000 || row.LatencySampleCount != 2 {
		t.Fatalf("latency totals mismatch: %+v", row)
	}
	if row.Peak5MRequestCount != 2 || row.Peak5MTotalTokens != 150 {
		t.Fatalf("peak mismatch: %+v", row)
	}
	if len(row.Providers) != 2 || row.Providers[0] != "azure" || row.Providers[1] != "openai" {
		t.Fatalf("providers = %v, want [azure openai]", row.Providers)
	}
}

func TestListModelRankingRowsFallsBackToRealModelName(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	blank := "   "
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "fallback-1", Model: "claude-sonnet", Timestamp: now, TotalTokens: 10})
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "fallback-2", Model: "gpt-5", ModelAlias: &blank, Timestamp: now, TotalTokens: 20})

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 2 || rows[0].Model != "claude-sonnet" || rows[1].Model != "gpt-5" {
		t.Fatalf("fallback rows = %+v, want claude-sonnet then gpt-5", rows)
	}
}

func TestListModelRankingRowsWindowIsHalfOpen(t *testing.T) {
	db := openModelRankingTestDB(t)
	start := time.Date(2026, 11, 16, 0, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "edge-start", Model: "at-start", Timestamp: start})
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "edge-end", Model: "at-end", Timestamp: end})
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "edge-before", Model: "before", Timestamp: start.Add(-time.Second)})

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: start, End: end})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1 || rows[0].Model != "at-start" {
		t.Fatalf("half-open window rows = %+v, want only at-start", rows)
	}
}

func TestListModelRankingRowsSelectsFiveMinutePeak(t *testing.T) {
	db := openModelRankingTestDB(t)
	base := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "peak-1", Model: "m", Timestamp: base, TotalTokens: 10})
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "peak-2", Model: "m", Timestamp: base.Add(6 * time.Minute), TotalTokens: 40})
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "peak-3", Model: "m", Timestamp: base.Add(7 * time.Minute), TotalTokens: 5})

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: base.Add(-time.Hour), End: base.Add(time.Hour)})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].Peak5MRequestCount != 2 || rows[0].Peak5MTotalTokens != 45 {
		t.Fatalf("peak = %d requests / %d tokens, want 2 / 45", rows[0].Peak5MRequestCount, rows[0].Peak5MTotalTokens)
	}
}

func TestListModelRankingRowsIncludesNonGenerationTraffic(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	generate := false
	createModelRankingEvent(t, db, entities.UsageEvent{EventKey: "embedding", Model: "text-embedding-3", Generate: &generate, Timestamp: now, TotalTokens: 7})

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1 || rows[0].Model != "text-embedding-3" || rows[0].RequestCount != 1 {
		t.Fatalf("non-generation rows = %+v, want the embedding model counted", rows)
	}
}

func TestListModelRankingRowsEmptyWindow(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: now, End: now})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("empty window returned %d rows", len(rows))
	}
}

func TestListModelRankingRowsNormalizesAndCapsProviders(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	providers := []string{" zeta ", "alpha", "alpha"}
	for index := 0; index < 10; index++ {
		providers = append(providers, fmt.Sprintf("p%02d", index))
	}
	for index, provider := range providers {
		createModelRankingEvent(t, db, entities.UsageEvent{
			EventKey: fmt.Sprintf("provider-%d", index), Model: "m", ModelAlias: modelRankingAlias("shared"),
			Provider: provider, Timestamp: now, TotalTokens: 1,
		})
	}

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	got := rows[0].Providers
	if len(got) != 10 {
		t.Fatalf("providers length = %d, want 10: %v", len(got), got)
	}
	if got[0] != "alpha" || got[1] != "p00" || got[9] != "p08" {
		t.Fatalf("providers = %v, want trimmed, de-duplicated, sorted, capped", got)
	}
}

func TestListModelRankingRowsCandidateCap(t *testing.T) {
	db := openModelRankingTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	events := make([]entities.UsageEvent, 0, 1005)
	for index := 0; index < 1005; index++ {
		events = append(events, entities.UsageEvent{
			EventKey: fmt.Sprintf("cap-%04d", index), Model: fmt.Sprintf("m%04d", index), Timestamp: now, TotalTokens: 1,
		})
	}
	if err := db.CreateInBatches(events, 250).Error; err != nil {
		t.Fatalf("create capped events: %v", err)
	}

	rows, err := repository.ListModelRankingRows(context.Background(), db, dto.ModelRankingWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)})
	if err != nil {
		t.Fatalf("ListModelRankingRows: %v", err)
	}
	if len(rows) != 1000 {
		t.Fatalf("candidate rows = %d, want 1000", len(rows))
	}
	if rows[0].Model != "m0000" || rows[999].Model != "m0999" {
		t.Fatalf("candidate ordering mismatch: first=%q last=%q", rows[0].Model, rows[999].Model)
	}
}