package modelranking

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
)

func TestServiceLeaderboardAggregatesAndCaches(t *testing.T) {
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "modelranking-service.db")})
	if err != nil {
		t.Fatalf("open database pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatalf("load reader pool: %v", err)
	}
	t.Cleanup(func() { _ = readerSQL.Close() })

	clock := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	current := clock
	alias := "gpt-5"
	event := entities.UsageEvent{
		EventKey: "service-1", Model: "openai/gpt-5", ModelAlias: &alias, Provider: "openai",
		Timestamp: clock, TotalTokens: 100, InputTokens: 20, CacheReadTokens: 10,
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}

	service, err := NewServiceWithOptions(db, ServiceOptions{Now: func() time.Time { return current }})
	if err != nil {
		t.Fatalf("NewServiceWithOptions: %v", err)
	}
	first, err := service.Leaderboard(context.Background(), PeriodToday, MetricTotalTokens)
	if err != nil {
		t.Fatalf("Leaderboard: %v", err)
	}
	if len(first.Entries) != 1 || first.Entries[0].Model != "gpt-5" || first.Entries[0].Value != 100 {
		t.Fatalf("entries = %+v, want gpt-5 with 100 tokens", first.Entries)
	}
	if first.Period != PeriodToday || first.Metric != MetricTotalTokens {
		t.Fatalf("selection mismatch: %+v", first)
	}
	if first.Timezone != time.Local.String() {
		t.Fatalf("timezone = %q, want %q", first.Timezone, time.Local.String())
	}
	if !first.GeneratedAt.Equal(clock.UTC()) {
		t.Fatalf("generated_at = %v, want %v", first.GeneratedAt, clock.UTC())
	}

	// Inside the 30-second TTL the cached response is returned unchanged.
	current = clock.Add(10 * time.Second)
	second, err := service.Leaderboard(context.Background(), PeriodToday, MetricTotalTokens)
	if err != nil {
		t.Fatalf("Leaderboard (cached): %v", err)
	}
	if !second.GeneratedAt.Equal(first.GeneratedAt) {
		t.Fatalf("expected a cache hit, generated_at moved from %v to %v", first.GeneratedAt, second.GeneratedAt)
	}

	// After the TTL the window is recomputed.
	current = clock.Add(31 * time.Second)
	third, err := service.Leaderboard(context.Background(), PeriodToday, MetricTotalTokens)
	if err != nil {
		t.Fatalf("Leaderboard (expired): %v", err)
	}
	if !third.GeneratedAt.Equal(current.UTC()) {
		t.Fatalf("expected a cache miss, generated_at = %v, want %v", third.GeneratedAt, current.UTC())
	}

	// A different metric uses a different cache key and an empty period is not an error.
	other, err := service.Leaderboard(context.Background(), PeriodPreviousMonth, MetricRequestCount)
	if err != nil {
		t.Fatalf("Leaderboard (previous month): %v", err)
	}
	if other.Entries == nil || len(other.Entries) != 0 {
		t.Fatalf("empty period entries = %+v, want an empty list", other.Entries)
	}
}

func TestServiceLeaderboardRejectsInvalidSelection(t *testing.T) {
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "modelranking-invalid.db")})
	if err != nil {
		t.Fatalf("open database pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatalf("load reader pool: %v", err)
	}
	t.Cleanup(func() { _ = readerSQL.Close() })
	service, err := NewService(db)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	for _, tt := range []struct {
		period Period
		metric Metric
	}{
		{Period("bogus"), MetricTotalTokens},
		{PeriodToday, Metric("overall")},
		{PeriodToday, Metric("tokens")},
	} {
		if _, err := service.Leaderboard(context.Background(), tt.period, tt.metric); !errors.Is(err, ErrInvalidSelection) {
			t.Fatalf("Leaderboard(%q, %q) error = %v, want ErrInvalidSelection", tt.period, tt.metric, err)
		}
	}
}

func TestNewServiceRejectsNilDatabase(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("expected an error for a nil database")
	}
}