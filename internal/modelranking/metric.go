package modelranking

import (
	"fmt"
	"math"
	"math/big"
)

// Metric is a quantitative ranking metric. The wire values intentionally match
// ranking.LeaderboardMetric (minus overall); metric_test.go asserts the parity so
// the two leaderboards cannot drift silently.
type Metric string

const (
	MetricTotalTokens    Metric = "total_tokens"
	MetricRequestCount   Metric = "request_count"
	MetricCacheReadRate  Metric = "cache_read_rate"
	MetricTTFTAverage    Metric = "ttft_average"
	MetricLatencyAverage Metric = "latency_average"
	MetricPeakTPM        Metric = "peak_tpm"
	MetricPeakRPM        Metric = "peak_rpm"
)

// AllMetrics returns the seven supported metrics in display order.
func AllMetrics() []Metric {
	return []Metric{
		MetricTotalTokens,
		MetricRequestCount,
		MetricCacheReadRate,
		MetricTTFTAverage,
		MetricLatencyAverage,
		MetricPeakTPM,
		MetricPeakRPM,
	}
}

// ParseMetric parses a metric value, rejecting unknown values.
func ParseMetric(value string) (Metric, error) {
	metric := Metric(value)
	for _, candidate := range AllMetrics() {
		if metric == candidate {
			return metric, nil
		}
	}
	return "", fmt.Errorf("unknown model ranking metric %q", value)
}

// lowerIsBetter reports whether a smaller value is a better rank. Only TTFT and
// latency averages are ascending; every other metric sorts descending.
func (m Metric) lowerIsBetter() bool {
	return m == MetricTTFTAverage || m == MetricLatencyAverage
}

// scaledRatio returns numerator * scale / denominator, saturating at MaxInt64.
// It mirrors the helper in internal/ranking/local_leaderboard.go so the two
// leaderboards produce identical values from identical rows.
func scaledRatio(numerator, denominator, scale int64) int64 {
	if denominator <= 0 || numerator <= 0 || scale <= 0 {
		return 0
	}
	value := new(big.Int).Mul(big.NewInt(numerator), big.NewInt(scale))
	value.Div(value, big.NewInt(denominator))
	if !value.IsInt64() {
		return math.MaxInt64
	}
	return value.Int64()
}