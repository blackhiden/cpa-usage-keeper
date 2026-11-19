package modelranking

import (
	"sync"
	"time"
)

// modelRankingReadCacheTTL matches the API-key leaderboard read cache. It only
// absorbs bursty polls and rapid period toggling; nothing is persisted and a
// restart simply recomputes.
const modelRankingReadCacheTTL = 30 * time.Second

type modelRankingCacheKey struct {
	kind     PeriodKind
	key      string
	metric   Metric
	scope    string
	provider string
	trend    bool
}

type modelRankingCacheEntry struct {
	value     Response
	expiresAt time.Time
}

// modelRankingReadCache stores successful responses for the fixed
// period-kind/key/metric combinations (six periods × seven metrics).
type modelRankingReadCache struct {
	mu      sync.Mutex
	entries map[modelRankingCacheKey]modelRankingCacheEntry
}

func newModelRankingReadCache() *modelRankingReadCache {
	return &modelRankingReadCache{entries: make(map[modelRankingCacheKey]modelRankingCacheEntry, 42)}
}

func (c *modelRankingReadCache) leaderboard(kind PeriodKind, key string, metric Metric, scope, provider string, trend bool, now time.Time) (Response, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	cacheKey := modelRankingCacheKey{kind: kind, key: key, metric: metric, scope: scope, provider: provider, trend: trend}
	entry, ok := c.entries[cacheKey]
	if !ok || !now.Before(entry.expiresAt) {
		delete(c.entries, cacheKey)
		return Response{}, false
	}
	return entry.value, true
}

func (c *modelRankingReadCache) store(kind PeriodKind, key string, metric Metric, scope, provider string, trend bool, value Response, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[modelRankingCacheKey{kind: kind, key: key, metric: metric, scope: scope, provider: provider, trend: trend}] = modelRankingCacheEntry{
		value: value, expiresAt: now.Add(modelRankingReadCacheTTL),
	}
}
