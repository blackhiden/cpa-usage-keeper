import { describe, expect, it } from 'vitest'
import { mergeLiveFlowModels, type LiveFlowLiveModel } from '../merge'
import type { LiveFlowHistoricalModel } from '../types'

const historical = (model: string, requestCount: number, overrides: Partial<LiveFlowHistoricalModel> = {}): LiveFlowHistoricalModel => ({
  model,
  request_count: requestCount,
  success_count: requestCount,
  failure_count: 0,
  total_tokens: requestCount * 10,
  last_seen_at: '2026-11-16T10:00:00Z',
  last_latency_ms: 120,
  last_status: 200,
  ...overrides,
})

const live = (id: string, overrides: Partial<LiveFlowLiveModel> = {}): LiveFlowLiveModel => ({
  id,
  requestCount: 0,
  activeRequests: 0,
  failures: 0,
  latestStatus: 0,
  latestLatencyMs: 0,
  lastSeenAt: 0,
  ...overrides,
})

describe('mergeLiveFlowModels', () => {
  it('returns historical models when there is no live traffic', () => {
    const merged = mergeLiveFlowModels([historical('gpt-5', 3)], [])
    expect(merged).toHaveLength(1)
    expect(merged[0]).toMatchObject({ id: 'gpt-5', requestCount: 3, totalTokens: 30, source: 'historical' })
  })

  it('unions historical and live models without duplicates', () => {
    const merged = mergeLiveFlowModels(
      [historical('gpt-5', 2)],
      [live('gpt-5', { activeRequests: 1, latestStatus: 0 }), live('claude', { activeRequests: 1 })],
    )
    expect(merged.map((model) => model.id)).toEqual(['gpt-5', 'claude'])
    expect(new Set(merged.map((model) => model.id)).size).toBe(merged.length)
  })

  it('lets live status override historical status', () => {
    const merged = mergeLiveFlowModels(
      [historical('gpt-5', 5, { last_status: 200 })],
      [live('gpt-5', { latestStatus: 429, activeRequests: 1, latestLatencyMs: 42 })],
    )
    expect(merged[0].latestStatus).toBe(429)
    expect(merged[0].latestLatencyMs).toBe(42)
    // Historical period totals remain the baseline.
    expect(merged[0].requestCount).toBe(5)
  })

  it('orders in-flight models first, then by request count', () => {
    const merged = mergeLiveFlowModels(
      [historical('a-model', 9), historical('b-model', 4)],
      [live('c-model', { activeRequests: 2 })],
    )
    expect(merged.map((model) => model.id)).toEqual(['c-model', 'a-model', 'b-model'])
  })

  it('falls back to the unresolved model name for empty ids', () => {
    const merged = mergeLiveFlowModels([historical('', 1)], [])
    expect(merged[0].id).toBe('unresolved')
    expect(merged[0].name).toBe('Unresolved model')
  })

  it('keeps ordering stable for equal request counts', () => {
    const merged = mergeLiveFlowModels([historical('zeta', 1), historical('alpha', 1)], [])
    expect(merged.map((model) => model.id)).toEqual(['alpha', 'zeta'])
  })
})
