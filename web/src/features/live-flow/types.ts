// Feature-local types for the Live Flow model history domain. Mirrors the
// backend response of GET /api/v1/live-flow/models (see internal/liveflow).

export type LiveFlowPeriod =
  | 'today'
  | 'yesterday'
  | 'current_week'
  | 'previous_week'
  | 'current_month'
  | 'previous_month'

export interface LiveFlowHistoricalModel {
  model: string
  request_count: number
  success_count: number
  failure_count: number
  total_tokens: number
  last_seen_at: string
  last_latency_ms: number
  last_status: number
}

export interface LiveFlowModelsResponse {
  period: LiveFlowPeriod
  period_key: string
  start: string
  end: string
  timezone: string
  total_models: number
  models: LiveFlowHistoricalModel[]
}
