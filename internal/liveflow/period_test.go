package liveflow

import (
	"testing"
	"time"
)

func TestParsePeriod(t *testing.T) {
	for _, period := range AllPeriods() {
		if got, err := ParsePeriod(string(period)); err != nil || got != period {
			t.Fatalf("ParsePeriod(%q) = %q, %v; want %q", period, got, err, period)
		}
	}
	for _, invalid := range []string{"", "all", "this_week", "TODAY"} {
		if _, err := ParsePeriod(invalid); err == nil {
			t.Fatalf("ParsePeriod(%q) unexpectedly accepted", invalid)
		}
	}
}

func TestResolvePeriodWindow(t *testing.T) {
	// 2026-11-16 is a Monday.
	now := time.Date(2026, 11, 16, 15, 30, 0, 0, time.Local)

	tests := []struct {
		period    Period
		wantStart string
		wantEnd   string
		wantKey   string
	}{
		{PeriodToday, "2026-11-16T00:00:00", "2026-11-17T00:00:00", "2026-11-16"},
		{PeriodYesterday, "2026-11-15T00:00:00", "2026-11-16T00:00:00", "2026-11-15"},
		{PeriodCurrentWeek, "2026-11-16T00:00:00", "2026-11-23T00:00:00", "2026-W47"},
		{PeriodPreviousWeek, "2026-11-09T00:00:00", "2026-11-16T00:00:00", "2026-W46"},
		{PeriodCurrentMonth, "2026-11-01T00:00:00", "2026-12-01T00:00:00", "2026-11"},
		{PeriodPreviousMonth, "2026-10-01T00:00:00", "2026-11-01T00:00:00", "2026-10"},
	}

	for _, tt := range tests {
		window, err := ResolvePeriodWindow(tt.period, now)
		if err != nil {
			t.Fatalf("ResolvePeriodWindow(%q): %v", tt.period, err)
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

func TestResolvePeriodWindowWeekStartsOnMonday(t *testing.T) {
	// 2026-11-15 is a Sunday; its week starts Monday 2026-11-09.
	now := time.Date(2026, 11, 15, 23, 59, 0, 0, time.Local)
	window, err := ResolvePeriodWindow(PeriodCurrentWeek, now)
	if err != nil {
		t.Fatalf("ResolvePeriodWindow: %v", err)
	}
	if got := window.Start.Format("2006-01-02"); got != "2026-11-09" {
		t.Fatalf("week start = %s, want 2026-11-09", got)
	}
}

func TestResolvePeriodWindowMonthArithmetic(t *testing.T) {
	// January previous month must roll to December of the prior year.
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local)
	window, err := ResolvePeriodWindow(PeriodPreviousMonth, now)
	if err != nil {
		t.Fatalf("ResolvePeriodWindow: %v", err)
	}
	if got := window.Start.Format("2006-01-02"); got != "2025-12-01" {
		t.Fatalf("previous month start = %s, want 2025-12-01", got)
	}
	if got := window.End.Format("2006-01-02"); got != "2026-01-01" {
		t.Fatalf("previous month end = %s, want 2026-01-01", got)
	}
}

func TestResolvePeriodWindowEndIsExclusive(t *testing.T) {
	now := time.Date(2026, 11, 16, 0, 0, 0, 0, time.Local)
	window, err := ResolvePeriodWindow(PeriodToday, now)
	if err != nil {
		t.Fatalf("ResolvePeriodWindow: %v", err)
	}
	if !window.End.After(window.Start) {
		t.Fatalf("end %v must be after start %v", window.End, window.Start)
	}
	if !now.Equal(window.Start) {
		t.Fatalf("start %v must equal now %v", window.Start, now)
	}
}

func TestResolvePeriodWindowRejectsUnknown(t *testing.T) {
	if _, err := ResolvePeriodWindow(Period("bogus"), time.Now()); err == nil {
		t.Fatal("expected error for unknown period")
	}
}
