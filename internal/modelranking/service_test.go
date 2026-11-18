package modelranking

import (
	"fmt"
	"testing"

	"cpa-usage-keeper/internal/repository/dto"
)

func fullModelRankingRow() dto.ModelRankingRow {
	return dto.ModelRankingRow{
		Model:              "m",
		Providers:          []string{"alpha", "beta"},
		RequestCount:       2,
		SuccessCount:       1,
		FailureCount:       1,
		InputTokens:        200,
		CacheReadTokens:    50,
		TotalTokens:        100,
		TTFTSumMS:          500,
		TTFTSampleCount:    2,
		LatencySumMS:       3000,
		LatencySampleCount: 3,
		Peak5MRequestCount: 7,
		Peak5MTotalTokens:  400,
	}
}

func TestModelMetricValueMatchesRankingFormulas(t *testing.T) {
	row := fullModelRankingRow()
	tests := []struct {
		metric      Metric
		wantValue   int64
		wantNum     int64
		wantDen     int64
	}{
		{MetricTotalTokens, 100, 0, 0},
		{MetricRequestCount, 2, 0, 0},
		{MetricCacheReadRate, 250_000, 50, 200},
		{MetricTTFTAverage, 250_000, 500, 2},
		{MetricLatencyAverage, 1_000_000, 3000, 3},
		{MetricPeakTPM, 400, 0, 0},
		{MetricPeakRPM, 7, 0, 0},
	}
	for _, tt := range tests {
		value, numerator, denominator, eligible := modelMetricValue(row, tt.metric)
		if !eligible {
			t.Fatalf("%s unexpectedly ineligible", tt.metric)
		}
		if value != tt.wantValue || numerator != tt.wantNum || denominator != tt.wantDen {
			t.Fatalf("%s = (%d, %d, %d), want (%d, %d, %d)", tt.metric, value, numerator, denominator, tt.wantValue, tt.wantNum, tt.wantDen)
		}
	}
}

func TestModelMetricValueOmitsIneligibleRows(t *testing.T) {
	for _, metric := range AllMetrics() {
		if _, _, _, eligible := modelMetricValue(dto.ModelRankingRow{Model: "empty"}, metric); eligible {
			t.Fatalf("%s marked an empty row eligible", metric)
		}
	}
	if value, _, _, eligible := modelMetricValue(dto.ModelRankingRow{InputTokens: 10}, MetricCacheReadRate); !eligible || value != 0 {
		t.Fatalf("cache rate with no cache reads = (%d, eligible=%v), want (0, true) matching the API-key leaderboard", value, eligible)
	}
	if _, _, _, eligible := modelMetricValue(dto.ModelRankingRow{TTFTSampleCount: 2}, MetricTTFTAverage); eligible {
		t.Fatal("TTFT without a positive sum must be ineligible")
	}
}

func TestBuildModelRankingEntriesOrdersAndRanks(t *testing.T) {
	rows := []dto.ModelRankingRow{
		{Model: "alpha", TotalTokens: 100},
		{Model: "beta", TotalTokens: 200},
		{Model: "gamma", TotalTokens: 200},
	}
	entries := buildModelRankingEntries(rows, MetricTotalTokens)
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	want := []string{"beta", "gamma", "alpha"}
	for index, model := range want {
		if entries[index].Model != model {
			t.Fatalf("entries[%d] = %q, want %q", index, entries[index].Model, model)
		}
		if entries[index].Rank != uint16(index+1) {
			t.Fatalf("entries[%d].Rank = %d, want %d", index, entries[index].Rank, index+1)
		}
	}
}

func TestBuildModelRankingEntriesAscendingForLatencyMetrics(t *testing.T) {
	rows := []dto.ModelRankingRow{
		{Model: "slow", TTFTSumMS: 900, TTFTSampleCount: 3},
		{Model: "fast", TTFTSumMS: 300, TTFTSampleCount: 3},
		{Model: "medium", TTFTSumMS: 600, TTFTSampleCount: 3},
	}
	entries := buildModelRankingEntries(rows, MetricTTFTAverage)
	want := []string{"fast", "medium", "slow"}
	for index, model := range want {
		if entries[index].Model != model {
			t.Fatalf("TTFT order[%d] = %q, want %q", index, entries[index].Model, model)
		}
	}
}

func TestBuildModelRankingEntriesTruncatesToTopLimit(t *testing.T) {
	rows := make([]dto.ModelRankingRow, 0, 110)
	for index := 0; index < 110; index++ {
		rows = append(rows, dto.ModelRankingRow{Model: modelName(index), TotalTokens: int64(index + 1)})
	}
	entries := buildModelRankingEntries(rows, MetricTotalTokens)
	if len(entries) != modelRankingTopLimit {
		t.Fatalf("entries = %d, want %d", len(entries), modelRankingTopLimit)
	}
	if entries[0].Model != modelName(109) || entries[99].Model != modelName(10) {
		t.Fatalf("truncated order first=%q last=%q", entries[0].Model, entries[99].Model)
	}
}

func TestBuildModelRankingEntriesEmpty(t *testing.T) {
	entries := buildModelRankingEntries(nil, MetricTotalTokens)
	if entries == nil {
		t.Fatal("expected an empty non-nil slice")
	}
	if len(entries) != 0 {
		t.Fatalf("entries = %d, want 0", len(entries))
	}
}

func modelName(index int) string {
	// A stable zero-padded name keeps tie-breaking assertions readable.
	return fmt.Sprintf("m%03d", index)
}