package ranking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"sync"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/timeutil"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"
)

const (
	localRankingTopLimit              = 100
	localRankingPeriodStateSettingKey = "ranking.local.period_state"
	localRankingWriteBatchSize        = 100
	// localRankingSettleDayLimit 限制单次聚合补算的完整日数量,避免补算长时间占住
	// 聚合锁;进度按日持久化,剩余部分由下一批次继续。
	localRankingSettleDayLimit = 31
	// localRankingSettleGraceHour 之前不结算昨天:跨零点的写入仍然可能落到上一天。
	localRankingSettleGraceHour = 2
)

var (
	ErrInvalidLeaderboard = errors.New("invalid leaderboard selection")
)

// LocalRankingServiceOptions 只开放测试需要固定的项目时钟。
type LocalRankingServiceOptions struct {
	Now func() time.Time
}

// LocalRankingService 以完整自然日快照维护本地 API Key 排行。
type LocalRankingService struct {
	db  *gorm.DB
	now func() time.Time

	aggregateMu sync.Mutex

	lastDynamicDay        string
	lastDynamicAggregates []localRankingRawAggregate
	periodStateLoaded     bool
	lastSettledDay        string
	currentMonthRebuild   bool
}

type localRankingPeriodState struct {
	LastSettledDay string `json:"last_settled_day"`
	// PeriodTimezone/PeriodOffsetSeconds 记录结算日历时使用的本地期间时区。名称或
	// offset 任一变化都意味着日历边界已改变,旧键快照必须重算。
	PeriodTimezone      string `json:"period_timezone,omitempty"`
	PeriodOffsetSeconds int    `json:"period_offset_seconds,omitempty"`
	// CurrentMonthRebuild 表示期间时区变化后本月窗口尚未在新日历下重算;置位时
	// 本月榜单先重算本月完整日,不返回只累加了一部分日贡献的快照。
	CurrentMonthRebuild bool `json:"current_month_rebuild,omitempty"`
}

type localRankingPeriodWindow struct {
	Period LeaderboardPeriod
	Kind   entities.LocalRankingPeriodKind
	Key    string
	Start  time.Time
	End    time.Time
}

type localRankingStatKey struct {
	Kind     entities.LocalRankingPeriodKind
	Key      string
	APIKeyID int64
}

