package modelranking

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"cpa-usage-keeper/internal/rankingmath"
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
	Rank                  uint16                 `json:"rank"`
	Model                 string                 `json:"model"`
	Providers             []string               `json:"providers"`
	RequestCount          int64                  `json:"request_count"`
	SuccessCount          int64                  `json:"success_count"`
	FailureCount          int64                  `json:"failure_count"`
	InputTokens           int64                  `json:"input_tokens"`
	CacheReadTokens       int64                  `json:"cache_read_tokens"`
	TotalTokens           int64                  `json:"total_tokens"`
	Value                 int64                  `json:"value"`
	RateNumerator         int64                  `json:"rate_numerator,omitempty"`
	RateDenominator       int64                  `json:"rate_denominator,omitempty"`
	CostUSD               float64                `json:"cost_usd"`
	UnavailableCostCount  int64                  `json:"unavailable_cost_count"`
	ProviderContributions []ProviderContribution `json:"provider_contributions"`
	Trend                 *Trend                 `json:"trend,omitempty"`
}

type ProviderContribution struct {
	Provider     string `json:"provider"`
	RequestCount int64  `json:"request_count"`
	TotalTokens  int64  `json:"total_tokens"`
}

type Trend struct {
	Change        int64    `json:"change"`
	Percent       *float64 `json:"percent,omitempty"`
	PreviousValue int64    `json:"previous_value"`
	Available     bool     `json:"available"`
}

type ScoreExplanation struct {
	Version int               `json:"version"`
	Texts   map[string]string `json:"texts"`
}

type LeaderboardOptions struct {
	Provider     string
	ScopeKey     string
	IncludeTrend bool
}

// Response is the read-only model leaderboard payload for one period and metric.
type Response struct {
	Period           Period            `json:"period"`
	PeriodKey        string            `json:"period_key"`
	Metric           Metric            `json:"metric"`
	Start            time.Time         `json:"start"`
	End              time.Time         `json:"end"`
	Timezone         string            `json:"timezone"`
	GeneratedAt      time.Time         `json:"generated_at"`
	Entries          []Entry           `json:"entries"`
	ScoreExplanation *ScoreExplanation `json:"score_explanation,omitempty"`
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
	if err := repository.EnsureModelRankingRollupCheckpoint(context.Background(), db, time.Now()); err != nil {
		return nil, fmt.Errorf("initialize model ranking checkpoint: %w", err)
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
	return s.LeaderboardWithOptions(ctx, period, metric, LeaderboardOptions{})
}

func (s *Service) LeaderboardWithOptions(ctx context.Context, period Period, metric Metric, options LeaderboardOptions) (Response, error) {
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
	if cached, ok := s.cache.leaderboard(window.Kind, window.Key, metric, options.ScopeKey, options.Provider, options.IncludeTrend, now); ok {
		return cached, nil
	}

	rows, err := repository.ListModelRankingRollups(ctx, s.db, string(window.Kind), window.Key, options.ScopeKey, options.Provider)
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
		Entries:     buildModelRankingEntriesFromRollups(rows, metric),
	}
	if options.IncludeTrend {
		attachTrends(ctx, s, &board, options)
	}
	if metric == MetricOverall {
		board.ScoreExplanation = modelOverallScoreExplanation()
	}
	s.cache.store(window.Kind, window.Key, metric, options.ScopeKey, options.Provider, options.IncludeTrend, board, now)
	return board, nil
}

func modelOverallScoreExplanation() *ScoreExplanation {
	return &ScoreExplanation{Version: 1, Texts: map[string]string{
		"en":    "Model overall score combines workload 30% (tokens 20%, requests 10%), reliability 30% (95% Wilson lower bound), response 25% (TTFT 15%, latency 10%), cache efficiency 10%, and peak capacity 5% (TPM 3%, RPM 2%). It is normalized among eligible models in this scope, provider filter, and period on a 0–100 scale.",
		"zh":    "模型综合分由使用规模 30%（Token 20%、请求数 10%）、可靠性 30%（95% Wilson 成功率下界）、响应 25%（首字延迟 15%、总延迟 10%）、缓存效率 10% 和峰值能力 5%（TPM 3%、RPM 2%）组成。分数在当前范围、提供商筛选和周期内的合格模型间归一化，范围为 0–100。",
		"zh-TW": "模型綜合分由使用規模 30%（Token 20%、請求數 10%）、可靠性 30%（95% Wilson 成功率下界）、回應 25%（首字延遲 15%、總延遲 10%）、快取效率 10% 和峰值能力 5%（TPM 3%、RPM 2%）組成。分數在目前範圍、提供商篩選和週期內的合格模型間正規化，範圍為 0–100。",
	}}
}

