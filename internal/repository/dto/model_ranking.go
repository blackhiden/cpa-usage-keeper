package dto

import "time"

// ModelRankingWindow is the half-open [Start, End) local-calendar window used to
// aggregate the model leaderboard. It lives here (not in the feature domain) so
// the repository never imports internal/modelranking.
type ModelRankingWindow struct {
	Start time.Time
	End   time.Time
}

// ModelRankingRow is one aggregated display-name row for a model ranking
// window. Model is the alias when present, otherwise the real model name;
// Providers lists the distinct trimmed provider values that contributed.
type ModelRankingRow struct {
	Model              string
	Providers          []string
	RequestCount       int64
	SuccessCount       int64
	FailureCount       int64
	InputTokens        int64
	CacheReadTokens    int64
	TotalTokens        int64
	TTFTSumMS          int64
	TTFTSampleCount    int64
	LatencySumMS       int64
	LatencySampleCount int64
	Peak5MRequestCount int64
	Peak5MTotalTokens  int64
}