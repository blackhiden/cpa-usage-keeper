package ranking

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	"gorm.io/gorm"
)

func openLocalRankingDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := repository.OpenDatabase(config.Config{SQLitePath: filepath.Join(t.TempDir(), "local-ranking.db")})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database handle: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})
	return db
}

func seedLocalRankingKey(t *testing.T, db *gorm.DB, apiKey string) entities.CPAAPIKey {
	t.Helper()
	key := entities.CPAAPIKey{APIKey: apiKey, DisplayKey: "sk-*********recovery"}
	if err := db.Create(&key).Error; err != nil {
		t.Fatalf("create API key: %v", err)
	}
	return key
}

// seedLocalRankingEvents 写入 count 个事件,每个事件贡献 160 total tokens。
func seedLocalRankingEvents(t *testing.T, db *gorm.DB, apiKey string, at time.Time, count int) {
	t.Helper()
	ttft := int64(120)
	events := make([]entities.UsageEvent, 0, count)
	for index := 0; index < count; index++ {
		events = append(events, entities.UsageEvent{
			EventKey:    fmt.Sprintf("%s-%d", at.Format("20060102150405"), index),
			APIGroupKey: apiKey, AuthType: "apikey",
			Timestamp: at.Add(time.Duration(index) * time.Second),
			Failed:    false, LatencyMS: 300, TTFTMS: &ttft,
			InputTokens: 100, CacheReadTokens: 20, TotalTokens: 160,
		})
	}
	if err := db.CreateInBatches(events, 100).Error; err != nil {
		t.Fatalf("create usage events: %v", err)
	}
}

func seedLocalRankingState(t *testing.T, db *gorm.DB, state localRankingPeriodState) {
	t.Helper()
	encoded, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("encode period state: %v", err)
	}
	value := string(encoded)
	setting := entities.AppSetting{
		SettingKey: localRankingPeriodStateSettingKey, Value: &value,
		ValueType: entities.AppSettingValueTypeJSON,
	}
	if err := db.Create(&setting).Error; err != nil {
		t.Fatalf("create period state: %v", err)
	}
}

func seedLocalRankingStat(t *testing.T, db *gorm.DB, kind entities.LocalRankingPeriodKind, key string, apiKeyID int64, now time.Time) {
	t.Helper()
	if err := db.Create(&entities.LocalRankingPeriodStat{
		PeriodKind: kind, PeriodKey: key, APIKeyID: apiKeyID,
		RequestCount: 1, SuccessCount: 1, InputTokens: 10, TotalTokens: 10,
		TTFTSumMS: 100, TTFTSampleCount: 1, LatencySumMS: 100, LatencySampleCount: 1,
		Peak5MRequestCount: 1, Peak5MTotalTokens: 10, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create local ranking stat: %v", err)
	}
}

func loadLocalRankingState(t *testing.T, db *gorm.DB) localRankingPeriodState {
	t.Helper()
	setting, found, err := repository.GetAppSetting(context.Background(), db, localRankingPeriodStateSettingKey)
	if err != nil {
		t.Fatalf("load period state: %v", err)
	}
	if !found || setting.Value == nil {
		return localRankingPeriodState{}
	}
	var state localRankingPeriodState
	if err := json.Unmarshal([]byte(*setting.Value), &state); err != nil {
		t.Fatalf("decode period state: %v", err)
	}
	return state
}

func countLocalRankingStats(t *testing.T, db *gorm.DB, kind entities.LocalRankingPeriodKind, key string) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&entities.LocalRankingPeriodStat{}).
		Where("period_kind = ? AND period_key = ?", kind, key).
		Count(&count).Error; err != nil {
		t.Fatalf("count local ranking stats: %v", err)
	}
	return count
}

