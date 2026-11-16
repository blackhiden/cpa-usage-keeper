import type { LiveFlowPeriod } from './types'

export interface LiveFlowPeriodOption {
  value: LiveFlowPeriod
  labelKey: string
  triggerLabelKey: string
}

// Display order matches the requested six-period dropdown.
export const LIVE_FLOW_PERIODS: ReadonlyArray<LiveFlowPeriodOption> = [
  { value: 'today', labelKey: 'usage_stats.live_flow_period_today', triggerLabelKey: 'usage_stats.live_flow_period_trigger_today' },
  { value: 'yesterday', labelKey: 'usage_stats.live_flow_period_yesterday', triggerLabelKey: 'usage_stats.live_flow_period_trigger_yesterday' },
  { value: 'current_week', labelKey: 'usage_stats.live_flow_period_current_week', triggerLabelKey: 'usage_stats.live_flow_period_trigger_current_week' },
  { value: 'previous_week', labelKey: 'usage_stats.live_flow_period_previous_week', triggerLabelKey: 'usage_stats.live_flow_period_trigger_previous_week' },
  { value: 'current_month', labelKey: 'usage_stats.live_flow_period_current_month', triggerLabelKey: 'usage_stats.live_flow_period_trigger_current_month' },
  { value: 'previous_month', labelKey: 'usage_stats.live_flow_period_previous_month', triggerLabelKey: 'usage_stats.live_flow_period_trigger_previous_month' },
]

export const LIVE_FLOW_PERIOD_STORAGE_KEY = 'cpa-usage-keeper-live-flow-period'
export const LIVE_FLOW_DEFAULT_PERIOD: LiveFlowPeriod = 'today'

const validPeriods = new Set<string>(LIVE_FLOW_PERIODS.map((option) => option.value))

export function isLiveFlowPeriod(value: unknown): value is LiveFlowPeriod {
  return typeof value === 'string' && validPeriods.has(value)
}

// The period immediately preceding the given one, used for trend comparison.
// Today vs Yesterday, Current Week vs Previous Week, Current Month vs Previous
// Month are the only unambiguous pairings among the six fixed periods.
export function previousComparablePeriod(period: LiveFlowPeriod): LiveFlowPeriod | null {
  switch (period) {
    case 'today': return 'yesterday'
    case 'current_week': return 'previous_week'
    case 'current_month': return 'previous_month'
    default: return null
  }
}

export function readStoredLiveFlowPeriod(storage: Pick<Storage, 'getItem'> | undefined): LiveFlowPeriod {
  if (!storage) return LIVE_FLOW_DEFAULT_PERIOD
  try {
    const stored = storage.getItem(LIVE_FLOW_PERIOD_STORAGE_KEY)
    return isLiveFlowPeriod(stored) ? stored : LIVE_FLOW_DEFAULT_PERIOD
  } catch {
    return LIVE_FLOW_DEFAULT_PERIOD
  }
}

export function storeLiveFlowPeriod(storage: Pick<Storage, 'setItem'> | undefined, period: LiveFlowPeriod): void {
  if (!storage) return
  try {
    storage.setItem(LIVE_FLOW_PERIOD_STORAGE_KEY, period)
  } catch {
    // Persisting the selection is best-effort; the view still works without it.
  }
}