type localRankingMetrics struct {
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

type localRankingRawAggregate struct {
	APIKeyID           int64 `gorm:"column:api_key_id"`
	RequestCount       int64 `gorm:"column:request_count"`
	SuccessCount       int64 `gorm:"column:success_count"`
	FailureCount       int64 `gorm:"column:failure_count"`
	InputTokens        int64 `gorm:"column:input_tokens"`
	CacheReadTokens    int64 `gorm:"column:cache_read_tokens"`
	TotalTokens        int64 `gorm:"column:total_tokens"`
	TTFTSumMS          int64 `gorm:"column:ttft_sum_ms"`
	TTFTSampleCount    int64 `gorm:"column:ttft_sample_count"`
	LatencySumMS       int64 `gorm:"column:latency_sum_ms"`
	LatencySampleCount int64 `gorm:"column:latency_sample_count"`
	Peak5MRequestCount int64 `gorm:"column:peak_5m_request_count"`
	Peak5MTotalTokens  int64 `gorm:"column:peak_5m_total_tokens"`
}

type localRankingPopulationRow struct {
	APIKeyID             int64 `gorm:"column:api_key_id"`
	APIKey               string
	KeyName              string
	DisplayKey           string
	KeyAlias             string
	LocalRankingAvatarID *uint8
	UpdatedAt            time.Time
	localRankingMetrics
}

// NewLocalRankingService 创建独立于 Community 同步服务的本地排行服务。
func NewLocalRankingService(db *gorm.DB, options LocalRankingServiceOptions) (*LocalRankingService, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &LocalRankingService{db: db, now: now}, nil
}

// AggregateOnce 先覆盖当前完整今日快照,再补算到期完整日。
func (s *LocalRankingService) AggregateOnce(ctx context.Context) error {
	if s == nil || s.db == nil || s.now == nil {
		return fmt.Errorf("local ranking service is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	s.aggregateMu.Lock()
	defer s.aggregateMu.Unlock()

	snapshotAt := timeutil.NormalizeStorageTime(s.now())
	periodNow := snapshotAt.In(time.Local)
	if err := s.ensureLocalRankingPeriodState(ctx, periodNow, snapshotAt); err != nil {
		return err
	}
	// 今日快照先写入:历史完整日的补算可能跨多个批次,今日榜单不能等它完成。
	if err := s.refreshLocalRankingToday(ctx, periodNow, snapshotAt, true); err != nil {
		return err
	}
	return s.settlePendingLocalRankingDays(ctx, periodNow, snapshotAt)
}

// refreshLocalRankingToday 覆盖当前完整今日快照;同一天数据无变化时跳过写入。
func (s *LocalRankingService) refreshLocalRankingToday(ctx context.Context, periodNow, snapshotAt time.Time, prune bool) error {
	today := localRankingTodayWindow(periodNow)
	aggregates, err := s.aggregateLocalRankingWindow(ctx, today)
	if err != nil {
		return err
	}
	// 同一自然日直接比较上次成功结果，保留每轮一次 reader 扫描，同时避免无变化写入。
	if !prune && s.lastDynamicDay == today.Key && slices.Equal(s.lastDynamicAggregates, aggregates) {
		return nil
	}
	if err := s.replaceLocalRankingDay(ctx, today, aggregates, snapshotAt, nil, prune); err != nil {
		return err
	}
	s.lastDynamicDay = today.Key
	s.lastDynamicAggregates = slices.Clone(aggregates)
	return nil
}

func (s *LocalRankingService) settlePendingLocalRankingDays(ctx context.Context, periodNow, snapshotAt time.Time) error {
	if err := s.ensureLocalRankingPeriodState(ctx, periodNow, snapshotAt); err != nil {
		return err
	}
	if periodNow.Hour() < localRankingSettleGraceHour {
		return nil
	}
	target := localRankingStartOfDay(periodNow).AddDate(0, 0, -1)
	last, err := time.ParseInLocation("2006-01-02", s.lastSettledDay, time.Local)
	if err != nil {
		return fmt.Errorf("parse local ranking last settled day: %w", err)
	}
	if last.After(target) {
		// 本地日历回退(时钟回拨或 offset 变化)时把游标收回到可结算范围,不报错。
		if err := s.saveLocalRankingPeriodState(ctx, periodNow, target.Format("2006-01-02"), s.currentMonthRebuild, snapshotAt); err != nil {
			return err
		}
		last = target
	}
	settled := 0
	for day := last.AddDate(0, 0, 1); !day.After(target); day = day.AddDate(0, 0, 1) {
		if settled >= localRankingSettleDayLimit {
			return nil
		}
		window := localRankingCompleteDayWindow(day)
		aggregates, err := s.aggregateLocalRankingWindow(ctx, window)
		if err != nil {
			return err
		}
		dayKey := day.Format("2006-01-02")
		next := localRankingPeriodStateFor(periodNow, dayKey, s.currentMonthRebuild && !day.Equal(target))
		if err := s.replaceLocalRankingDay(ctx, window, aggregates, snapshotAt, &next, day.Equal(target)); err != nil {
			return err
		}
		s.lastSettledDay = next.LastSettledDay
		s.currentMonthRebuild = next.CurrentMonthRebuild
		settled++
	}
	return nil
}

func (s *LocalRankingService) ensureLocalRankingPeriodState(ctx context.Context, periodNow, snapshotAt time.Time) error {
	if s.periodStateLoaded {
		return nil
	}
	setting, found, err := repository.GetAppSetting(ctx, s.db.Clauses(dbresolver.Read), localRankingPeriodStateSettingKey)
	if err != nil {
		return err
	}
	if found {
		if setting.Value == nil {
			return fmt.Errorf("local ranking period state is empty")
		}
		var state localRankingPeriodState
		if err := json.Unmarshal([]byte(*setting.Value), &state); err != nil {
			return fmt.Errorf("decode local ranking period state: %w", err)
		}
		if _, err := time.ParseInLocation("2006-01-02", state.LastSettledDay, time.Local); err != nil {
			return fmt.Errorf("validate local ranking period state: %w", err)
		}
		zoneName, zoneOffset := localRankingZone(periodNow)
		if state.PeriodTimezone != zoneName || state.PeriodOffsetSeconds != zoneOffset {
			// 期间时区变化后旧键快照已不对应新日历:清空并从上一月窗口开始重算。
			state = localRankingPeriodStateFor(periodNow, localRankingResettlementStartDay(periodNow).Format("2006-01-02"), true)
			if err := s.db.Clauses(dbresolver.Write).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Where("1 = 1").Delete(&entities.LocalRankingPeriodStat{}).Error; err != nil {
					return fmt.Errorf("reset local ranking snapshots for timezone change: %w", err)
				}
				return saveLocalRankingPeriodStateTx(tx, state, snapshotAt)
			}); err != nil {
				return err
			}
			s.lastDynamicDay = ""
			s.lastDynamicAggregates = nil
		}
		s.lastSettledDay = state.LastSettledDay
		s.currentMonthRebuild = state.CurrentMonthRebuild
		s.periodStateLoaded = true
		return nil
	}

	// 首次启用从今天开始，直接把昨天标记为边界，绝不补齐上线前周期。
	initial := localRankingPeriodStateFor(periodNow, localRankingStartOfDay(periodNow).AddDate(0, 0, -1).Format("2006-01-02"), false)
	if err := s.db.Clauses(dbresolver.Write).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return saveLocalRankingPeriodStateTx(tx, initial, snapshotAt)
	}); err != nil {
		return err
	}
	s.lastSettledDay = initial.LastSettledDay
	s.currentMonthRebuild = initial.CurrentMonthRebuild
	s.lastDynamicDay = ""
	s.lastDynamicAggregates = nil
	s.periodStateLoaded = true
	return nil
}