// 升级前写入的期间状态没有时区指纹:重建必须立刻恢复今日榜单,而不是等完整日补算跑完。
func TestLocalLeaderboardRecoversImmediatelyAfterLegacyPeriodState(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 30, 0, 0, time.Local)
	db := openLocalRankingDB(t)
	key := seedLocalRankingKey(t, db, "sk-recovery-legacy")
	seedLocalRankingEvents(t, db, key.APIKey, now.Add(-5*time.Minute), 1)
	seedLocalRankingState(t, db, localRankingPeriodState{LastSettledDay: now.AddDate(0, 0, -1).Format("2006-01-02")})
	seedLocalRankingStat(t, db, entities.LocalRankingPeriodDay, now.Format("2006-01-02"), key.ID, now)

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	if err := service.AggregateOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once: %v", err)
	}
	if got := countLocalRankingStats(t, db, entities.LocalRankingPeriodDay, now.Format("2006-01-02")); got != 1 {
		t.Fatalf("expected today snapshot after aggregation, got %d rows", got)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardToday); got != 160 {
		t.Fatalf("expected today leaderboard value 160, got %d", got)
	}
	state := loadLocalRankingState(t, db)
	if state.LastSettledDay == "" || state.PeriodTimezone == "" {
		t.Fatalf("expected a refreshed period state, got %+v", state)
	}
	start := localRankingResettlementStartDay(now)
	settled := parseLocalRankingDayForTest(t, state.LastSettledDay)
	if settled.Before(start) {
		t.Fatalf("expected settlement to start at %s, got cursor %s", start.Format("2006-01-02"), state.LastSettledDay)
	}
	if settled.After(start.AddDate(0, 0, localRankingSettleDayLimit)) {
		t.Fatalf("expected a bounded settlement window, got cursor %s", state.LastSettledDay)
	}
}

// 期间时区名称相同但 offset 变化(容器 /etc/localtime 变化)也必须触发重建。
func TestLocalLeaderboardResetsWhenPeriodZoneOffsetChanges(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 30, 0, 0, time.Local)
	db := openLocalRankingDB(t)
	key := seedLocalRankingKey(t, db, "sk-recovery-offset")
	seedLocalRankingEvents(t, db, key.APIKey, now.Add(-5*time.Minute), 1)
	stale := seedLocalRankingKey(t, db, "sk-recovery-stale")
	_, zoneOffset := localRankingZone(now)
	seedLocalRankingState(t, db, localRankingPeriodState{
		LastSettledDay:      now.AddDate(0, 0, -1).Format("2006-01-02"),
		PeriodTimezone:      time.Local.String(),
		PeriodOffsetSeconds: zoneOffset + 3600,
	})
	seedLocalRankingStat(t, db, entities.LocalRankingPeriodDay, now.Format("2006-01-02"), stale.ID, now)

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	if err := service.AggregateOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once: %v", err)
	}
	if count := countLocalRankingStats(t, db, entities.LocalRankingPeriodDay, now.Format("2006-01-02")); count != 1 {
		t.Fatalf("expected stale snapshot to be dropped, got %d rows for today", count)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardToday); got != 160 {
		t.Fatalf("expected today leaderboard value 160 after zone change, got %d", got)
	}
	if state := loadLocalRankingState(t, db); state.PeriodOffsetSeconds != zoneOffset {
		t.Fatalf("expected stored offset %d, got %+v", zoneOffset, state)
	}
}

// 本地日历回退后聚合必须自愈:不能因为“状态超前”永久报错。
func TestLocalAggregationClampsStateAheadOfPeriodTime(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 30, 0, 0, time.Local)
	db := openLocalRankingDB(t)
	key := seedLocalRankingKey(t, db, "sk-recovery-ahead")
	seedLocalRankingEvents(t, db, key.APIKey, now.Add(-5*time.Minute), 1)
	_, zoneOffset := localRankingZone(now)
	seedLocalRankingState(t, db, localRankingPeriodState{
		LastSettledDay: now.Format("2006-01-02"), PeriodTimezone: time.Local.String(), PeriodOffsetSeconds: zoneOffset,
	})

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	for attempt := 1; attempt <= 2; attempt++ {
		if err := service.AggregateOnce(context.Background()); err != nil {
			t.Fatalf("aggregate once attempt %d: %v", attempt, err)
		}
	}
	if got := localRankingTotalTokens(t, service, LeaderboardToday); got != 160 {
		t.Fatalf("expected today leaderboard value 160, got %d", got)
	}
	if state := loadLocalRankingState(t, db); state.LastSettledDay != now.AddDate(0, 0, -1).Format("2006-01-02") {
		t.Fatalf("expected cursor clamped to yesterday, got %+v", state)
	}
}

