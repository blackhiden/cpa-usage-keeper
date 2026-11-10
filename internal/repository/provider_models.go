package repository

import (
	"context"
	"fmt"
	"strings"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository/dto"

	"gorm.io/gorm"
	"gorm.io/plugin/dbresolver"
)

const (
	ProviderModelSortTotalRequests = "total_requests"
	ProviderModelSortSuccessRate   = "success_rate"
	ProviderModelSortLastUsedAt    = "last_used_at"
)

type ListProviderModelsRequest struct {
	Provider    string
	Sort        string
	Page        int
	PageSize    int
	APIGroupKey string
}

// ListProviderModels returns paginated models for one provider. When an API-group
// scope is supplied, only models used by that group are returned and statistics
// are calculated from that group's usage events.
func ListProviderModels(ctx context.Context, db *gorm.DB, request ListProviderModelsRequest) ([]dto.ProviderModel, int64, error) {
	if db == nil {
		return nil, 0, fmt.Errorf("database is nil")
	}
	provider := strings.TrimSpace(request.Provider)
	if provider == "" {
		return nil, 0, fmt.Errorf("provider is required")
	}
	page, pageSize := normalizedProviderModelsPage(request.Page, request.PageSize)
	if apiGroupKey := strings.TrimSpace(request.APIGroupKey); apiGroupKey != "" {
		return listProviderModelsForAPIGroup(ctx, db, provider, apiGroupKey, request.Sort, page, pageSize)
	}

	// Admin scope: check if provider exists
	query := db.WithContext(ctx).Clauses(dbresolver.Read).Model(&entities.UsageIdentity{}).
		Select("1").
		Where("auth_type = ? AND provider = ? AND is_deleted = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false)

	var exists int
	if err := query.Limit(1).Scan(&exists).Error; err != nil {
		return nil, 0, fmt.Errorf("check provider: %w", err)
	}
	if exists == 0 {
		return nil, 0, nil
	}

	// Count total models
	var total int64
	countQuery := db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select("COUNT(DISTINCT event.model)").
		Joins("JOIN usage_identities AS identity ON event.auth_index = identity.identity AND event.auth_type = ?", "apikey").
		Where("identity.auth_type = ? AND identity.provider = ? AND identity.is_deleted = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false)

	if err := countQuery.Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count provider models: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	// Aggregate models from usage_events
	expressions := providerModelEventAggregateExpressions()
	models := make([]dto.ProviderModel, 0)
	query = db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select(providerModelAggregateSelect(expressions)).
		Joins("JOIN usage_identities AS identity ON event.auth_index = identity.identity AND event.auth_type = ?", "apikey").
		Where("identity.auth_type = ? AND identity.provider = ? AND identity.is_deleted = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false).
		Group("event.model")

	query = applyProviderModelAggregateSort(query, request.Sort, expressions)
	offset := (page - 1) * pageSize
	if err := query.Limit(pageSize).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list provider models: %w", err)
	}
	return models, total, nil
}

func listProviderModelsForAPIGroup(ctx context.Context, db *gorm.DB, provider, apiGroupKey, sort string, page, pageSize int) ([]dto.ProviderModel, int64, error) {
	// Verify the provider has keys accessible by this API group
	query := db.WithContext(ctx).Clauses(dbresolver.Read).Model(&entities.UsageIdentity{}).
		Select("1").
		Where("auth_type = ? AND provider = ? AND is_deleted = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false).
		Where("EXISTS (SELECT 1 FROM usage_events WHERE usage_events.auth_index = usage_identities.identity AND usage_events.auth_type = ? AND usage_events.api_group_key = ?)", "apikey", apiGroupKey)

	var exists int
	if err := query.Limit(1).Scan(&exists).Error; err != nil {
		return nil, 0, fmt.Errorf("check provider access: %w", err)
	}
	if exists == 0 {
		return nil, 0, nil
	}

	// Count total models for this provider and API group
	var total int64
	countQuery := db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select("COUNT(DISTINCT event.model)").
		Joins("JOIN usage_identities AS identity ON event.auth_index = identity.identity AND event.auth_type = ?", "apikey").
		Where("identity.auth_type = ? AND identity.provider = ? AND identity.is_deleted = ? AND event.api_group_key = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false, apiGroupKey)

	if err := countQuery.Scan(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count provider models for API group: %w", err)
	}
	if total == 0 {
		return nil, 0, nil
	}

	// Aggregate models from usage_events for this API group
	expressions := providerModelEventAggregateExpressions()
	models := make([]dto.ProviderModel, 0)
	query = db.WithContext(ctx).Clauses(dbresolver.Read).
		Table("usage_events AS event").
		Select(providerModelAggregateSelect(expressions)).
		Joins("JOIN usage_identities AS identity ON event.auth_index = identity.identity AND event.auth_type = ?", "apikey").
		Where("identity.auth_type = ? AND identity.provider = ? AND identity.is_deleted = ? AND event.api_group_key = ?", entities.UsageIdentityAuthTypeAIProvider, provider, false, apiGroupKey).
		Group("event.model")

	query = applyProviderModelAggregateSort(query, sort, expressions)
	offset := (page - 1) * pageSize
	if err := query.Limit(pageSize).Offset(offset).Find(&models).Error; err != nil {
		return nil, 0, fmt.Errorf("list provider models for API group: %w", err)
	}
	return models, total, nil
}

func normalizedProviderModelsPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return page, pageSize
}

type providerModelAggregateExpressions struct {
	Model           string
	TotalRequests   string
	SuccessCount    string
	FailureCount    string
	InputTokens     string
	CacheReadTokens string
	TotalTokens     string
	LastUsedAt      string
}

func providerModelEventAggregateExpressions() providerModelAggregateExpressions {
	return providerModelAggregateExpressions{
		Model:           "event.model",
		TotalRequests:   "COUNT(event.id)",
		SuccessCount:    "SUM(CASE WHEN event.failed THEN 0 ELSE 1 END)",
		FailureCount:    "SUM(CASE WHEN event.failed THEN 1 ELSE 0 END)",
		InputTokens:     "SUM(event.input_tokens)",
		CacheReadTokens: "SUM(event.cache_read_tokens)",
		TotalTokens:     "SUM(event.total_tokens)",
		LastUsedAt:      "MAX(event.timestamp)",
	}
}

func providerModelAggregateSelect(expressions providerModelAggregateExpressions) string {
	return fmt.Sprintf("%s AS model, %s AS total_requests, %s AS success_count, %s AS failure_count, %s AS input_tokens, %s AS cache_read_tokens, %s AS total_tokens, %s AS last_used_at",
		expressions.Model, expressions.TotalRequests, expressions.SuccessCount, expressions.FailureCount,
		expressions.InputTokens, expressions.CacheReadTokens, expressions.TotalTokens, expressions.LastUsedAt)
}

func applyProviderModelAggregateSort(query *gorm.DB, sort string, expressions providerModelAggregateExpressions) *gorm.DB {
	switch sort {
	case ProviderModelSortSuccessRate:
		return query.Order(fmt.Sprintf("CASE WHEN %s = 0 THEN 0 ELSE CAST(%s AS REAL) / CAST(%s AS REAL) END DESC", expressions.TotalRequests, expressions.SuccessCount, expressions.TotalRequests)).Order(expressions.Model + " ASC")
	case ProviderModelSortLastUsedAt:
		return query.Order(expressions.LastUsedAt + " IS NULL ASC").Order(expressions.LastUsedAt + " DESC").Order(expressions.Model + " ASC")
	default:
		return query.Order(expressions.TotalRequests + " DESC").Order(expressions.Model + " ASC")
	}
}

