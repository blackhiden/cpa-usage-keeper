package test

import (
	"context"
	"testing"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
)

func TestListModelRankingSpendersAggregatesScopedRows(t *testing.T) {
	db := openModelRankingRollupDB(t)
	now := time.Date(2026, 11, 19, 12, 0, 0, 0, time.Local)
	generate := true
	available := true
	smallCost := 0.5
	largeCost := 2.0
	events := []entities.UsageEvent{
		{EventKey: "spender-1", APIGroupKey: "key-a", Model: "gpt-5", Provider: "openai", Timestamp: now, TotalTokens: 100, InputTokens: 40, CacheReadTokens: 30, CostUSD: &smallCost, CostAvailable: &available, Generate: &generate},
		{EventKey: "spender-2", APIGroupKey: "key-b", Model: "gpt-5", Provider: "openai", Timestamp: now.Add(time.Minute), TotalTokens: 300, InputTokens: 120, CacheReadTokens: 70, CostUSD: &largeCost, CostAvailable: &available, Generate: &generate},
		{EventKey: "spender-3", APIGroupKey: "key-b", Model: "gpt-5", Provider: "azure", Timestamp: now.Add(2 * time.Minute), TotalTokens: 50, InputTokens: 20, CacheReadTokens: 5, CostUSD: &smallCost, CostAvailable: &available, Generate: &generate},
		{EventKey: "spender-4", APIGroupKey: "key-a", Model: "claude", Provider: "anthropic", Timestamp: now, TotalTokens: 700, InputTokens: 500, CacheReadTokens: 400, Generate: &generate},
	}
	for index := range events {
		if err := db.Create(&events[index]).Error; err != nil {
			t.Fatalf("create event %d: %v", index, err)
		}
	}
	if _, _, err := repository.AggregateModelRankingPage(context.Background(), db, now); err != nil {
		t.Fatalf("aggregate rollup: %v", err)
	}

	rows, err := repository.ListModelRankingSpenders(context.Background(), db, "day", "2026-11-19", "", "", []string{"gpt-5", "claude"})
	if err != nil {
		t.Fatalf("ListModelRankingSpenders: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("spender rows = %d, want 3: %+v", len(rows), rows)
	}
	byModelKey := make(map[string]repository.ModelRankingSpenderRow, len(rows))
	for _, row := range rows {
		byModelKey[row.Model+"/"+row.ScopeKey] = row
	}
	keyA := byModelKey["gpt-5/key-a"]
	if keyA.RequestCount != 1 || keyA.TotalTokens != 100 || keyA.CacheReadTokens != 30 || keyA.CostUSD != 0.5 || keyA.UnavailableCostCount != 0 {
		t.Fatalf("gpt-5/key-a = %+v", keyA)
	}
	keyB := byModelKey["gpt-5/key-b"]
	if keyB.RequestCount != 2 || keyB.TotalTokens != 350 || keyB.CacheReadTokens != 75 || keyB.CostUSD != 2.5 || keyB.UnavailableCostCount != 0 {
		t.Fatalf("gpt-5/key-b = %+v", keyB)
	}
	claude := byModelKey["claude/key-a"]
	if claude.RequestCount != 1 || claude.TotalTokens != 700 || claude.CacheReadTokens != 400 || claude.CostUSD != 0 || claude.UnavailableCostCount != 1 {
		t.Fatalf("claude/key-a = %+v", claude)
	}

	scoped, err := repository.ListModelRankingSpenders(context.Background(), db, "day", "2026-11-19", "key-a", "", []string{"gpt-5", "claude"})
	if err != nil {
		t.Fatalf("scoped spenders: %v", err)
	}
	if len(scoped) != 2 {
		t.Fatalf("scoped rows = %d, want 2: %+v", len(scoped), scoped)
	}
	for _, row := range scoped {
		if row.ScopeKey != "key-a" {
			t.Fatalf("scoped row leaked another key: %+v", row)
		}
	}

	filtered, err := repository.ListModelRankingSpenders(context.Background(), db, "day", "2026-11-19", "", "azure", []string{"gpt-5", "claude"})
	if err != nil {
		t.Fatalf("provider spenders: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ScopeKey != "key-b" || filtered[0].TotalTokens != 50 {
		t.Fatalf("provider-filtered rows = %+v, want gpt-5/key-b with 50 tokens", filtered)
	}

	unbounded, err := repository.ListModelRankingSpenders(context.Background(), db, "day", "2026-11-19", "", "", []string{"other-model"})
	if err != nil {
		t.Fatalf("unmatched models: %v", err)
	}
	if len(unbounded) != 0 {
		t.Fatalf("unmatched model rows = %+v, want none", unbounded)
	}

	empty, err := repository.ListModelRankingSpenders(context.Background(), db, "day", "2026-11-19", "", "", nil)
	if err != nil {
		t.Fatalf("empty model list: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("empty model list rows = %+v, want none", empty)
	}

	if _, err := repository.ListModelRankingSpenders(context.Background(), nil, "day", "2026-11-19", "", "", []string{"gpt-5"}); err == nil {
		t.Fatal("expected an error for a nil database")
	}
}

func TestFindCPAAPIKeysByValuesResolvesStoredNames(t *testing.T) {
	db := openModelRankingRollupDB(t)
	for _, row := range []entities.CPAAPIKey{
		{APIKey: "sk-alpha-1234567890", DisplayKey: "sk-*********7890", KeyName: "Alpha", SourceType: entities.CPAAPIKeySourceNative},
		{APIKey: "sk-beta-0987654321", DisplayKey: "sk-*********4321", KeyAlias: "Beta", SourceType: entities.CPAAPIKeySourceNative, IsDeleted: true},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create api key: %v", err)
		}
	}

	rows, err := repository.FindCPAAPIKeysByValues(db, []string{" sk-alpha-1234567890 ", "sk-beta-0987654321", "sk-alpha-1234567890", "", "sk-unknown"})
	if err != nil {
		t.Fatalf("FindCPAAPIKeysByValues: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("resolved keys = %d, want 2: %+v", len(rows), rows)
	}
	byValue := make(map[string]entities.CPAAPIKey, len(rows))
	for _, row := range rows {
		byValue[row.APIKey] = row
	}
	if byValue["sk-alpha-1234567890"].KeyName != "Alpha" {
		t.Fatalf("alpha key = %+v", byValue["sk-alpha-1234567890"])
	}
	if !byValue["sk-beta-0987654321"].IsDeleted || byValue["sk-beta-0987654321"].KeyAlias != "Beta" {
		t.Fatalf("soft-deleted key must stay resolvable: %+v", byValue["sk-beta-0987654321"])
	}

	empty, err := repository.FindCPAAPIKeysByValues(db, nil)
	if err != nil {
		t.Fatalf("empty lookup: %v", err)
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("empty lookup rows = %+v, want an empty slice", empty)
	}
}
