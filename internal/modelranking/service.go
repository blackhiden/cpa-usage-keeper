package modelranking

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"
	"cpa-usage-keeper/internal/timeutil"

	"gorm.io/gorm"
)

const modelRankingTopLimit = 100

// ErrInvalidSelection is returned for an unknown period or metric. The HTTP
// layer maps it to 400 invalid_model_ranking_selection.
var ErrInvalidSelection = errors.New("invalid model ranking selection")

// Entry is one ranked model row. Model is the alias when present, otherwise the
// real model name; Providers lists the contributing providers after collapsing.
type Entry struct {
	Rank            uint16   `json:"rank"`
	Model           string   `json:"model"`
	Providers       []string `json:"providers"`
	RequestCount    int64    `json:"request_count"`
	SuccessCount    int64    `json:"success_count"`
	FailureCount    int64    `json:"failure_count"`
	InputTokens     int64    `json:"input_tokens"`
	CacheReadTokens int64    `json:"cache_read_tokens"`
	TotalTokens     int64    `json:"total_tokens"`
	Value           int64    `json:"value"`
	RateNumerator   int64    `json:"rate_numerator,omitempty"`
	RateDenominator int64    `json:"rate_denominator,omitempty"`
}

// Response is the read-only model leaderboard payload for one period and metric.
type Response struct {
	Period      Period    `json:"period"`
	PeriodKey   string    `json:"period_key"`
	Metric      Metric    `json:"metric"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Timezone    string    `json:"timezone"`
	GeneratedAt time.Time `json:"generated_at"`
	Entries     []Entry   `json:"entries"`
}

// ServiceOptions only exposes the project clock the tests need to pin.
type ServiceOptions struct {
	Now func() time.Time
}

// Service aggregates model ranking rows on demand. It has no runner and no
// persisted state beyond the short-lived read cache.
type Service struct {
	db    *gorm.DB
	now   func() time.Time
	cache *modelRankingReadCache
}

// NewService builds the model ranking service over the given database.
func NewService(db *gorm.DB) (*Service, error) {
	return NewServiceWithOptions(db, ServiceOptions{})
}

// NewServiceWithOptions builds the service with an optional fixed clock.
func NewServiceWithOptions(db *gorm.DB, options ServiceOptions) (*Service, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &Service{db: db, now: now, cache: newModelRankingReadCache()}, nil
}

// Leaderboard returns the ranked model entries for one period and metric. An
// empty window returns an empty list, not an error. Successful responses are
// cached for 30 seconds keyed by period kind + period key + metric.
func (s *Service) Leaderboard(ctx context.Context, period Period, metric Metric) (Response, error) {
	if s == nil || s.db == nil || s.now == nil {
		return Response{}, fmt.Errorf("model ranking service is not configured")
	}
	if _, err := ParsePeriod(string(period)); err != nil {
		return Response{}, ErrInvalidSelection
	}
	if _, err := ParseMetric(string(metric)); err != nil {
		return Response{}, ErrInvalidSelection
	}
	now := s.now()
	window, err := ResolveWindow(period, timeutil.NormalizeStorageTime(now))
	if err != nil {
		return Response{}, ErrInvalidSelection
	}
	if cached, ok := s.cache.leaderboard(window.Kind, window.Key, metric, now); ok {
		return cached, nil
	}

	rows, err := repository.ListModelRankingRows(ctx, s.db, dto.ModelRankingWindow{Start: window.Start, End: window.End})
	if err != nil {
		return Response{}, err
	}
	board := Response{
		Period:      period,
		PeriodKey:   window.Key,
		Metric:      metric,
		Start:       window.Start,
		End:         window.End,
		Timezone:    time.Local.String(),
		GeneratedAt: now.UTC(),
		Entries:     buildModelRankingEntries(rows, metric),
	}
	s.cache.store(window.Kind, window.Key, metric, board, now)
	return board, nil
}

// buildModelRankingEntries projects repository rows onto one metric, drops rows
// with no eligible sample, orders them, truncates to the top limit, and assigns
// ranks starting at 1. Ties break by model name so the order is deterministic.
func buildModelRankingEntries(rows []dto.ModelRankingRow, metric Metric) []Entry {
	candidates := make([]Entry, 0, len(rows))
	for _, row := range rows {
		value, numerator, denominator, eligible := modelMetricValue(row, metric)
		if !eligible {
			continue
		}
		providers := row.Providers
		if providers == nil {
			providers = []string{}
		}
		candidates = append(candidates, Entry{
			Model:           row.Model,
			Providers:       providers,
			RequestCount:    row.RequestCount,
			SuccessCount:    row.SuccessCount,
			FailureCount:    row.FailureCount,
			InputTokens:     row.InputTokens,
			CacheReadTokens: row.CacheReadTokens,
			TotalTokens:     row.TotalTokens,
			Value:           value,
			RateNumerator:   numerator,
			RateDenominator: denominator,
		})
	}

	lowerIsBetter := metric.lowerIsBetter()
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].Value != candidates[right].Value {
			if lowerIsBetter {
				return candidates[left].Value < candidates[right].Value
			}
			return candidates[left].Value > candidates[right].Value
		}
		return candidates[left].Model < candidates[right].Model
	})
	if len(candidates) > modelRankingTopLimit {
		candidates = candidates[:modelRankingTopLimit]
	}
	for index := range candidates {
		candidates[index].Rank = uint16(index + 1)
	}
	return candidates
}

// modelMetricValue mirrors localMetricValue in
// internal/ranking/local_leaderboard.go: identical formulas, identical
// eligibility rules. Duration metrics are scaled means in milliseconds; cache
// rate is scaled by 1e6.
func modelMetricValue(row dto.ModelRankingRow, metric Metric) (int64, int64, int64, bool) {
	switch metric {
	case MetricTotalTokens:
		return row.TotalTokens, 0, 0, row.TotalTokens > 0
	case MetricRequestCount:
		return row.RequestCount, 0, 0, row.RequestCount > 0
	case MetricCacheReadRate:
		return scaledRatio(row.CacheReadTokens, row.InputTokens, 1_000_000), row.CacheReadTokens, row.InputTokens, row.InputTokens > 0
	case MetricTTFTAverage:
		return scaledRatio(row.TTFTSumMS, row.TTFTSampleCount, 1_000), row.TTFTSumMS, row.TTFTSampleCount, row.TTFTSampleCount > 0 && row.TTFTSumMS > 0
	case MetricLatencyAverage:
		return scaledRatio(row.LatencySumMS, row.LatencySampleCount, 1_000), row.LatencySumMS, row.LatencySampleCount, row.LatencySampleCount > 0 && row.LatencySumMS > 0
	case MetricPeakTPM:
		return row.Peak5MTotalTokens, 0, 0, row.Peak5MTotalTokens > 0
	case MetricPeakRPM:
		return row.Peak5MRequestCount, 0, 0, row.Peak5MRequestCount > 0
	default:
		return 0, 0, 0, false
	}
}