// localRankingZone 返回本地期间时区的名称与 offset,两者共同决定日历边界。
func localRankingZone(periodNow time.Time) (string, int) {
	_, offset := periodNow.Zone()
	return time.Local.String(), offset
}

// localRankingPeriodStateFor 组装期间状态,始终写入当前本地期间时区指纹。
func localRankingPeriodStateFor(periodNow time.Time, lastSettledDay string, currentMonthRebuild bool) localRankingPeriodState {
	name, offset := localRankingZone(periodNow)
	return localRankingPeriodState{
		LastSettledDay: lastSettledDay, PeriodTimezone: name, PeriodOffsetSeconds: offset,
		CurrentMonthRebuild: currentMonthRebuild,
	}
}

// saveLocalRankingPeriodState 持久化期间状态并同步内存游标,供读取路径与后台批次共享。
func (s *LocalRankingService) saveLocalRankingPeriodState(ctx context.Context, periodNow time.Time, lastSettledDay string, currentMonthRebuild bool, snapshotAt time.Time) error {
	state := localRankingPeriodStateFor(periodNow, lastSettledDay, currentMonthRebuild)
	if err := s.db.Clauses(dbresolver.Write).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return saveLocalRankingPeriodStateTx(tx, state, snapshotAt)
	}); err != nil {
		return err
	}
	s.lastSettledDay = state.LastSettledDay
	s.currentMonthRebuild = state.CurrentMonthRebuild
	return nil
}

