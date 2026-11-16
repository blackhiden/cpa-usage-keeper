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
type LiveFlowModel struct {
	Model         string
	RequestCount  int64
	SuccessCount  int64
	FailureCount  int64
	TotalTokens   int64
	LastSeenAt    time.Time
	LastLatencyMS int64
	LastStatus    int
}
