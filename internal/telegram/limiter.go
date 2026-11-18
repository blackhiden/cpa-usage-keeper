package telegram

import (
	"sync"
	"time"
)

const (
	defaultCooldown     = time.Minute
	defaultHourlyLimit  = 10
	defaultMaxReporters = 4096
)

// LimiterOptions configures the per-reporter notification limiter.
type LimiterOptions struct {
	// Cooldown suppresses repeat notifications for the same reporter and model.
	Cooldown time.Duration
	// HourlyLimit caps notifications per reporter in a rolling hour.
	HourlyLimit int
	// MaxReporters bounds the tracked reporter table.
	MaxReporters int
	// Now is injectable for tests.
	Now func() time.Time
}

type reporterState struct {
	windowStart time.Time
	hourCount   int
	lastByModel map[string]time.Time
	lastSeen    time.Time
}

// Limiter enforces a per-reporter-per-model cooldown plus a per-reporter
// hourly cap so a viewer cannot flood the administrator chat.
type Limiter struct {
	mu sync.Mutex

	cooldown     time.Duration
	hourlyLimit  int
	maxReporters int
	now          func() time.Time

	reporters map[string]*reporterState
}

// NewLimiter builds a limiter, applying defaults for zero options.
func NewLimiter(options LimiterOptions) *Limiter {
	if options.Cooldown <= 0 {
		options.Cooldown = defaultCooldown
	}
	if options.HourlyLimit <= 0 {
		options.HourlyLimit = defaultHourlyLimit
	}
	if options.MaxReporters <= 0 {
		options.MaxReporters = defaultMaxReporters
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Limiter{
		cooldown:     options.Cooldown,
		hourlyLimit:  options.HourlyLimit,
		maxReporters: options.MaxReporters,
		now:          options.Now,
		reporters:    make(map[string]*reporterState),
	}
}

// Allow reports whether a notification may be sent and, when it may not, how
// long the caller should wait before retrying.
func (l *Limiter) Allow(reporter, model string) (bool, time.Duration) {
	if l == nil {
		return true, 0
	}
	if reporter == "" {
		reporter = "unknown"
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	state, exists := l.reporters[reporter]
	if !exists {
		l.makeCapacityLocked()
		state = &reporterState{windowStart: now, lastByModel: make(map[string]time.Time)}
		l.reporters[reporter] = state
	}
	if now.Sub(state.windowStart) >= time.Hour {
		state.windowStart = now
		state.hourCount = 0
	}
	if state.hourCount >= l.hourlyLimit {
		state.lastSeen = now
		return false, retryAfter(now, state.windowStart.Add(time.Hour))
	}
	if last, ok := state.lastByModel[model]; ok {
		if elapsed := now.Sub(last); elapsed < l.cooldown {
			state.lastSeen = now
			return false, l.cooldown - elapsed
		}
	}
	l.pruneModelsLocked(state, now)
	state.hourCount++
	state.lastByModel[model] = now
	state.lastSeen = now
	return true, 0
}

func (l *Limiter) pruneModelsLocked(state *reporterState, now time.Time) {
	for model, last := range state.lastByModel {
		if now.Sub(last) >= l.cooldown {
			delete(state.lastByModel, model)
		}
	}
}

func (l *Limiter) makeCapacityLocked() {
	if len(l.reporters) < l.maxReporters {
		return
	}
	var oldestReporter string
	var oldestSeen time.Time
	for reporter, state := range l.reporters {
		if oldestReporter == "" || state.lastSeen.Before(oldestSeen) {
			oldestReporter = reporter
			oldestSeen = state.lastSeen
		}
	}
	delete(l.reporters, oldestReporter)
}

func retryAfter(now, until time.Time) time.Duration {
	wait := until.Sub(now)
	if wait <= 0 {
		return time.Second
	}
	return wait
}