func buildModelRankingEntriesFromRollups(rows []repository.ModelRankingRollupRow, metric Metric) []Entry {
	byModel := make(map[string]*dto.ModelRankingRow)
	type providerKey struct{ model, provider string }
	providerTotals := make(map[providerKey]ProviderContribution)
	type peakKey struct {
		model  string
		bucket int64
	}
	type peakValue struct{ requests, tokens int64 }
	peaks := make(map[peakKey]peakValue)
	costs := make(map[string]repository.ModelRankingRollupRow)
	for _, row := range rows {
		current := byModel[row.Model]
		if current == nil {
			current = &dto.ModelRankingRow{Model: row.Model, Providers: []string{}}
			byModel[row.Model] = current
		}
		current.RequestCount += row.RequestCount
		current.SuccessCount += row.SuccessCount
		current.FailureCount += row.FailureCount
		current.InputTokens += row.InputTokens
		current.CacheReadTokens += row.CacheReadTokens
		current.TotalTokens += row.TotalTokens
		current.TTFTSumMS += row.TTFTSumMS
		current.TTFTSampleCount += row.TTFTSampleCount
		current.LatencySumMS += row.LatencySumMS
		current.LatencySampleCount += row.LatencySampleCount
		if row.Provider != "" {
			key := providerKey{row.Model, row.Provider}
			share := providerTotals[key]
			share.Provider = row.Provider
			share.RequestCount += row.RequestCount
			share.TotalTokens += row.TotalTokens
			providerTotals[key] = share
		}
		key := peakKey{row.Model, row.BucketKey}
		value := peaks[key]
		value.requests += row.RequestCount
		value.tokens += row.TotalTokens
		peaks[key] = value
		cost := costs[row.Model]
		cost.CostUSD += row.CostUSD
		cost.UnavailableCostCount += row.UnavailableCostCount
		costs[row.Model] = cost
	}
	for key, value := range peaks {
		row := byModel[key.model]
		row.Peak5MRequestCount = max(row.Peak5MRequestCount, value.requests)
		row.Peak5MTotalTokens = max(row.Peak5MTotalTokens, value.tokens)
	}
	contributions := make(map[string][]ProviderContribution)
	for key, share := range providerTotals {
		contributions[key.model] = append(contributions[key.model], share)
		byModel[key.model].Providers = append(byModel[key.model].Providers, key.provider)
	}
	flat := make([]dto.ModelRankingRow, 0, len(byModel))
	for _, row := range byModel {
		flat = append(flat, *row)
	}
	var entries []Entry
	if metric == MetricOverall {
		entries = buildModelOverallEntries(flat)
	} else {
		entries = buildModelRankingEntries(flat, metric)
	}
	for i := range entries {
		cost := costs[entries[i].Model]
		entries[i].CostUSD = cost.CostUSD
		entries[i].UnavailableCostCount = cost.UnavailableCostCount
		entries[i].ProviderContributions = contributions[entries[i].Model]
	}
	return entries
}

func buildModelOverallEntries(rows []dto.ModelRankingRow) []Entry {
	eligibleRows := make([]dto.ModelRankingRow, 0, len(rows))
	scoresIn := make([]rankingmath.Metrics, 0, len(rows))
	for _, row := range rows {
		m := rankingmath.Metrics{RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, FailureCount: row.FailureCount, InputTokens: row.InputTokens, CacheReadTokens: row.CacheReadTokens, TotalTokens: row.TotalTokens, TTFTSumMS: row.TTFTSumMS, TTFTSampleCount: row.TTFTSampleCount, LatencySumMS: row.LatencySumMS, LatencySampleCount: row.LatencySampleCount, Peak5MRequestCount: row.Peak5MRequestCount, Peak5MTotalTokens: row.Peak5MTotalTokens}
		if rankingmath.Eligible(m) {
			eligibleRows = append(eligibleRows, row)
			scoresIn = append(scoresIn, m)
		}
	}
	scores := rankingmath.Score(scoresIn)
	entries := make([]Entry, 0, len(eligibleRows))
	for index, row := range eligibleRows {
		entries = append(entries, Entry{Model: row.Model, Providers: row.Providers, RequestCount: row.RequestCount, SuccessCount: row.SuccessCount, FailureCount: row.FailureCount, InputTokens: row.InputTokens, CacheReadTokens: row.CacheReadTokens, TotalTokens: row.TotalTokens, Value: scores[index]})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Value != entries[j].Value {
			return entries[i].Value > entries[j].Value
		}
		return entries[i].Model < entries[j].Model
	})
	if len(entries) > modelRankingTopLimit {
		entries = entries[:modelRankingTopLimit]
	}
	for i := range entries {
		entries[i].Rank = uint16(i + 1)
	}
	return entries
}

func attachTrends(ctx context.Context, service *Service, response *Response, options LeaderboardOptions) {
	previous, ok := predecessor(response.Period)
	if !ok {
		return
	}
	window, err := ResolveWindow(previous, response.Start)
	if err != nil {
		return
	}
	rows, err := repository.ListModelRankingRollups(ctx, service.db, string(window.Kind), window.Key, options.ScopeKey, options.Provider)
	if err != nil {
		return
	}
	prior := buildModelRankingEntriesFromRollups(rows, response.Metric)
	values := make(map[string]int64, len(prior))
	for _, entry := range prior {
		values[entry.Model] = entry.Value
	}
	for i := range response.Entries {
		old, exists := values[response.Entries[i].Model]
		if !exists {
			response.Entries[i].Trend = &Trend{Available: false}
			continue
		}
		change := response.Entries[i].Value - old
		trend := &Trend{Change: change, PreviousValue: old, Available: true}
		if old != 0 {
			percent := float64(change) * 100 / float64(old)
			trend.Percent = &percent
		}
		response.Entries[i].Trend = trend
	}
}

func predecessor(period Period) (Period, bool) {
	switch period {
	case PeriodToday:
		return PeriodYesterday, true
	case PeriodCurrentWeek:
		return PeriodPreviousWeek, true
	case PeriodCurrentMonth:
		return PeriodPreviousMonth, true
	default:
		return "", false
	}
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
