// Feature-local API client for the Live Flow model history domain. Kept inside
// the feature folder (like web/src/features/model-gnn/api.ts) so the whole
// feature is independently mergeable from upstream.
import { apiFetch, apiPath, ApiError } from '@/lib/api'
import type { LiveFlowModelsResponse, LiveFlowPeriod } from './types'

export async function fetchLiveFlowModels(
  period: LiveFlowPeriod,
  signal?: AbortSignal,
): Promise<LiveFlowModelsResponse> {
  const url = new URL(apiPath('/live-flow/models'), window.location.href)
  url.searchParams.set('period', period)
  const response = await apiFetch(url.toString(), { signal, cache: 'no-store' })
  if (!response.ok) {
    let message = `Failed to load live flow models: ${response.status}`
    try {
      const payload = await response.json() as { error?: string }
      if (payload.error) message = payload.error
    } catch {
      // ignore invalid error payloads
    }
    throw new ApiError(message, response.status)
  }
  return response.json()
}
