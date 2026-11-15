package test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"

	"gorm.io/gorm"
)

func openLiveFlowTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "liveflow.db")})
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

func createLiveFlowEvent(t *testing.T, db *gorm.DB, event entities.UsageEvent) {
	t.Helper()
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}
}

func TestListLiveFlowModelsAggregatesByModel(t *testing.T) {
	db := openLiveFlowTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	generate := true
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "a1", Model: "gpt-5", Timestamp: now, Failed: false, Generate: &generate, TotalTokens: 100, LatencyMS: 10})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "a2", Model: "gpt-5", Timestamp: now.Add(time.Minute), Failed: true, Generate: &generate, TotalTokens: 50, LatencyMS: 20})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "a3", Model: "claude", Timestamp: now.Add(2 * time.Minute), Failed: false, Generate: &generate, TotalTokens: 30, LatencyMS: 5})

	window := dto.LiveFlowModelWindow{Start: now.Add(-time.Hour), End: now.Add(3 * time.Hour)}
	rows, err := repository.ListLiveFlowModels(context.Background(), db, window)
	if err != nil {
		t.Fatalf("ListLiveFlowModels: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d models, want 2", len(rows))
	}
	// Ordered by request_count DESC, model ASC.
	if rows[0].Model != "gpt-5" || rows[1].Model != "claude" {
		t.Fatalf("unexpected order: %+v", rows)
	}
	if rows[0].RequestCount != 2 || rows[0].SuccessCount != 1 || rows[0].FailureCount != 1 || rows[0].TotalTokens != 150 {
		t.Fatalf("gpt-5 aggregate mismatch: %+v", rows[0])
	}
	if rows[0].LastLatencyMS != 20 || rows[0].LastStatus != 0 {
		t.Fatalf("gpt-5 latest mismatch: %+v", rows[0])
	}
}

func TestListLiveFlowModelsWindowEdgesAreHalfOpen(t *testing.T) {
	db := openLiveFlowTestDB(t)
	generate := true
	start := time.Date(2026, 11, 16, 0, 0, 0, 0, time.Local)
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "at-start", Model: "m1", Timestamp: start, Generate: &generate})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "at-end", Model: "m2", Timestamp: start.Add(24 * time.Hour), Generate: &generate})

	window := dto.LiveFlowModelWindow{Start: start, End: start.Add(24 * time.Hour)}
	rows, err := repository.ListLiveFlowModels(context.Background(), db, window)
	if err != nil {
		t.Fatalf("ListLiveFlowModels: %v", err)
	}
	if len(rows) != 1 || rows[0].Model != "m1" {
		t.Fatalf("half-open window mismatch: %+v", rows)
	}
}

func TestListLiveFlowModelsFiltersGenerationAndEmptyModel(t *testing.T) {
	db := openLiveFlowTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	generate := true
	nonGenerate := false
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "gen", Model: "gen-model", Timestamp: now, Generate: &generate})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "non-gen", Model: "non-gen-model", Timestamp: now, Generate: &nonGenerate})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "nil-gen", Model: "nil-gen-model", Timestamp: now})
	createLiveFlowEvent(t, db, entities.UsageEvent{EventKey: "empty", Model: "", Timestamp: now, Generate: &generate})

	window := dto.LiveFlowModelWindow{Start: now.Add(-time.Hour), End: now.Add(time.Hour)}
	rows, err := repository.ListLiveFlowModels(context.Background(), db, window)
	if err != nil {
		t.Fatalf("ListLiveFlowModels: %v", err)
	}
	models := map[string]bool{}
	for _, row := range rows {
		models[row.Model] = true
	}
	if !models["gen-model"] || !models["nil-gen-model"] {
		t.Fatalf("generation models missing: %+v", rows)
	}
	if models["non-gen-model"] || models[""] {
		t.Fatalf("non-generation or empty model included: %+v", rows)
	}
}

func TestListLiveFlowModelsEmptyWindowReturnsEmpty(t *testing.T) {
	db := openLiveFlowTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	rows, err := repository.ListLiveFlowModels(context.Background(), db, dto.LiveFlowModelWindow{Start: now, End: now})
	if err != nil {
		t.Fatalf("ListLiveFlowModels: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("expected empty result, got %+v", rows)
	}
}
