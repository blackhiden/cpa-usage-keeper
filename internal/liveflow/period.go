// Package liveflow isolates the Live Flow model-history feature domain: it
// resolves the six fixed calendar periods (Today, Yesterday, Current Week,
// Previous Week, Current Month, Previous Month) into half-open local-calendar
// windows and exposes the aggregated model usage for a selected period.
//
// This package is a self-contained feature domain (see docs/ARCHITECTURE.md);
// it must stay independently mergeable so upstream changes cannot remove or
// replace the feature.
package liveflow

import (
	"fmt"
	"time"

	"cpa-usage-keeper/internal/timeutil"
)

// Period is a fixed calendar period selectable in the Live Flow history view.
type Period string

const (
	PeriodToday         Period = "today"
	PeriodYesterday     Period = "yesterday"
	PeriodCurrentWeek   Period = "current_week"
	PeriodPreviousWeek  Period = "previous_week"
	PeriodCurrentMonth  Period = "current_month"
	PeriodPreviousMonth Period = "previous_month"
)

// AllPeriods returns the six supported periods in display order.
func AllPeriods() []Period {
	return []Period{
		PeriodToday,
		PeriodYesterday,
		PeriodCurrentWeek,
		PeriodPreviousWeek,
		PeriodCurrentMonth,
		PeriodPreviousMonth,
	}
}

// ParsePeriod parses a period value, rejecting unknown values.
func ParsePeriod(value string) (Period, error) {
	period := Period(value)
	for _, candidate := range AllPeriods() {
		if period == candidate {
			return period, nil
		}
	}
	return "", fmt.Errorf("unknown live flow period %q", value)
}

// Window is a half-open [Start, End) local-calendar period boundary.
type Window struct {
	Start time.Time
	End   time.Time
	// Key is a stable period label such as 2026-11-15, 2026-W46 or 2026-11.
	Key string
}

// ResolvePeriodWindow returns the half-open local-calendar bounds for period
// relative to now. Weeks are ISO-style Monday→Sunday. End is always exclusive.
func ResolvePeriodWindow(period Period, now time.Time) (Window, error) {
	local := timeutil.NormalizeStorageTime(now)
	dayStart := startOfDay(local)
	weekStart := startOfWeek(local)
	monthStart := startOfMonth(local)

	switch period {
	case PeriodToday:
		return Window{Start: dayStart, End: dayStart.AddDate(0, 0, 1), Key: dayStart.Format("2006-01-02")}, nil
	case PeriodYesterday:
		start := dayStart.AddDate(0, 0, -1)
		return Window{Start: start, End: dayStart, Key: start.Format("2006-01-02")}, nil
	case PeriodCurrentWeek:
		return Window{Start: weekStart, End: weekStart.AddDate(0, 0, 7), Key: isoWeekKey(weekStart)}, nil
	case PeriodPreviousWeek:
		start := weekStart.AddDate(0, 0, -7)
		return Window{Start: start, End: weekStart, Key: isoWeekKey(start)}, nil
	case PeriodCurrentMonth:
		return Window{Start: monthStart, End: monthStart.AddDate(0, 1, 0), Key: monthStart.Format("2006-01")}, nil
	case PeriodPreviousMonth:
		start := monthStart.AddDate(0, -1, 0)
		return Window{Start: start, End: monthStart, Key: start.Format("2006-01")}, nil
	default:
		return Window{}, fmt.Errorf("unsupported live flow period %q", period)
	}
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// startOfWeek returns the Monday 00:00 of the week containing t (ISO week).
func startOfWeek(t time.Time) time.Time {
	day := startOfDay(t)
	// time.Weekday: Sunday=0 ... Saturday=6. Monday is the ISO start.
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func isoWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}
