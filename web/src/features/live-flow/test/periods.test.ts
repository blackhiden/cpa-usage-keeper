import { describe, expect, it } from 'vitest'
import {
  isLiveFlowPeriod,
  LIVE_FLOW_DEFAULT_PERIOD,
  LIVE_FLOW_PERIODS,
  previousComparablePeriod,
  readStoredLiveFlowPeriod,
  storeLiveFlowPeriod,
} from '../periods'

describe('live flow periods', () => {
  it('exposes the six requested periods in order', () => {
    expect(LIVE_FLOW_PERIODS.map((option) => option.value)).toEqual([
      'today',
      'yesterday',
      'current_week',
      'previous_week',
      'current_month',
      'previous_month',
    ])
  })

  it('validates period values', () => {
    expect(isLiveFlowPeriod('current_week')).toBe(true)
    expect(isLiveFlowPeriod('this_week')).toBe(false)
    expect(isLiveFlowPeriod(null)).toBe(false)
  })

  it('maps periods to their comparison baseline', () => {
    expect(previousComparablePeriod('today')).toBe('yesterday')
    expect(previousComparablePeriod('current_week')).toBe('previous_week')
    expect(previousComparablePeriod('current_month')).toBe('previous_month')
    expect(previousComparablePeriod('yesterday')).toBeNull()
    expect(previousComparablePeriod('previous_week')).toBeNull()
    expect(previousComparablePeriod('previous_month')).toBeNull()
  })

  it('restores a stored period and falls back for invalid values', () => {
    const storage = { getItem: () => 'current_month' }
    expect(readStoredLiveFlowPeriod(storage)).toBe('current_month')
    expect(readStoredLiveFlowPeriod({ getItem: () => 'bogus' })).toBe(LIVE_FLOW_DEFAULT_PERIOD)
    expect(readStoredLiveFlowPeriod(undefined)).toBe(LIVE_FLOW_DEFAULT_PERIOD)
  })

  it('persists the selection without throwing on failure', () => {
    const written: string[] = []
    storeLiveFlowPeriod({ setItem: (_key, value) => written.push(value) }, 'previous_week')
    expect(written).toEqual(['previous_week'])
    expect(() => storeLiveFlowPeriod({ setItem: () => { throw new Error('quota') } }, 'today')).not.toThrow()
  })
})
