package test

import (
	"context"
	"testing"
	"time"

	. "cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/entities"
)

func TestListProviderModelsAdminSeesAllModels(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	// Create provider identities
	identities := []entities.UsageIdentity{
		{Name: "OpenAI Key 1", AuthType: entities.UsageIdentityAuthTypeAIProvider, AuthTypeName: "apikey", Identity: "key-1", Type: "openai", Provider: "OpenAI"},
		{Name: "OpenAI Key 2", AuthType: entities.UsageIdentityAuthTypeAIProvider, AuthTypeName: "apikey", Identity: "key-2", Type: "openai", Provider: "OpenAI"},
	}
	if err := db.Create(&identities).Error; err != nil {
		t.Fatalf("seed identities: %v", err)
	}

	// Create usage events with different models
	events := []entities.UsageEvent{
		{EventKey: "evt-1", AuthType: "apikey", AuthIndex: "key-1", Model: "gpt-4", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", AuthType: "apikey", AuthIndex: "key-1", Model: "gpt-4", Timestamp: now.Add(time.Hour), Failed: false, InputTokens: 200, CacheReadTokens: 50, TotalTokens: 300},
		{EventKey: "evt-3", AuthType: "apikey", AuthIndex: "key-2", Model: "gpt-3.5-turbo", Timestamp: now, Failed: true, InputTokens: 50, TotalTokens: 50},
		{EventKey: "evt-4", AuthType: "apikey", AuthIndex: "key-2", Model: "gpt-4", Timestamp: now.Add(2 * time.Hour), Failed: false, InputTokens: 150, TotalTokens: 200},
	}
	if _, _, err := InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	// Admin request (no API group key)
	models, total, err := ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider: "OpenAI",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels failed: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected 2 models, got %d", total)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 model results, got %d", len(models))
	}

	// Check gpt-4 aggregation
	gpt4 := models[0]
	if gpt4.Model != "gpt-4" {
		t.Fatalf("expected first model to be gpt-4 (sorted by total_requests desc), got %s", gpt4.Model)
	}
	if gpt4.TotalRequests != 3 || gpt4.SuccessCount != 2 || gpt4.FailureCount != 1 {
		t.Fatalf("gpt-4 stats incorrect: requests=%d success=%d failure=%d", gpt4.TotalRequests, gpt4.SuccessCount, gpt4.FailureCount)
	}
	if gpt4.InputTokens != 450 || gpt4.CacheReadTokens != 50 || gpt4.TotalTokens != 650 {
		t.Fatalf("gpt-4 tokens incorrect: input=%d cache_read=%d total=%d", gpt4.InputTokens, gpt4.CacheReadTokens, gpt4.TotalTokens)
	}

	// Check gpt-3.5-turbo
	gpt35 := models[1]
	if gpt35.Model != "gpt-3.5-turbo" {
		t.Fatalf("expected second model to be gpt-3.5-turbo, got %s", gpt35.Model)
	}
	if gpt35.TotalRequests != 1 || gpt35.SuccessCount != 0 || gpt35.FailureCount != 1 {
		t.Fatalf("gpt-3.5-turbo stats incorrect: requests=%d success=%d failure=%d", gpt35.TotalRequests, gpt35.SuccessCount, gpt35.FailureCount)
	}
}

