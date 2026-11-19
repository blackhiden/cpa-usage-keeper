import { apiFetch, apiPath } from '@/lib/api';
import type { ModelRankingMetric, ModelRankingPeriod, ModelRankingResponse } from './types';

export class ModelRankingApiError extends Error {
  status: number;
  code: string;

  constructor(code: string, status: number) {
    super(code);
    this.name = 'ModelRankingApiError';
    this.status = status;
    this.code = code;
  }
}

export const fetchModelRankingLeaderboard = async (
  period: ModelRankingPeriod,
  metric: ModelRankingMetric,
  signal?: AbortSignal,
  provider = '',
  viewerScope = false,
): Promise<ModelRankingResponse> => {
  const query = new URLSearchParams({ period, metric });
  if (provider) query.set('provider', provider);
  const path = viewerScope ? '/key-model-ranking/leaderboards' : '/model-ranking/leaderboards';
  const response = await apiFetch(apiPath(`${path}?${query.toString()}`), {
    credentials: 'include',
    cache: 'no-store',
    signal,
  });
  if (!response.ok) {
    let code = 'model_ranking_request_failed';
    try {
      const payload = await response.json() as { error?: string };
      code = payload.error?.trim() || code;
    } catch {
      // Non-JSON error bodies keep the stable local code.
    }
    throw new ModelRankingApiError(code, response.status);
  }
  return response.json() as Promise<ModelRankingResponse>;
};