// 期间时区变化后本月窗口必须按新日历完整重算,上一月窗口由后台批次重算。
func TestLocalCurrentMonthRebuildsAfterPeriodZoneChange(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 30, 0, 0, time.Local)
	db := openLocalRankingDB(t)
	key := seedLocalRankingKey(t, db, "sk-recovery-month")
	seedLocalRankingEvents(t, db, key.APIKey, time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local), 2)
	seedLocalRankingEvents(t, db, key.APIKey, time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local), 1)
	seedLocalRankingEvents(t, db, key.APIKey, now.Add(-5*time.Minute), 1)
	seedLocalRankingEvents(t, db, key.APIKey, time.Date(2026, 10, 20, 12, 0, 0, 0, time.Local), 3)
	seedLocalRankingState(t, db, localRankingPeriodState{
		LastSettledDay: now.AddDate(0, 0, -1).Format("2006-01-02"), PeriodTimezone: "Legacy/Ranking",
	})

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	if err := service.AggregateOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once: %v", err)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardCurrentMonth); got != 640 {
		t.Fatalf("expected rebuilt current month value 640, got %d", got)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardPreviousMonth); got != 480 {
		t.Fatalf("expected rebuilt previous month value 480, got %d", got)
	}
}

// 读取路径重算本月后,后台批次再次结算同一天不能重复计入月快照。
func TestLocalCurrentMonthRebuildIsIdempotentWithBackgroundSettlement(t *testing.T) {
	now := time.Date(2026, 11, 19, 19, 30, 0, 0, time.Local)
	db := openLocalRankingDB(t)
	key := seedLocalRankingKey(t, db, "sk-recovery-ledger")
	seedLocalRankingEvents(t, db, key.APIKey, time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local), 2)
	seedLocalRankingEvents(t, db, key.APIKey, now.Add(-5*time.Minute), 1)
	seedLocalRankingEvents(t, db, key.APIKey, time.Date(2026, 10, 20, 12, 0, 0, 0, time.Local), 3)
	seedLocalRankingState(t, db, localRankingPeriodState{
		LastSettledDay: now.AddDate(0, 0, -1).Format("2006-01-02"), PeriodTimezone: "Legacy/Ranking",
	})

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	if err := service.AggregateOnce(context.Background()); err != nil {
		t.Fatalf("aggregate once: %v", err)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardCurrentMonth); got != 480 {
		t.Fatalf("expected rebuilt current month value 480, got %d", got)
	}
	if err := service.AggregateOnce(context.Background()); err != nil {
		t.Fatalf("second aggregate once: %v", err)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardCurrentMonth); got != 480 {
		t.Fatalf("expected current month value 480 after re-settlement, got %d", got)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardPreviousMonth); got != 480 {
		t.Fatalf("expected previous month value 480, got %d", got)
	}
	if got := localRankingTotalTokens(t, service, LeaderboardYesterday); got != 0 {
		t.Fatalf("expected empty yesterday value, got %d", got)
	}
}

func localRankingTotalTokens(t *testing.T, service *LocalRankingService, period LeaderboardPeriod) int64 {
	t.Helper()
	board, err := service.Leaderboard(context.Background(), period, MetricTotalTokens)
	if err != nil {
		t.Fatalf("load %s leaderboard: %v", period, err)
	}
	var total int64
	for _, entry := range board.Entries {
		total += entry.Value
	}
	return total
}

func parseLocalRankingDayForTest(t *testing.T, value string) time.Time {
	t.Helper()
	day, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		t.Fatalf("parse local ranking day %q: %v", value, err)
	}
	return day
}
