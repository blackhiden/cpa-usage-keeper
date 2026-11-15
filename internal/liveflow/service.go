package liveflow

import (
	"context"
	"fmt"
	"time"

	"cpa-usage-keeper/internal/repository"
	"cpa-usage-keeper/internal/repository/dto"

	"gorm.io/gorm"
)

// Model is one aggregated model row exposed by the Live Flow history API.
type Model struct {
	Model         string    `json:"model"`
	RequestCount  int64     `json:"request_count"`
	SuccessCount  int64     `json:"success_count"`
	FailureCount  int64     `json:"failure_count"`
	TotalTokens   int64     `json:"total_tokens"`
	LastSeenAt    time.Time `json:"last_seen_at"`
	LastLatencyMS int64     `json:"last_latency_ms"`
	LastStatus    int       `json:"last_status"`
}

// ModelsResponse is the Live Flow history payload for one selected period.
// It carries metadata only: model id, counts, status, latency, and last-seen.
type ModelsResponse struct {
	Period      Period    `json:"period"`
	PeriodKey   string    `json:"period_key"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Timezone    string    `json:"timezone"`
	TotalModels int       `json:"total_models"`
	Models      []Model   `json:"models"`
}

// Service resolves the Live Flow model history for a selected period.
type Service struct {
	db  *gorm.DB
	now func() time.Time
}

// NewService builds a Live Flow history service over the given database.
func NewService(db *gorm.DB) *Service {
	return &Service{db: db, now: time.Now}
}

// Models aggregates the models used during period. It never scans the archive
// table and returns an empty list (not an error) for an empty period.
func (s *Service) Models(ctx context.Context, period Period) (ModelsResponse, error) {
	if s == nil || s.db == nil {
		return ModelsResponse{}, fmt.Errorf("live flow service is not configured")
	}
	window, err := ResolvePeriodWindow(period, s.now())
	if err != nil {
		return ModelsResponse{}, err
	}

	rows, err := repository.ListLiveFlowModels(ctx, s.db, dto.LiveFlowModelWindow{Start: window.Start, End: window.End})
	if err != nil {
		return ModelsResponse{}, err
	}

	models := make([]Model, 0, len(rows))
	for _, row := range rows {
		models = append(models, Model{
			Model:         row.Model,
			RequestCount:  row.RequestCount,
			SuccessCount:  row.SuccessCount,
			FailureCount:  row.FailureCount,
			TotalTokens:   row.TotalTokens,
			LastSeenAt:    row.LastSeenAt,
			LastLatencyMS: row.LastLatencyMS,
			LastStatus:    row.LastStatus,
		})
	}

	return ModelsResponse{
		Period:      period,
		PeriodKey:   window.Key,
		Start:       window.Start,
		End:         window.End,
		Timezone:    time.Local.String(),
		TotalModels: len(models),
		Models:      models,
	}, nil
}
