// Package modelranking is the self-contained Model Ranking feature domain. It
// resolves the six fixed calendar periods (Today, Yesterday, Current Week,
// Previous Week, Current Month, Previous Month) into half-open local-calendar
// windows, aggregates model usage for a selected period through the repository,
// projects the rows onto the seven quantitative leaderboard metrics, and shapes
// the read-only HTTP payload.
//
// The domain must stay independently mergeable: it does not import
// internal/ranking, has no schema of its own, and never touches
// usage_events_archive.
package modelranking

import (
	"fmt"
	"time"

	"cpa-usage-keeper/internal/timeutil"
)

// Period is one of the six fixed calendar periods offered by the model ranking.
type Period string

const (
	PeriodToday         Period = "today"
	PeriodYesterday     Period = "yesterday"
	PeriodCurrentWeek   Period = "current_week"
	PeriodPreviousWeek  Period = "previous_week"
	PeriodCurrentMonth  Period = "current_month"
	PeriodPreviousMonth Period = "previous_month"
)

// PeriodKind is the calendar grain a period resolves to. It participates in the
// read-cache key so a future period change cannot reuse a stale window.
type PeriodKind string

const (
	PeriodKindDay   PeriodKind = "day"
	PeriodKindWeek  PeriodKind = "week"
	PeriodKindMonth PeriodKind = "month"
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
	return "", fmt.Errorf("unknown model ranking period %q", value)
}

// Window is a half-open [Start, End) local-calendar period boundary.
type Window struct {
	Kind  PeriodKind
	Start time.Time
	End   time.Time
	// Key is a stable period label such as 2026-11-16, 2026-W46 or 2026-11.
	Key string
}

// ResolveWindow returns the half-open local-calendar bounds for period relative
// to now. Weeks are ISO-style Monday→Sunday and months use AddDate so short
// months stay correct. End is always exclusive and the local clock follows the
// project TZ (time.Local), matching Live Flow and the dashboard day boundaries.
func ResolveWindow(period Period, now time.Time) (Window, error) {
	local := timeutil.NormalizeStorageTime(now)
	dayStart := startOfDay(local)
	weekStart := startOfWeek(local)
	monthStart := startOfMonth(local)

	switch period {
	case PeriodToday:
		return Window{Kind: PeriodKindDay, Start: dayStart, End: dayStart.AddDate(0, 0, 1), Key: dayStart.Format("2006-01-02")}, nil
	case PeriodYesterday:
		start := dayStart.AddDate(0, 0, -1)
		return Window{Kind: PeriodKindDay, Start: start, End: dayStart, Key: start.Format("2006-01-02")}, nil
	case PeriodCurrentWeek:
		return Window{Kind: PeriodKindWeek, Start: weekStart, End: weekStart.AddDate(0, 0, 7), Key: isoWeekKey(weekStart)}, nil
	case PeriodPreviousWeek:
		start := weekStart.AddDate(0, 0, -7)
		return Window{Kind: PeriodKindWeek, Start: start, End: weekStart, Key: isoWeekKey(start)}, nil
	case PeriodCurrentMonth:
		return Window{Kind: PeriodKindMonth, Start: monthStart, End: monthStart.AddDate(0, 1, 0), Key: monthStart.Format("2006-01")}, nil
	case PeriodPreviousMonth:
		start := monthStart.AddDate(0, -1, 0)
		return Window{Kind: PeriodKindMonth, Start: start, End: monthStart, Key: start.Format("2006-01")}, nil
	default:
		return Window{}, fmt.Errorf("unsupported model ranking period %q", period)
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