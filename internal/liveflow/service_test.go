package liveflow_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"cpa-usage-keeper/internal/config"
	"cpa-usage-keeper/internal/entities"
	"cpa-usage-keeper/internal/liveflow"
	"cpa-usage-keeper/internal/repository"

	"gorm.io/gorm"
)

func openServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, reader, err := repository.OpenDatabasePools(config.Config{SQLitePath: filepath.Join(t.TempDir(), "liveflow-service.db")})
	if err != nil {
		t.Fatalf("open database pools: %v", err)
	}
	readerSQL, err := reader.DB()
	if err != nil {
		t.Fatalf("load reader pool: %v", err)
	}
	t.Cleanup(func() { _ = readerSQL.Close() })
	return db
}

func TestServiceModelsReturnsResponseShape(t *testing.T) {
	db := openServiceTestDB(t)
	now := time.Date(2026, 11, 16, 12, 0, 0, 0, time.Local)
	generate := true
	if err := db.Create(&entities.UsageEvent{EventKey: "s1", Model: "gpt-5", Timestamp: now, Generate: &generate}).Error; err != nil {
		t.Fatalf("create usage event: %v", err)
	}

	service := liveflow.NewService(db)
	response, err := service.Models(context.Background(), liveflow.PeriodToday)
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if response.Period != liveflow.PeriodToday {
		t.Fatalf("period = %q, want today", response.Period)
	}
	if response.PeriodKey == "" {
		t.Fatal("period_key must not be empty")
	}
	if response.TotalModels != len(response.Models) {
		t.Fatalf("total_models = %d, len(models) = %d", response.TotalModels, len(response.Models))
	}
	if response.TotalModels != 1 || response.Models[0].Model != "gpt-5" {
		t.Fatalf("unexpected models: %+v", response.Models)
	}
	if response.Start.IsZero() || response.End.IsZero() || response.Timezone == "" {
		t.Fatalf("incomplete window metadata: %+v", response)
	}
}

func TestServiceModelsEmptyPeriodReturnsEmptyList(t *testing.T) {
	db := openServiceTestDB(t)
	service := liveflow.NewService(db)
	response, err := service.Models(context.Background(), liveflow.PeriodPreviousMonth)
	if err != nil {
		t.Fatalf("Models: %v", err)
	}
	if response.TotalModels != 0 || len(response.Models) != 0 {
		t.Fatalf("expected empty result, got %+v", response)
	}
}

func TestServiceModelsRejectsUnknownPeriod(t *testing.T) {
	db := openServiceTestDB(t)
	service := liveflow.NewService(db)
	if _, err := service.Models(context.Background(), liveflow.Period("bogus")); err == nil {
		t.Fatal("expected error for unknown period")
	}
}
