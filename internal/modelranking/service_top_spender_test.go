package modelranking

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/helper"
	"cpa-usage-keeper/internal/repository"
)

func spenderRow(model, scope string, cost float64, tokens, requests int64) repository.ModelRankingSpenderRow {
	return repository.ModelRankingSpenderRow{Model: model, ScopeKey: scope, CostUSD: cost, TotalTokens: tokens, RequestCount: requests}
}

func spenderRowWithCache(model, scope string, cost float64, tokens, requests, cacheRead int64) repository.ModelRankingSpenderRow {
	row := spenderRow(model, scope, cost, tokens, requests)
	row.CacheReadTokens = cacheRead
	return row
}

func TestPickModelRankingSpendersFollowsSelectedMetric(t *testing.T) {
	// key-a spends the most, key-b calls the most, key-c moves the most tokens
	// and key-d reads the most cache, so each metric must name a different key.
	rows := []repository.ModelRankingSpenderRow{
		spenderRowWithCache("gpt-5", "key-a", 9.5, 400, 4, 10),
		spenderRowWithCache("gpt-5", "key-b", 0.5, 300, 40, 20),
		spenderRowWithCache("gpt-5", "key-c", 1.5, 9_000, 3, 30),
		spenderRowWithCache("gpt-5", "key-d", 0.1, 200, 2, 5_000),
	}
	cases := []struct {
		metric Metric
		want   string
		reason string
	}{
		{MetricOverall, "key-a", "overall falls back to estimated cost"},
		{MetricTTFTAverage, "key-a", "latency is not a workload dimension"},
		{MetricLatencyAverage, "key-a", "latency is not a workload dimension"},
		{MetricTotalTokens, "key-c", "tokens lead the Tokens board"},
		{MetricPeakTPM, "key-c", "peak TPM follows token volume"},
		{MetricRequestCount, "key-b", "requests lead the Requests board"},
		{MetricPeakRPM, "key-b", "peak RPM follows request volume"},
		{MetricCacheReadRate, "key-d", "cache-read tokens lead the Cache board"},
	}
	for _, testCase := range cases {
		leaders := pickModelRankingSpenders(rows, testCase.metric)
		if got := leaders["gpt-5"].ScopeKey; got != testCase.want {
			t.Fatalf("%s: leader = %q, want %q (%s)", testCase.metric, got, testCase.want, testCase.reason)
		}
	}
}

func TestModelRankingSpenderOutranksTieBreaks(t *testing.T) {
	unused := spenderRow("m", "key-a", 0, 0, 0)
	used := spenderRow("m", "key-a", 0, 10, 0)
	if !modelRankingSpenderOutranks(used, unused, MetricOverall) {
		t.Fatal("an unpriced but used contribution must outrank an empty one")
	}
	if modelRankingSpenderOutranks(spenderRow("m", "key-b", 0, 10, 5), spenderRow("m", "key-a", 0, 10, 5), MetricOverall) {
		t.Fatal("equal contributions must keep the lexicographically first key")
	}
	if !modelRankingSpenderOutranks(spenderRow("m", "key-a", 1, 1, 1), spenderRow("m", "key-a", 0, 10_000, 100), MetricOverall) {
		t.Fatal("on the spend order estimated cost must outrank any unpriced usage")
	}
	if !modelRankingSpenderOutranks(spenderRow("m", "key-a", 0, 10_000, 100), spenderRow("m", "key-a", 1, 1, 1), MetricTotalTokens) {
		t.Fatal("on the Tokens board tokens must outrank estimated cost")
	}
	if !modelRankingSpenderOutranks(spenderRow("m", "key-a", 5, 10, 100), spenderRow("m", "key-a", 900, 900, 1), MetricRequestCount) {
		t.Fatal("on the Requests board requests must outrank tokens and cost")
	}
	if !modelRankingSpenderOutranks(spenderRow("m", "key-a", 0, 50, 7), spenderRow("m", "key-a", 9, 1, 7), MetricRequestCount) {
		t.Fatal("equal requests must fall through to tokens on the Requests board")
	}
	if !modelRankingSpenderOutranks(
		spenderRowWithCache("m", "key-a", 0, 1, 1, 8_000),
		spenderRowWithCache("m", "key-a", 9, 9_000, 90, 100),
		MetricCacheReadRate,
	) {
		t.Fatal("on the Cache board cache-read tokens must outrank cost and tokens")
	}
}