func (s *LocalRankingService) aggregateLocalRankingWindow(ctx context.Context, window localRankingPeriodWindow) ([]localRankingRawAggregate, error) {
	if !window.End.After(window.Start) {
		return []localRankingRawAggregate{}, nil
	}
	predicate, arguments := rankingTimeRangePredicate(window.Start, window.End)
	query := fmt.Sprintf(`
WITH ranged_events AS (
	SELECT api_group_key, timestamp, failed, latency_ms, ttft_ms, input_tokens, cache_read_tokens, total_tokens
	FROM usage_events
	WHERE %s
), five_minute_buckets AS (
	SELECT
		keys.id AS api_key_id,
		CAST(strftime('%%s', events.timestamp) AS INTEGER) / 300 AS bucket_key,
		COUNT(*) AS request_count,
		SUM(CASE WHEN events.failed = 0 THEN 1 ELSE 0 END) AS success_count,
		SUM(CASE WHEN events.failed <> 0 THEN 1 ELSE 0 END) AS failure_count,
		SUM(events.input_tokens) AS input_tokens,
		SUM(events.cache_read_tokens) AS cache_read_tokens,
		SUM(events.total_tokens) AS total_tokens,
		SUM(CASE WHEN events.failed = 0 AND events.ttft_ms IS NOT NULL AND events.ttft_ms > 0 AND events.latency_ms > 0 THEN events.ttft_ms ELSE 0 END) AS ttft_sum_ms,
		SUM(CASE WHEN events.failed = 0 AND events.ttft_ms IS NOT NULL AND events.ttft_ms > 0 AND events.latency_ms > 0 THEN 1 ELSE 0 END) AS ttft_sample_count,
		SUM(CASE WHEN events.failed = 0 AND events.ttft_ms IS NOT NULL AND events.ttft_ms > 0 AND events.latency_ms > 0 THEN events.latency_ms ELSE 0 END) AS latency_sum_ms,
		SUM(CASE WHEN events.failed = 0 AND events.ttft_ms IS NOT NULL AND events.ttft_ms > 0 AND events.latency_ms > 0 THEN 1 ELSE 0 END) AS latency_sample_count
	FROM ranged_events AS events
	JOIN cpa_api_keys AS keys ON keys.api_key = TRIM(events.api_group_key)
	GROUP BY keys.id, bucket_key
)
SELECT
	api_key_id,
	COALESCE(SUM(request_count), 0) AS request_count,
	COALESCE(SUM(success_count), 0) AS success_count,
	COALESCE(SUM(failure_count), 0) AS failure_count,
	COALESCE(SUM(input_tokens), 0) AS input_tokens,
	COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens,
	COALESCE(SUM(total_tokens), 0) AS total_tokens,
	COALESCE(SUM(ttft_sum_ms), 0) AS ttft_sum_ms,
	COALESCE(SUM(ttft_sample_count), 0) AS ttft_sample_count,
	COALESCE(SUM(latency_sum_ms), 0) AS latency_sum_ms,
	COALESCE(SUM(latency_sample_count), 0) AS latency_sample_count,
	COALESCE(MAX(request_count), 0) AS peak_5m_request_count,
	COALESCE(MAX(total_tokens), 0) AS peak_5m_total_tokens
FROM five_minute_buckets
GROUP BY api_key_id
ORDER BY api_key_id ASC`, predicate)
	var rows []localRankingRawAggregate
	if err := s.db.Clauses(dbresolver.Read).WithContext(ctx).Raw(query, arguments...).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("aggregate local ranking day %s: %w", window.Key, err)
	}
	return rows, nil
}

func (s *LocalRankingService) replaceLocalRankingDay(
	ctx context.Context,
	window localRankingPeriodWindow,
	aggregates []localRankingRawAggregate,
	snapshotAt time.Time,
	state *localRankingPeriodState,
	prune bool,
) error {
	monthKey := window.Start.In(time.Local).Format("2006-01")
	return s.db.Clauses(dbresolver.Write).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tx = tx.Session(&gorm.Session{NowFunc: func() time.Time { return snapshotAt }})
		var existing []entities.LocalRankingPeriodStat
		if err := tx.Where(
			"(period_kind = ? AND period_key = ?) OR (period_kind = ? AND period_key = ?)",
			entities.LocalRankingPeriodDay, window.Key,
			entities.LocalRankingPeriodMonth, monthKey,
		).Find(&existing).Error; err != nil {
			return fmt.Errorf("load local ranking day and month snapshots: %w", err)
		}
		dayByKey := make(map[int64]entities.LocalRankingPeriodStat, len(existing))
		monthByKey := make(map[int64]entities.LocalRankingPeriodStat, len(existing))
		for _, row := range existing {
			if row.PeriodKind == entities.LocalRankingPeriodDay {
				dayByKey[row.APIKeyID] = row
			} else if row.PeriodKind == entities.LocalRankingPeriodMonth {
				monthByKey[row.APIKeyID] = row
			}
		}

		rows := make([]entities.LocalRankingPeriodStat, 0, len(aggregates)*2)
		for _, aggregate := range aggregates {
			newDay := localRankingMetricsFromRaw(aggregate)
			oldDay := localRankingMetricsFromStat(dayByKey[aggregate.APIKeyID])
			month := localRankingMetricsFromStat(monthByKey[aggregate.APIKeyID])
			month = replaceLocalRankingContribution(month, oldDay, newDay)
			rows = append(rows,
				localRankingStatFromMetrics(localRankingStatKey{Kind: entities.LocalRankingPeriodDay, Key: window.Key, APIKeyID: aggregate.APIKeyID}, newDay, snapshotAt),
				localRankingStatFromMetrics(localRankingStatKey{Kind: entities.LocalRankingPeriodMonth, Key: monthKey, APIKeyID: aggregate.APIKeyID}, month, snapshotAt),
			)
		}
		sort.Slice(rows, func(left, right int) bool {
			if rows[left].PeriodKind != rows[right].PeriodKind {
				return rows[left].PeriodKind < rows[right].PeriodKind
			}
			if rows[left].PeriodKey != rows[right].PeriodKey {
				return rows[left].PeriodKey < rows[right].PeriodKey
			}
			return rows[left].APIKeyID < rows[right].APIKeyID
		})
		if len(rows) > 0 {
			if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(rows, localRankingWriteBatchSize).Error; err != nil {
				return fmt.Errorf("replace local ranking day %s: %w", window.Key, err)
			}
		}
		if state != nil {
			if err := saveLocalRankingPeriodStateTx(tx, *state, snapshotAt); err != nil {
				return err
			}
		}
		if prune {
			if err := pruneLocalRankingPeriodsTx(tx, snapshotAt.In(time.Local)); err != nil {
				return err
			}
		}
		return nil
	})
}

