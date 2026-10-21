package dto

import "time"

// UsageIdentityStatsDelta 是 usage identity 聚合统计的仓储层扫描结果。
type UsageIdentityStatsDelta struct {
	TotalRequests   int64
	SuccessCount    int64
	FailureCount    int64
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	CachedTokens    int64
	CacheReadTokens int64
	TotalTokens     int64
	FirstUsedAt     *time.Time `gorm:"serializer:storageTime"`
	LastUsedAt      *time.Time `gorm:"serializer:storageTime"`
	MaxUsageEventID int64
}

// UsageIdentityTypeCount 是 usage identity 按原始 type 聚合后的计数。
type UsageIdentityTypeCount struct {
	Type  string `gorm:"column:type"`
	Count int64  `gorm:"column:count"`
}

// ProviderAggregate 是按 provider 分组聚合的 AI provider 统计。
type ProviderAggregate struct {
	Provider        string     `gorm:"column:provider"`
	ProviderType    string     `gorm:"column:provider_type"`
	KeyCount        int64      `gorm:"column:key_count"`
	TotalRequests   int64      `gorm:"column:total_requests"`
	SuccessCount    int64      `gorm:"column:success_count"`
	FailureCount    int64      `gorm:"column:failure_count"`
	InputTokens     int64      `gorm:"column:input_tokens"`
	CacheReadTokens int64      `gorm:"column:cache_read_tokens"`
	TotalTokens     int64      `gorm:"column:total_tokens"`
	LastUsedAt      *time.Time `gorm:"column:last_used_at;serializer:storageTime"`
}

// ProviderKeyStats 是 API-key viewer 范围内单条 provider key 的事件聚合。
type ProviderKeyStats struct {
	ID              int64      `gorm:"column:id"`
	AuthIndex       string     `gorm:"column:auth_index"`
	TotalRequests   int64      `gorm:"column:total_requests"`
	SuccessCount    int64      `gorm:"column:success_count"`
	FailureCount    int64      `gorm:"column:failure_count"`
	InputTokens     int64      `gorm:"column:input_tokens"`
	OutputTokens    int64      `gorm:"column:output_tokens"`
	ReasoningTokens int64      `gorm:"column:reasoning_tokens"`
	CachedTokens    int64      `gorm:"column:cached_tokens"`
	CacheReadTokens int64      `gorm:"column:cache_read_tokens"`
	TotalTokens     int64      `gorm:"column:total_tokens"`
	FirstUsedAt     *time.Time `gorm:"column:first_used_at;serializer:storageTime"`
	LastUsedAt      *time.Time `gorm:"column:last_used_at;serializer:storageTime"`
}
