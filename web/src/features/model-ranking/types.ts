export const MODEL_RANKING_PERIODS = [
  'today',
  'yesterday',
  'current_week',
  'previous_week',
  'current_month',
  'previous_month',
] as const;
export type ModelRankingPeriod = (typeof MODEL_RANKING_PERIODS)[number];

export const MODEL_RANKING_METRICS = [
  'total_tokens',
  'request_count',
  'cache_read_rate',
  'ttft_average',
  'latency_average',
  'peak_tpm',
  'peak_rpm',
  'overall',
] as const;
export type ModelRankingMetric = (typeof MODEL_RANKING_METRICS)[number];

export interface ModelRankingEntry {
  rank: number;
  model: string;
  providers: string[];
  request_count: number;
  success_count: number;
  failure_count: number;
  input_tokens: number;
  cache_read_tokens: number;
  total_tokens: number;
  value: number;
  rate_numerator?: number;
  rate_denominator?: number;
  cost_usd?: number;
  unavailable_cost_count?: number;
  top_spender?: { display_name: string; masked_key: string };
  provider_contributions?: { provider: string; request_count: number; total_tokens: number }[];
  trend?: { change: number; percent?: number; previous_value: number; available: boolean };
}

export interface ModelRankingResponse {
  period: ModelRankingPeriod;
  period_key: string;
  metric: ModelRankingMetric;
  start: string;
  end: string;
  timezone: string;
  generated_at: string;
  entries: ModelRankingEntry[];
  /**
   * Client-side marker only: set by the data hook when a silent refresh fails
   * and the last successful board stays visible. The API never sends it.
   */
  stale?: boolean;
  score_explanation?: { version: number; texts: Partial<Record<'en'|'zh'|'zh-TW',string>> };
}