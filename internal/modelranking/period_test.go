package modelranking

import (
	"testing"
	"time"
)

func TestAllPeriodsDisplayOrder(t *testing.T) {
	want := []Period{
		PeriodToday, PeriodYesterday, PeriodCurrentWeek,
		PeriodPreviousWeek, PeriodCurrentMonth, PeriodPreviousMonth,
	}
	got := AllPeriods()
	if len(got) != len(want) {
		t.Fatalf("AllPeriods() length = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("AllPeriods()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
}

func TestParsePeriod(t *testing.T) {
	for _, period := range AllPeriods() {
		if got, err := ParsePeriod(string(period)); err != nil || got != period {
			t.Fatalf("ParsePeriod(%q) = %q, %v; want %q", period, got, err, period)
		}
	}
	for _, invalid := range []string{"", "all", "this_week", "TODAY", "current_day"} {
		if _, err := ParsePeriod(invalid); err == nil {
			t.Fatalf("ParsePeriod(%q) unexpectedly accepted", invalid)
		}
	}
}

func TestResolveWindow(t *testing.T) {
	// 2026-11-16 is a Monday.
	now := time.Date(2026, 11, 16, 15, 30, 0, 0, time.Local)

	tests := []struct {
		period    Period
		kind      PeriodKind
		wantStart string
		wantEnd   string
		wantKey   string
	}{
		{PeriodToday, PeriodKindDay, "2026-11-16T00:00:00", "2026-11-17T00:00:00", "2026-11-16"},
		{PeriodYesterday, PeriodKindDay, "2026-11-15T00:00:00", "2026-11-16T00:00:00", "2026-11-15"},
		{PeriodCurrentWeek, PeriodKindWeek, "2026-11-16T00:00:00", "2026-11-23T00:00:00", "2026-W47"},
		{PeriodPreviousWeek, PeriodKindWeek, "2026-11-09T00:00:00", "2026-11-16T00:00:00", "2026-W46"},
		{PeriodCurrentMonth, PeriodKindMonth, "2026-11-01T00:00:00", "2026-12-01T00:00:00", "2026-11"},
		{PeriodPreviousMonth, PeriodKindMonth, "2026-10-01T00:00:00", "2026-11-01T00:00:00", "2026-10"},
	}

	for _, tt := range tests {
		window, err := ResolveWindow(tt.period, now)
		if err != nil {
			t.Fatalf("ResolveWindow(%q): %v", tt.period, err)
		}
		if window.Kind != tt.kind {
			t.Errorf("%s kind = %q, want %q", tt.period, window.Kind, tt.kind)
		}
		if got := window.Start.Format("2006-01-02T15:04:05"); got != tt.wantStart {
			t.Errorf("%s start = %s, want %s", tt.period, got, tt.wantStart)
		}
		if got := window.End.Format("2006-01-02T15:04:05"); got != tt.wantEnd {
			t.Errorf("%s end = %s, want %s", tt.period, got, tt.wantEnd)
		}
		if window.Key != tt.wantKey {
			t.Errorf("%s key = %s, want %s", tt.period, window.Key, tt.wantKey)
		}
	}
}

func TestResolveWindowWeekStartsOnMonday(t *testing.T) {
	// 2026-11-15 is a Sunday; its week starts Monday 2026-11-09.
	now := time.Date(2026, 11, 15, 23, 59, 0, 0, time.Local)
	window, err := ResolveWindow(PeriodCurrentWeek, now)
	if err != nil {
		t.Fatalf("ResolveWindow: %v", err)
	}
	if got := window.Start.Format("2006-01-02"); got != "2026-11-09" {
		t.Fatalf("week start = %s, want 2026-11-09", got)
	}
	if got := window.End.Format("2006-01-02"); got != "2026-11-16" {
		t.Fatalf("week end = %s, want 2026-11-16", got)
	}
}

func TestResolveWindowMonthArithmetic(t *testing.T) {
	// The 31st must roll into a shorter previous month without overflowing.
	now := time.Date(2026, 12, 31, 23, 0, 0, 0, time.Local)
	current, err := ResolveWindow(PeriodCurrentMonth, now)
	if err != nil {
		t.Fatalf("ResolveWindow current month: %v", err)
	}
	if got := current.Start.Format("2006-01-02"); got != "2026-12-01" {
		t.Fatalf("current month start = %s, want 2026-12-01", got)
	}
	if got := current.End.Format("2006-01-02"); got != "2027-01-01" {
		t.Fatalf("current month end = %s, want 2027-01-01", got)
	}

	previous, err := ResolveWindow(PeriodPreviousMonth, now)
	if err != nil {
		t.Fatalf("ResolveWindow previous month: %v", err)
	}
	if got := previous.Start.Format("2006-01-02"); got != "2026-11-01" {
		t.Fatalf("previous month start = %s, want 2026-11-01", got)
	}
	if got := previous.End.Format("2006-01-02"); got != "2026-12-01" {
		t.Fatalf("previous month end = %s, want 2026-12-01", got)
	}
}

func TestResolveWindowUsesProjectTimezone(t *testing.T) {
	original := time.Local
	time.Local = time.FixedZone("Test/Plus3", 3*60*60)
	t.Cleanup(func() { time.Local = original })

	// 22:30 UTC is already the next day in the project TZ.
	now := time.Date(2026, 11, 15, 22, 30, 0, 0, time.UTC)
	window, err := ResolveWindow(PeriodToday, now)
	if err != nil {
		t.Fatalf("ResolveWindow: %v", err)
	}
	if got := window.Start.Format("2006-01-02T15:04:05"); got != "2026-11-16T00:00:00" {
		t.Fatalf("today start = %s, want 2026-11-16T00:00:00", got)
	}
	if _, offset := window.Start.Zone(); offset != 3*60*60 {
		t.Fatalf("window start offset = %d, want +03:00", offset)
	}
}

func TestResolveWindowEndIsExclusive(t *testing.T) {
	for _, period := range AllPeriods() {
		now := time.Date(2026, 11, 16, 9, 0, 0, 0, time.Local)
		window, err := ResolveWindow(period, now)
		if err != nil {
			t.Fatalf("ResolveWindow(%q): %v", period, err)
		}
		if !window.End.After(window.Start) {
			t.Fatalf("%s end %v must be after start %v", period, window.End, window.Start)
		}
		if window.Kind == PeriodKindDay {
			if next := window.Start.AddDate(0, 0, 1); !next.Equal(window.End) {
				t.Fatalf("%s day end %v must be exclusive (want %v)", period, window.End, next)
			}
		}
	}
}

func TestResolveWindowRejectsUnknown(t *testing.T) {
	if _, err := ResolveWindow(Period("bogus"), time.Now()); err == nil {
		t.Fatal("expected error for unknown period")
	}
}