func saveLocalRankingPeriodStateTx(tx *gorm.DB, state localRankingPeriodState, now time.Time) error {
	encoded, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode local ranking period state: %w", err)
	}
	value := string(encoded)
	setting := entities.AppSetting{
		SettingKey: localRankingPeriodStateSettingKey,
		Value:      &value,
		ValueType:  entities.AppSettingValueTypeJSON,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "setting_key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "value_type", "updated_at"}),
	}).Create(&setting).Error; err != nil {
		return fmt.Errorf("save local ranking period state: %w", err)
	}
	return nil
}

func replaceLocalRankingContribution(total, previous, current localRankingMetrics) localRankingMetrics {
	return localRankingMetrics{
		RequestCount:       replaceLocalRankingValue(total.RequestCount, previous.RequestCount, current.RequestCount),
		SuccessCount:       replaceLocalRankingValue(total.SuccessCount, previous.SuccessCount, current.SuccessCount),
		FailureCount:       replaceLocalRankingValue(total.FailureCount, previous.FailureCount, current.FailureCount),
		InputTokens:        replaceLocalRankingValue(total.InputTokens, previous.InputTokens, current.InputTokens),
		CacheReadTokens:    replaceLocalRankingValue(total.CacheReadTokens, previous.CacheReadTokens, current.CacheReadTokens),
		TotalTokens:        replaceLocalRankingValue(total.TotalTokens, previous.TotalTokens, current.TotalTokens),
		TTFTSumMS:          replaceLocalRankingValue(total.TTFTSumMS, previous.TTFTSumMS, current.TTFTSumMS),
		TTFTSampleCount:    replaceLocalRankingValue(total.TTFTSampleCount, previous.TTFTSampleCount, current.TTFTSampleCount),
		LatencySumMS:       replaceLocalRankingValue(total.LatencySumMS, previous.LatencySumMS, current.LatencySumMS),
		LatencySampleCount: replaceLocalRankingValue(total.LatencySampleCount, previous.LatencySampleCount, current.LatencySampleCount),
		Peak5MRequestCount: max(total.Peak5MRequestCount, current.Peak5MRequestCount),
		Peak5MTotalTokens:  max(total.Peak5MTotalTokens, current.Peak5MTotalTokens),
	}
}

func replaceLocalRankingValue(total, previous, current int64) int64 {
	base := total - min(total, previous)
	if current > 0 && base > math.MaxInt64-current {
		return math.MaxInt64
	}
	return base + current
}

func pruneLocalRankingPeriodsTx(tx *gorm.DB, now time.Time) error {
	windows := localRankingPeriodWindows(now)
	dayKeys := []string{windows[0].Key, windows[1].Key}
	monthKeys := []string{windows[2].Key, windows[3].Key}
	// 日行是月累计的账本:本月与上月的日行必须保留,否则跨批次重复结算同一天时
	// 会把该天的贡献重复计入月快照。
	result := tx.Where(
		"(period_kind = ? AND period_key NOT IN ? AND substr(period_key, 1, 7) NOT IN ?) OR (period_kind = ? AND period_key NOT IN ?)",
		entities.LocalRankingPeriodDay, dayKeys, monthKeys, entities.LocalRankingPeriodMonth, monthKeys,
	).Delete(&entities.LocalRankingPeriodStat{})
	if result.Error != nil {
		return fmt.Errorf("prune local ranking period stats: %w", result.Error)
	}
	return nil
}

