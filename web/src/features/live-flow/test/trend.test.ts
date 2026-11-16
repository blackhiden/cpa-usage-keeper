import { describe, expect, it } from 'vitest'
import { computeLiveFlowTrends } from '../trend'
import type { LiveFlowHistoricalModel } from '../types'

const row = (model: string, requestCount: number): LiveFlowHistoricalModel => ({
  model,
  request_count: requestCount,
  success_count: requestCount,
  failure_count: 0,
  total_tokens: 0,
  last_seen_at: '2026-11-16T10:00:00Z',
  last_latency_ms: 0,
  last_status: 200,
})

describe('computeLiveFlowTrends', () => {
  it('compares totals against the preceding period', () => {
    const summary = computeLiveFlowTrends([row('gpt-5', 30)], [row('gpt-5', 20)])
    expect(summary.totalCurrent).toBe(30)
    expect(summary.totalPrevious).toBe(20)
    expect(summary.deltaPercent).toBeCloseTo(50)
  })

  it('reports a negative delta when traffic drops', () => {
    const summary = computeLiveFlowTrends([row('gpt-5', 10)], [row('gpt-5', 40)])
    expect(summary.deltaPercent).toBeCloseTo(-75)
  })

  it('returns null deltas when the baseline period is empty', () => {
    const summary = computeLiveFlowTrends([row('gpt-5', 10)], [])
    expect(summary.deltaPercent).toBeNull()
    expect(summary.models[0].deltaPercent).toBeNull()
  })

  it('returns null deltas when there is no comparable period', () => {
    const summary = computeLiveFlowTrends([row('gpt-5', 10)], null)
    expect(summary.deltaPercent).toBeNull()
    expect(summary.models.every((model) => model.deltaPercent === null)).toBe(true)
  })

  it('includes models that disappeared from the current period', () => {
    const summary = computeLiveFlowTrends([row('gpt-5', 5)], [row('retired-model', 7)])
    const retired = summary.models.find((model) => model.model === 'retired-model')
    expect(retired).toMatchObject({ current: 0, previous: 7, deltaPercent: -100 })
  })

  it('handles an empty current period', () => {
    const summary = computeLiveFlowTrends([], [row('gpt-5', 5)])
    expect(summary.totalCurrent).toBe(0)
    expect(summary.deltaPercent).toBeCloseTo(-100)
  })
})
