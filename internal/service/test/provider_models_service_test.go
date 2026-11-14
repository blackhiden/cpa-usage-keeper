package test

import (
	"context"
	"testing"
	"time"

	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/repository"
	keeperservice "cpa-usage-keeper/internal/service"
)

func TestListProviderModelsServiceCalculatesSuccessRate(t *testing.T) {
	db := openUsageServiceTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	identity := entities.UsageIdentity{
		Name: "OpenAI Key", AuthType: entities.UsageIdentityAuthTypeAIProvider,
		AuthTypeName: "apikey", Identity: "openai-key", Type: "openai", Provider: "OpenAI",
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	events := []entities.UsageEvent{
		{EventKey: "evt-1", AuthType: "apikey", AuthIndex: "openai-key", Model: "gpt-4", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", AuthType: "apikey", AuthIndex: "openai-key", Model: "gpt-4", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-3", AuthType: "apikey", AuthIndex: "openai-key", Model: "gpt-4", Timestamp: now, Failed: true, InputTokens: 0, TotalTokens: 0},
	}
	if _, _, err := repository.InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	service := keeperservice.NewUsageIdentityService(db)
	result, err := service.ListProviderModels(ctx, keeperservice.ListProviderModelsRequest{
		Provider:    "OpenAI",
		CPAAPIKeyID: 0, // Admin
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 model, got %d", result.TotalCount)
	}
	if len(result.Models) != 1 {
		t.Fatalf("expected 1 model in response, got %d", len(result.Models))
	}

	model := result.Models[0]
	if model.Model != "gpt-4" {
		t.Fatalf("expected gpt-4, got %s", model.Model)
	}
	if model.TotalRequests != 3 || model.SuccessCount != 2 || model.FailureCount != 1 {
		t.Fatalf("unexpected stats: requests=%d success=%d failure=%d", model.TotalRequests, model.SuccessCount, model.FailureCount)
	}

	expectedRate := 2.0 / 3.0
	if model.SuccessRate < expectedRate-0.01 || model.SuccessRate > expectedRate+0.01 {
		t.Fatalf("expected success rate ~0.667, got %.3f", model.SuccessRate)
	}
}

func TestListProviderModelsServiceScopesViewerByAPIKey(t *testing.T) {
	db := openUsageServiceTestDatabase(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)

	// Create CPA API key for viewer
	apiKey := entities.CPAAPIKey{APIKey: "viewer-key", DisplayKey: "viewer-key", Enabled: true}
	if err := db.Create(&apiKey).Error; err != nil {
		t.Fatalf("create API key: %v", err)
	}

	identity := entities.UsageIdentity{
		Name: "Claude Key", AuthType: entities.UsageIdentityAuthTypeAIProvider,
		AuthTypeName: "apikey", Identity: "claude-key", Type: "claude", Provider: "Claude",
	}
	if err := db.Create(&identity).Error; err != nil {
		t.Fatalf("seed identity: %v", err)
	}

	events := []entities.UsageEvent{
		{EventKey: "evt-1", APIGroupKey: "viewer-key", AuthType: "apikey", AuthIndex: "claude-key", Model: "claude-3-opus", Timestamp: now, Failed: false, InputTokens: 100, TotalTokens: 150},
		{EventKey: "evt-2", APIGroupKey: "other-key", AuthType: "apikey", AuthIndex: "claude-key", Model: "claude-3-haiku", Timestamp: now, Failed: false, InputTokens: 50, TotalTokens: 70},
	}
	if _, _, err := repository.InsertUsageEvents(db, events); err != nil {
		t.Fatalf("insert events: %v", err)
	}

	service := keeperservice.NewUsageIdentityService(db)
	result, err := service.ListProviderModels(ctx, keeperservice.ListProviderModelsRequest{
		Provider:    "Claude",
		CPAAPIKeyID: apiKey.ID,
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("ListProviderModels failed: %v", err)
	}

	if result.TotalCount != 1 {
		t.Fatalf("expected viewer to see 1 model, got %d", result.TotalCount)
	}
	if len(result.Models) != 1 {
		t.Fatalf("expected 1 model in response, got %d", len(result.Models))
	}
	if result.Models[0].Model != "claude-3-opus" {
		t.Fatalf("expected viewer to see claude-3-opus, got %s", result.Models[0].Model)
	}
}
