import type { LiveFlowHistoricalModel } from './types'

// Structural subset of the live panel's model shape that the merge needs.
export interface LiveFlowLiveModel {
  id: string
  requestCount: number
  activeRequests: number
  failures: number
  latestStatus: number
  latestLatencyMs: number
  lastSeenAt: number
}

export type LiveFlowMergedSource = 'live' | 'historical'

export interface LiveFlowMergedModel extends LiveFlowLiveModel {
  name: string
  source: LiveFlowMergedSource
  totalTokens: number
}

const UNRESOLVED_MODEL_ID = 'unresolved'
const UNRESOLVED_MODEL_NAME = 'Unresolved model'

function toTimestamp(value: string): number {
  const parsed = Date.parse(value)
  return Number.isFinite(parsed) ? parsed : 0
}

/**
 * Builds a map from a real model name to its display (alias) name. The live
 * stream only carries the routed model name, so this is what lets a live event
 * land on the same block as the aggregated history row.
 */
export function buildLiveFlowAliasMap(historical: readonly LiveFlowHistoricalModel[]): Map<string, string> {
  const aliases = new Map<string, string>()
  for (const row of historical) {
    const display = row.model?.trim()
    if (!display || !row.source_models?.length) continue
    for (const source of row.source_models) {
      const key = source?.trim()
      if (key && key !== display) aliases.set(key, display)
    }
  }
  return aliases
}

/**
 * Unions historical period nodes with live in-flight nodes.
 *
 * Live model ids are first resolved through the alias map so prefixed real
 * model names do not create duplicate blocks. Live status always overrides
 * historical status, historical counts are kept as the period baseline, and
 * live-only models appear immediately. Ordering is stable: in-flight first,
 * then request count descending, then model id.
 */
export function mergeLiveFlowModels(
  historical: readonly LiveFlowHistoricalModel[],
  liveModels: readonly LiveFlowLiveModel[],
): LiveFlowMergedModel[] {
  const merged = new Map<string, LiveFlowMergedModel>()
  const aliasMap = buildLiveFlowAliasMap(historical)

  for (const row of historical) {
    const id = row.model?.trim() || UNRESOLVED_MODEL_ID
    const current = merged.get(id)
    merged.set(id, {
      id,
      name: id === UNRESOLVED_MODEL_ID ? UNRESOLVED_MODEL_NAME : id,
      requestCount: (current?.requestCount ?? 0) + row.request_count,
      activeRequests: current?.activeRequests ?? 0,
      failures: (current?.failures ?? 0) + row.failure_count,
      latestStatus: current?.latestStatus ?? row.last_status,
      latestLatencyMs: current?.latestLatencyMs ?? row.last_latency_ms,
      lastSeenAt: Math.max(current?.lastSeenAt ?? 0, toTimestamp(row.last_seen_at)),
      totalTokens: (current?.totalTokens ?? 0) + row.total_tokens,
      source: 'historical',
    })
  }

  for (const live of liveModels) {
    const rawID = live.id?.trim() || UNRESOLVED_MODEL_ID
    // Prefer the alias so a prefixed real name reuses the historical block.
    const id = aliasMap.get(rawID) ?? rawID
    const current = merged.get(id)
    if (!current) {
      merged.set(id, {
        id,
        name: id === UNRESOLVED_MODEL_ID ? UNRESOLVED_MODEL_NAME : id,
        requestCount: live.requestCount,
        activeRequests: live.activeRequests,
        failures: live.failures,
        latestStatus: live.latestStatus,
        latestLatencyMs: live.latestLatencyMs,
        lastSeenAt: live.lastSeenAt,
        totalTokens: 0,
        source: 'live',
      })
      continue
    }
    // Live status wins; historical period totals remain the baseline.
    merged.set(id, {
      ...current,
      requestCount: Math.max(current.requestCount, live.requestCount),
      activeRequests: live.activeRequests,
      failures: Math.max(current.failures, live.failures),
      latestStatus: live.latestStatus || current.latestStatus,
      latestLatencyMs: live.latestLatencyMs || current.latestLatencyMs,
      lastSeenAt: Math.max(current.lastSeenAt, live.lastSeenAt),
      source: live.activeRequests > 0 ? 'live' : current.source,
    })
  }

  return [...merged.values()].sort((left, right) => {
    const activeDelta = right.activeRequests - left.activeRequests
    if (activeDelta !== 0) return activeDelta
    const countDelta = right.requestCount - left.requestCount
    if (countDelta !== 0) return countDelta
    return left.id.localeCompare(right.id)
  })
}
