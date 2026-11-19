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
	"gorm.io/gorm"
)

func openModelRankingRollupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "model-ranking-rollup.db")})
	if err != nil {
		t.Fatalf("open pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = readerSQL.Close() })
	return db
}

func TestModelRankingRollupPageIsScopedAndReplaySafe(t *testing.T) {
	db := openModelRankingRollupDB(t)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.Local)
	alias := "shared-model"
	generate := true
	cost := 1.25
	available := true
	for i, event := range []entities.UsageEvent{
		{EventKey: "roll-a", APIGroupKey: "key-a", Model: "provider-a/model", ModelAlias: &alias, Provider: "provider-a", Timestamp: now, TotalTokens: 100, InputTokens: 50, CostUSD: &cost, CostAvailable: &available, Generate: &generate},
		{EventKey: "roll-b", APIGroupKey: "key-b", Model: "provider-b/model", ModelAlias: &alias, Provider: "provider-b", Timestamp: now.Add(time.Minute), TotalTokens: 200, InputTokens: 100, CostUSD: nil, CostAvailable: nil, Generate: &generate},
	} {
		event.ID = int64(i + 1)
		if err := db.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, caught, err := repository.AggregateModelRankingPage(context.Background(), db, now); err != nil || !caught {
		t.Fatalf("aggregate: caught=%v err=%v", caught, err)
	}
	var checkpoint entities.ModelRankingAggregationCheckpoint
	if err := db.Where("name = ?", "model_ranking").Take(&checkpoint).Error; err != nil {
		t.Fatal(err)
	}
	if checkpoint.LastAggregatedUsageEventID != 2 {
		t.Fatalf("cursor=%d", checkpoint.LastAggregatedUsageEventID)
	}
	var raw []entities.ModelRankingPeriodStat
	if err := db.Where("scope_key = '' AND period_kind = 'day'").Find(&raw).Error; err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2 {
		t.Fatalf("raw global rows=%+v", raw)
	}
	// Re-running after the committed cursor must be a no-op.
	if count, caught, err := repository.AggregateModelRankingPage(context.Background(), db, now); err != nil || !caught || count != 0 {
		t.Fatalf("replay: count=%d caught=%v err=%v", count, caught, err)
	}
	global, err := repository.ListModelRankingRollups(context.Background(), db, "day", "2026-11-19", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(global) != 2 {
		t.Fatalf("global provider rows=%d, want 2: %+v", len(global), global)
	}
	adminModels, err := repository.ListModelRankingRollups(context.Background(), db, "day", "2026-11-19", "", "")
	if err != nil || len(adminModels) != 2 {
		t.Fatalf("admin model contributions=%+v err=%v", adminModels, err)
	}
	keyA, err := repository.ListModelRankingRollups(context.Background(), db, "day", "2026-11-19", "key-a", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(keyA) != 1 || keyA[0].TotalTokens != 100 || keyA[0].CostUSD != 1.25 || keyA[0].UnavailableCostCount != 0 {
		t.Fatalf("key A rollup=%+v", keyA)
	}
	keyB, err := repository.ListModelRankingRollups(context.Background(), db, "day", "2026-11-19", "key-b", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(keyB) != 1 || keyB[0].TotalTokens != 200 || keyB[0].UnavailableCostCount != 1 {
		t.Fatalf("key B rollup=%+v", keyB)
	}
}

func TestBackfillModelRankingRollupsIsBoundedAndResumable(t *testing.T) {
	db := openModelRankingRollupDB(t)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.Local)
	const pageSize = 1000
	events := make([]entities.UsageEvent, 0, pageSize+1)
	for i := 0; i < pageSize+1; i++ {
		events = append(events, entities.UsageEvent{EventKey: fmt.Sprintf("backfill-%04d", i), APIGroupKey: "scope", Model: "model", Provider: "provider", Timestamp: now.Add(time.Duration(i) * time.Second), TotalTokens: 1})
	}
	if err := db.CreateInBatches(&events, 200).Error; err != nil {
		t.Fatal(err)
	}
	first, err := repository.BackfillModelRankingRollups(context.Background(), db, now, 1)
	if err != nil {
		t.Fatal(err)
	}
	if first.ProcessedEvents != pageSize || first.CaughtUp {
		t.Fatalf("first backfill batch=%+v", first)
	}
	second, err := repository.BackfillModelRankingRollups(context.Background(), db, now, 1)
	if err != nil {
		t.Fatal(err)
	}
	if second.ProcessedEvents != 1 || !second.CaughtUp {
		t.Fatalf("second backfill batch=%+v", second)
	}
}
