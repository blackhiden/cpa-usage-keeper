import type { LiveFlowHistoricalModel } from './types'

export interface LiveFlowModelTrend {
  model: string
  current: number
  previous: number
  /** Relative change in request count, or null when there is no baseline. */
  deltaPercent: number | null
}

export interface LiveFlowTrendSummary {
  totalCurrent: number
  totalPrevious: number
  deltaPercent: number | null
  models: LiveFlowModelTrend[]
}

function requestCounts(models: readonly LiveFlowHistoricalModel[]): Map<string, number> {
  const counts = new Map<string, number>()
  for (const model of models) {
    const id = model.model?.trim() || 'unresolved'
    counts.set(id, (counts.get(id) ?? 0) + model.request_count)
  }
  return counts
}

function percentDelta(current: number, previous: number): number | null {
  // No baseline: the comparison period had no traffic at all.
  if (previous <= 0) return null
  return ((current - previous) / previous) * 100
}

/**
 * Compares the selected period against the immediately preceding period of the
 * same kind. The oldest periods have no baseline and report null deltas instead
 * of inventing a comparison.
 */
export function computeLiveFlowTrends(
  current: readonly LiveFlowHistoricalModel[],
  previous: readonly LiveFlowHistoricalModel[] | null,
): LiveFlowTrendSummary {
  const currentCounts = requestCounts(current)
  const previousCounts = previous ? requestCounts(previous) : new Map<string, number>()

  let totalCurrent = 0
  for (const count of currentCounts.values()) totalCurrent += count
  let totalPrevious = 0
  for (const count of previousCounts.values()) totalPrevious += count

  const modelIDs = new Set<string>([...currentCounts.keys(), ...previousCounts.keys()])
  const models: LiveFlowModelTrend[] = [...modelIDs].map((model) => {
    const currentCount = currentCounts.get(model) ?? 0
    const previousCount = previousCounts.get(model) ?? 0
    return {
      model,
      current: currentCount,
      previous: previousCount,
      deltaPercent: previous ? percentDelta(currentCount, previousCount) : null,
    }
  })
  models.sort((left, right) => right.current - left.current || left.model.localeCompare(right.model))

  return {
    totalCurrent,
    totalPrevious,
    deltaPercent: previous ? percentDelta(totalCurrent, totalPrevious) : null,
    models,
  }
}