func TestListProviderModelsViewerSeesScopedModels(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	identities := []entities.UsageIdentity{
		{Name: "Claude Key 1", AuthType: entities.UsageIdentityAuthTypeAIProvider, AuthTypeName: "apikey", Identity: "claude-key-1", Type: "claude", Provider: "Claude"},
		{Name: "Claude Key 2", AuthType: entities.UsageIdentityAuthTypeAIProvider, AuthTypeName: "apikey", Identity: "claude-key-2", Type: "claude", Provider: "Claude"},
	}
	if err := db.Create(&identities).Error; err != nil {
		t.Fatalf("seed identities: %v", err)
	}

	events := []entities.UsageEvent{
		{EventKey: "evt-1", APIGroupKey: "viewer-key", AuthType: "apikey", AuthIndex: "claude-key-1", Model: "claude-3-opus", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", APIGroupKey: "viewer-key", AuthType: "apikey", AuthIndex: "claude-key-1", Model: "claude-3-sonnet", Timestamp: now, Failed: false, InputTokens: 80, TotalTokens: 120},
		{EventKey: "evt-3", APIGroupKey: "other-key", AuthType: "apikey", AuthIndex: "claude-key-2", Model: "claude-3-haiku", Timestamp: now, Failed: false, InputTokens: 50, TotalTokens: 70},
	}
	if _, _, err := InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	models, total, err := ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider:    "Claude",
		APIGroupKey: "viewer-key",
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels failed: %v", err)
	}

	if total != 2 {
		t.Fatalf("expected viewer to see 2 models, got %d", total)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 model results, got %d", len(models))
	}

	modelNames := make(map[string]bool)
	for _, model := range models {
		modelNames[model.Model] = true
	}
	if !modelNames["claude-3-opus"] || !modelNames["claude-3-sonnet"] {
		t.Fatalf("expected viewer to see opus and sonnet, got %v", modelNames)
	}
	if modelNames["claude-3-haiku"] {
		t.Fatalf("viewer should not see haiku model from other-key")
	}
}

func TestListProviderModelsPagination(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	identity := entities.UsageIdentity{
		Name: "Provider Key", AuthType: entities.UsageIdentityAuthTypeAIProvider,
		AuthTypeName: "apikey", Identity: "provider-key", Type: "openai", Provider: "OpenAI",
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	events := []entities.UsageEvent{
		{EventKey: "evt-1", AuthType: "apikey", AuthIndex: "provider-key", Model: "model-a", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", AuthType: "apikey", AuthIndex: "provider-key", Model: "model-b", Timestamp: now, Failed: false, InputTokens: 90, TotalTokens: 130},
		{EventKey: "evt-3", AuthType: "apikey", AuthIndex: "provider-key", Model: "model-c", Timestamp: now, Failed: false, InputTokens: 80, TotalTokens: 120},
	}
	if _, _, err := InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	models, total, err := ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider: "OpenAI",
		Page:     1,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("ListProviderModels page 1 failed: %v", err)
	}

	if total != 3 {
		t.Fatalf("expected total 3 models, got %d", total)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models on page 1, got %d", len(models))
	}

	models, total, err = ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider: "OpenAI",
		Page:     2,
		PageSize: 2,
	})
	if err != nil {
		t.Fatalf("ListProviderModels page 2 failed: %v", err)
	}

	if total != 3 {
		t.Fatalf("expected total 3 models on page 2, got %d", total)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model on page 2, got %d", len(models))
	}
}

func TestListProviderModelsReturnsEmptyForNonexistentProvider(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()

	models, total, err := ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider: "NonExistent",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels failed: %v", err)
	}

	if total != 0 || len(models) != 0 {
		t.Fatalf("expected empty result for nonexistent provider, got total=%d models=%d", total, len(models))
	}
}

func TestListProviderModelsSortBySuccessRate(t *testing.T) {
	db := openTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	identity := entities.UsageIdentity{
		Name: "Provider Key", AuthType: entities.UsageIdentityAuthTypeAIProvider,
		AuthTypeName: "apikey", Identity: "provider-key", Type: "openai", Provider: "OpenAI",
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	events := []entities.UsageEvent{
		{EventKey: "evt-1", AuthType: "apikey", AuthIndex: "provider-key", Model: "reliable-model", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", AuthType: "apikey", AuthIndex: "provider-key", Model: "reliable-model", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-3", AuthType: "apikey", AuthIndex: "provider-key", Model: "unreliable-model", Timestamp: now, Failed: true, InputTokens: 50, TotalTokens: 50},
		{EventKey: "evt-4", AuthType: "apikey", AuthIndex: "provider-key", Model: "unreliable-model", Timestamp: now, Failed: false, InputTokens: 50, TotalTokens: 50},
	}
	if _, _, err := InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	models, _, err := ListProviderModels(ctx, db, ListProviderModelsRequest{
		Provider: "OpenAI",
		Sort:     "success_rate",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels with success_rate sort failed: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Model != "reliable-model" {
		t.Fatalf("expected reliable-model first when sorted by success_rate, got %s", models[0].Model)
	}
	if models[1].Model != "unreliable-model" {
		t.Fatalf("expected unreliable-model second, got %s", models[1].Model)
	}
}