func TestServiceLeaderboardAttachesTopSpender(t *testing.T) {
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "modelranking-top-spender.db")})
	if err != nil {
		t.Fatalf("open database pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatalf("load reader pool: %v", err)
	}
	t.Cleanup(func() { _ = readerSQL.Close() })

	clock := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	generate := true
	available := true
	cheapCost := 0.25
	expensiveCost := 3.5
	ttftMS := int64(120)
	latencyMS := int64(900)
	events := []entities.UsageEvent{
		{EventKey: "spender-cheap", APIGroupKey: "sk-cheap-111111111111", Model: "gpt-5", Provider: "openai", Timestamp: clock, InputTokens: 800, TotalTokens: 900, TTFTMS: &ttftMS, LatencyMS: latencyMS, CostUSD: &cheapCost, CostAvailable: &available, Generate: &generate},
		{EventKey: "spender-pricey", APIGroupKey: "sk-pricey-222222222222", Model: "gpt-5", Provider: "openai", Timestamp: clock.Add(time.Minute), InputTokens: 80, TotalTokens: 100, TTFTMS: &ttftMS, LatencyMS: latencyMS, CostUSD: &expensiveCost, CostAvailable: &available, Generate: &generate},
		{EventKey: "spender-frequent-1", APIGroupKey: "sk-frequent-444444444444", Model: "gpt-5", Provider: "openai", Timestamp: clock.Add(2 * time.Minute), InputTokens: 5, TotalTokens: 5, TTFTMS: &ttftMS, LatencyMS: latencyMS, Generate: &generate},
		{EventKey: "spender-frequent-2", APIGroupKey: "sk-frequent-444444444444", Model: "gpt-5", Provider: "openai", Timestamp: clock.Add(3 * time.Minute), InputTokens: 5, TotalTokens: 5, TTFTMS: &ttftMS, LatencyMS: latencyMS, Generate: &generate},
		{EventKey: "spender-frequent-3", APIGroupKey: "sk-frequent-444444444444", Model: "gpt-5", Provider: "openai", Timestamp: clock.Add(4 * time.Minute), InputTokens: 5, TotalTokens: 5, TTFTMS: &ttftMS, LatencyMS: latencyMS, Generate: &generate},
		{EventKey: "spender-unpriced", APIGroupKey: "sk-noname-333333333333", Model: "claude", Provider: "anthropic", Timestamp: clock, InputTokens: 400, TotalTokens: 500, TTFTMS: &ttftMS, LatencyMS: latencyMS, Generate: &generate},
	}
	for index := range events {
		if err := db.Create(&events[index]).Error; err != nil {
			t.Fatalf("create event %d: %v", index, err)
		}
	}
	pricey := entities.CPAAPIKey{APIKey: "sk-pricey-222222222222", KeyName: "Pricey Key", DisplayKey: "sk-*********2222", SourceType: entities.CPAAPIKeySourceNative}
	if err := db.Create(&pricey).Error; err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if _, _, err := repository.AggregateModelRankingPage(context.Background(), db, clock); err != nil {
		t.Fatalf("aggregate rollup: %v", err)
	}

	service, err := NewServiceWithOptions(db, ServiceOptions{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatalf("NewServiceWithOptions: %v", err)
	}
	maskedKey := func(apiKey string) string {
		return helper.CPAAPIKeyDisplayName(entities.CPAAPIKey{APIKey: apiKey})
	}
	topSpender := func(metric Metric, model string) *TopSpender {
		t.Helper()
		board, err := service.Leaderboard(context.Background(), PeriodToday, metric)
		if err != nil {
			t.Fatalf("Leaderboard(%s): %v", metric, err)
		}
		for _, entry := range board.Entries {
			if entry.Model == model {
				return entry.TopSpender
			}
		}
		t.Fatalf("%s board is missing %q: %+v", metric, model, board.Entries)
		return nil
	}

	spend := topSpender(MetricOverall, "gpt-5")
	if spend == nil || spend.DisplayName != "Pricey Key" {
		t.Fatalf("overall top spender = %+v, want the priced key name", spend)
	}
	if spend.MaskedKey != "sk-*********222222" {
		t.Fatalf("gpt-5 masked key = %q", spend.MaskedKey)
	}
	if tokens := topSpender(MetricTotalTokens, "gpt-5"); tokens == nil || tokens.DisplayName != maskedKey("sk-cheap-111111111111") {
		t.Fatalf("tokens top spender = %+v, want the token-heavy key", tokens)
	}
	if requests := topSpender(MetricRequestCount, "gpt-5"); requests == nil || requests.DisplayName != maskedKey("sk-frequent-444444444444") {
		t.Fatalf("requests top spender = %+v, want the call-heavy key", requests)
	}

	claudeSpender := topSpender(MetricOverall, "claude")
	if claudeSpender == nil {
		t.Fatal("claude top spender is missing for an unpriced but used model")
	}
	if want := maskedKey("sk-noname-333333333333"); claudeSpender.DisplayName != want {
		t.Fatalf("claude top spender = %q, want masked %q", claudeSpender.DisplayName, want)
	}
	if claudeSpender.DisplayName == "sk-noname-333333333333" {
		t.Fatal("an unresolved key must never render raw")
	}

	scoped, err := service.LeaderboardWithOptions(context.Background(), PeriodToday, MetricRequestCount, LeaderboardOptions{ScopeKey: "sk-cheap-111111111111"})
	if err != nil {
		t.Fatalf("LeaderboardWithOptions: %v", err)
	}
	if len(scoped.Entries) != 1 || scoped.Entries[0].Model != "gpt-5" {
		t.Fatalf("scoped entries = %+v", scoped.Entries)
	}
	if scoped.Entries[0].TopSpender == nil || scoped.Entries[0].TopSpender.DisplayName != maskedKey("sk-cheap-111111111111") {
		t.Fatalf("scoped top spender = %+v, want the viewer's own key", scoped.Entries[0].TopSpender)
	}
}
