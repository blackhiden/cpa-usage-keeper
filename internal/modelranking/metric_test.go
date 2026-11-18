package modelranking

import (
	"testing"

	"cpa-usage-keeper/internal/ranking"
)

func TestMetricWireStringsMatchRanking(t *testing.T) {
	// Parity is asserted against the existing API-key leaderboard vocabulary so
	// the two domains cannot drift while staying import-free of each other in
	// production code.
	rankingMetrics := []ranking.LeaderboardMetric{
		ranking.MetricTotalTokens,
		ranking.MetricRequestCount,
		ranking.MetricCacheReadRate,
		ranking.MetricTTFTAverage,
		ranking.MetricLatencyAverage,
		ranking.MetricPeakTPM,
		ranking.MetricPeakRPM,
	}
	metrics := AllMetrics()
	if len(metrics) != len(rankingMetrics) {
		t.Fatalf("metric count = %d, want %d", len(metrics), len(rankingMetrics))
	}
	for index, metric := range metrics {
		if string(metric) != string(rankingMetrics[index]) {
			t.Fatalf("metric[%d] = %q, want %q", index, metric, rankingMetrics[index])
		}
	}
}

func TestParseMetric(t *testing.T) {
	for _, metric := range AllMetrics() {
		if got, err := ParseMetric(string(metric)); err != nil || got != metric {
			t.Fatalf("ParseMetric(%q) = %q, %v; want %q", metric, got, err, metric)
		}
	}
	for _, invalid := range []string{"", "overall", "tokens", "TOTAL_TOKENS", "peak"} {
		if _, err := ParseMetric(invalid); err == nil {
			t.Fatalf("ParseMetric(%q) unexpectedly accepted", invalid)
		}
	}
}