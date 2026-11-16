package dto

import "time"

// LiveFlowModelWindow is the half-open [Start, End) local-calendar window used
// to aggregate Live Flow model history. It is defined here (not in the
// liveflow domain) so the repository never imports the feature domain.
type LiveFlowModelWindow struct {
	Start time.Time
	End   time.Time
}

// LiveFlowModel is one aggregated model row for a Live Flow history period.
// Model holds the display name: the model alias when present, otherwise the
// real model name. SourceModels lists the real model names grouped under it so
// live stream events can be mapped onto the same block.
type LiveFlowModel struct {
	Model         string
	SourceModels  []string
	RequestCount  int64
	SuccessCount  int64
	FailureCount  int64
	TotalTokens   int64
	LastSeenAt    time.Time
	LastLatencyMS int64
	LastStatus    int
}
