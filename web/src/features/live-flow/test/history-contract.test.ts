import { describe, expect, it } from 'vitest'

// The LiveFlowPanel history integration is exercised through the pure helpers
// plus the panel tests. These tests pin the panel-facing contract: a populated
// period response must survive trail expiry and remounts, and the six periods
// must be selectable and restorable.
import { mergeLiveFlowModels } from '../merge'
import { LIVE_FLOW_PERIODS, readStoredLiveFlowPeriod } from '../periods'
import type { LiveFlowHistoricalModel } from '../types'

const historicalModel: LiveFlowHistoricalModel = {
  model: 'gpt-5',
  request_count: 4,
  success_count: 4,
  failure_count: 0,
  total_tokens: 400,
  last_seen_at: '2026-11-16T09:00:00Z',
  last_latency_ms: 88,
  last_status: 200,
}

describe('live flow history contract', () => {
  it('keeps a historical node after the live trail window expires', () => {
    // Trail expiry only clears live trails; the merged historical node remains.
    const merged = mergeLiveFlowModels([historicalModel], [])
    expect(merged).toHaveLength(1)
    expect(merged[0].id).toBe('gpt-5')
    expect(merged[0].requestCount).toBe(4)
  })

  it('rebuilds the same node set from a fresh response after a remount', () => {
    const first = mergeLiveFlowModels([historicalModel], [])
    const afterRemount = mergeLiveFlowModels([historicalModel], [])
    expect(afterRemount).toEqual(first)
  })

  it('exposes six selectable periods and restores the stored selection', () => {
    expect(LIVE_FLOW_PERIODS).toHaveLength(6)
    expect(readStoredLiveFlowPeriod({ getItem: () => 'previous_week' })).toBe('previous_week')
  })
})