// Leaderboard 返回当前 Keeper 内 API Key 的本地排行榜。
func (s *LocalRankingService) Leaderboard(ctx context.Context, period LeaderboardPeriod, metric LeaderboardMetric) (Leaderboard, error) {
	if s == nil || s.db == nil || s.now == nil {
		return Leaderboard{}, fmt.Errorf("local ranking service is not configured")
	}
	if !validLeaderboardPeriod(period) || !validLeaderboardMetric(metric) {
		return Leaderboard{}, ErrInvalidLeaderboard
	}
	snapshotAt := timeutil.NormalizeStorageTime(s.now())
	window, ok := localRankingWindowForPeriod(snapshotAt.In(time.Local), period)
	if !ok {
		return Leaderboard{}, ErrInvalidLeaderboard
	}
	if err := s.ensureCurrentLocalRankingSnapshot(ctx, window); err != nil {
		return Leaderboard{}, err
	}
	rows, err := s.loadLocalRankingPopulation(ctx, window)
	if err != nil {
		return Leaderboard{}, err
	}
	entries := buildLocalLeaderboardEntries(rows, metric)
	generatedAt := time.Time{}
	for _, row := range rows {
		if generatedAt.IsZero() || row.UpdatedAt.After(generatedAt) {
			generatedAt = row.UpdatedAt
		}
	}
	if generatedAt.IsZero() {
		generatedAt = snapshotAt
	}
	board := Leaderboard{
		Period: period, PeriodKey: window.Key, Metric: metric, GeneratedAt: generatedAt, Stale: false, Entries: entries,
	}
	if metric == MetricOverall {
		board.ScoreExplanation = localOverallScoreExplanation()
	}
	return board, nil
}

func (s *LocalRankingService) ensureCurrentLocalRankingSnapshot(ctx context.Context, window localRankingPeriodWindow) error {
	if window.Period != LeaderboardToday && window.Period != LeaderboardYesterday && window.Period != LeaderboardCurrentMonth {
		return nil
	}
	snapshotAt := timeutil.NormalizeStorageTime(s.now())
	periodNow := snapshotAt.In(time.Local)
	s.aggregateMu.Lock()
	defer s.aggregateMu.Unlock()
	if err := s.ensureLocalRankingPeriodState(ctx, periodNow, snapshotAt); err != nil {
		return err
	}
	switch window.Period {
	case LeaderboardToday:
		empty, err := s.localRankingWindowEmpty(ctx, window)
		if err != nil || !empty {
			return err
		}
		return s.refreshLocalRankingToday(ctx, periodNow, snapshotAt, true)
	case LeaderboardYesterday:
		empty, err := s.localRankingWindowEmpty(ctx, window)
		if err != nil || !empty {
			return err
		}
		if periodNow.Hour() < localRankingSettleGraceHour {
			return nil
		}
		day := localRankingStartOfDay(window.Start)
		return s.settleLocalRankingDayRange(ctx, day, day.AddDate(0, 0, 1), snapshotAt)
	default:
		rangeEnd := localRankingStartOfDay(periodNow)
		if periodNow.Hour() < localRankingSettleGraceHour {
			// 昨天还在宽限期内,本月窗口先只覆盖更早的完整日。
			rangeEnd = rangeEnd.AddDate(0, 0, -1)
		}
		if s.currentMonthRebuild {
			// 期间时区变化后本月窗口必须在新日历下完整重算,否则会返回只累加了一部分
			// 日贡献的快照;范围止于本月完整日,不涉及更早的历史。
			if err := s.settleLocalRankingDayRange(ctx, localRankingMonthStart(periodNow), rangeEnd, snapshotAt); err != nil {
				return err
			}
		}
		empty, err := s.localRankingWindowEmpty(ctx, localRankingTodayWindow(periodNow))
		if err != nil {
			return err
		}
		if empty {
			if err := s.refreshLocalRankingToday(ctx, periodNow, snapshotAt, true); err != nil {
				return err
			}
		}
		if s.currentMonthRebuild {
			return s.saveLocalRankingPeriodState(ctx, periodNow, s.lastSettledDay, false, snapshotAt)
		}
		return nil
	}
}

// localRankingWindowEmpty 判断窗口是否还没有写入任何快照行。
func (s *LocalRankingService) localRankingWindowEmpty(ctx context.Context, window localRankingPeriodWindow) (bool, error) {
	var count int64
	if err := s.db.Clauses(dbresolver.Read).WithContext(ctx).
		Model(&entities.LocalRankingPeriodStat{}).
		Where("period_kind = ? AND period_key = ?", window.Kind, window.Key).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("check current local ranking snapshot: %w", err)
	}
	return count == 0, nil
}

