package ranking

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
)

func TestLocalLeaderboardBuildsMissingTodaySnapshotForCurrentMonth(t *testing.T) {
	now := time.Date(2026, 10, 22, 19, 30, 0, 0, localRankingPeriodLocation)
	db, err := repository.OpenDatabase(config.Config{SQLitePath: filepath.Join(t.TempDir(), "ranking.db")})
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

	key := entities.CPAAPIKey{APIKey: "sk-current-ranking-key", DisplayKey: "sk-*********key"}
	if err := db.Create(&key).Error; err != nil {
		t.Fatalf("create API key: %v", err)
	}
	ttft := int64(120)
	event := entities.UsageEvent{
		EventKey: "current-ranking-event", APIGroupKey: key.APIKey, AuthType: "apikey",
		Timestamp: now.Add(-time.Minute), Failed: false, LatencyMS: 300, TTFTMS: &ttft,
		InputTokens: 100, CacheReadTokens: 20, TotalTokens: 160,
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}
	month := localRankingWindowForTest(t, now, LeaderboardCurrentMonth)
	if err := db.Create(&entities.LocalRankingPeriodStat{
		PeriodKind: month.Kind, PeriodKey: month.Key, APIKeyID: key.ID,
		RequestCount: 1, SuccessCount: 1, InputTokens: 10, TotalTokens: 10,
		TTFTSumMS: 100, TTFTSampleCount: 1, LatencySumMS: 100, LatencySampleCount: 1,
		Peak5MRequestCount: 1, Peak5MTotalTokens: 10, UpdatedAt: now.Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("create existing month snapshot: %v", err)
	}

	service, err := NewLocalRankingService(db, LocalRankingServiceOptions{Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("create local ranking service: %v", err)
	}
	board, err := service.Leaderboard(context.Background(), LeaderboardCurrentMonth, MetricOverall)
	if err != nil {
		t.Fatalf("load current-month leaderboard: %v", err)
	}
	if len(board.Entries) != 1 || board.Entries[0].ParticipantID != "1" {
		t.Fatalf("expected the current key without waiting for the runner, got %+v", board.Entries)
	}

	var stats []entities.LocalRankingPeriodStat
	if err := db.Where("api_key_id = ?", key.ID).Find(&stats).Error; err != nil {
		t.Fatalf("load local ranking stats: %v", err)
	}
	if len(stats) != 2 {
		t.Fatalf("expected day and month snapshots, got %+v", stats)
	}
}

func localRankingWindowForTest(t *testing.T, now time.Time, period LeaderboardPeriod) localRankingPeriodWindow {
	t.Helper()
	window, ok := localRankingWindowForPeriod(now, period)
	if !ok {
		t.Fatalf("resolve %s window", period)
	}
	return window
}