// settleLocalRankingDayRange 结算半开区间 [start, end) 内的完整自然日,按天幂等覆盖,
// 不移动后台补算游标。
func (s *LocalRankingService) settleLocalRankingDayRange(ctx context.Context, start, end, snapshotAt time.Time) error {
	for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
		window := localRankingCompleteDayWindow(day)
		aggregates, err := s.aggregateLocalRankingWindow(ctx, window)
		if err != nil {
			return err
		}
		if err := s.replaceLocalRankingDay(ctx, window, aggregates, snapshotAt, nil, false); err != nil {
			return err
		}
	}
	return nil
}

func (s *LocalRankingService) loadLocalRankingPopulation(ctx context.Context, window localRankingPeriodWindow) ([]localRankingPopulationRow, error) {
	type row struct {
		APIKeyID             int64     `gorm:"column:api_key_id"`
		APIKey               string    `gorm:"column:api_key"`
		KeyName              string    `gorm:"column:key_name"`
		DisplayKey           string    `gorm:"column:display_key"`
		KeyAlias             string    `gorm:"column:key_alias"`
		LocalRankingAvatarID *uint8    `gorm:"column:local_ranking_avatar_id"`
		RequestCount         int64     `gorm:"column:request_count"`
		SuccessCount         int64     `gorm:"column:success_count"`
		FailureCount         int64     `gorm:"column:failure_count"`
		InputTokens          int64     `gorm:"column:input_tokens"`
		CacheReadTokens      int64     `gorm:"column:cache_read_tokens"`
		TotalTokens          int64     `gorm:"column:total_tokens"`
		TTFTSumMS            int64     `gorm:"column:ttft_sum_ms"`
		TTFTSampleCount      int64     `gorm:"column:ttft_sample_count"`
		LatencySumMS         int64     `gorm:"column:latency_sum_ms"`
		LatencySampleCount   int64     `gorm:"column:latency_sample_count"`
		Peak5MRequestCount   int64     `gorm:"column:peak_5m_request_count"`
		Peak5MTotalTokens    int64     `gorm:"column:peak_5m_total_tokens"`
		UpdatedAt            time.Time `gorm:"column:updated_at"`
	}
	var loaded []row
	if err := s.db.Clauses(dbresolver.Read).WithContext(ctx).
		Table("local_ranking_period_stats AS stats").
		Select("stats.api_key_id, keys.api_key, keys.key_name, keys.display_key, keys.key_alias, keys.local_ranking_avatar_id, stats.request_count, stats.success_count, stats.failure_count, stats.input_tokens, stats.cache_read_tokens, stats.total_tokens, stats.ttft_sum_ms, stats.ttft_sample_count, stats.latency_sum_ms, stats.latency_sample_count, stats.peak_5m_request_count, stats.peak_5m_total_tokens, stats.updated_at").
		Joins("JOIN cpa_api_keys AS keys ON keys.id = stats.api_key_id").
		Where("stats.period_kind = ? AND stats.period_key = ?", window.Kind, window.Key).
		Order("stats.api_key_id ASC").
		Scan(&loaded).Error; err != nil {
		return nil, fmt.Errorf("load local ranking population: %w", err)
	}
	result := make([]localRankingPopulationRow, 0, len(loaded))
	for _, item := range loaded {
		result = append(result, localRankingPopulationRow{
			APIKeyID: item.APIKeyID, APIKey: item.APIKey, KeyName: item.KeyName, DisplayKey: item.DisplayKey, KeyAlias: item.KeyAlias, LocalRankingAvatarID: item.LocalRankingAvatarID, UpdatedAt: item.UpdatedAt,
			localRankingMetrics: localRankingMetrics{
				RequestCount: item.RequestCount, SuccessCount: item.SuccessCount, FailureCount: item.FailureCount,
				InputTokens: item.InputTokens, CacheReadTokens: item.CacheReadTokens, TotalTokens: item.TotalTokens,
				TTFTSumMS: item.TTFTSumMS, TTFTSampleCount: item.TTFTSampleCount,
				LatencySumMS: item.LatencySumMS, LatencySampleCount: item.LatencySampleCount,
				Peak5MRequestCount: item.Peak5MRequestCount, Peak5MTotalTokens: item.Peak5MTotalTokens,
			},
		})
	}
	return result, nil
}
