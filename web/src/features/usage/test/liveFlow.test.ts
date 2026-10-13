import { describe, expect, it } from 'vitest'
import { appendLiveFlowEvent, deriveLiveFlowModels, parseLiveFlowEvent } from '../liveFlow'

describe('live flow event helpers', () => {
  const event = {
    id: 'flow-1',
    ts: 1_768_000_000_000,
    method: 'POST',
    path: '/v1/responses',
    model: 'gpt-5',
    status: 200,
    latency_ms: 42,
  }

  it('accepts exactly the metadata contract exposed by CPA Live Flow', () => {
    expect(parseLiveFlowEvent(event)).toEqual(event)
    expect(parseLiveFlowEvent({ ...event, latency_ms: '42' })).toBeNull()
  })

  it('derives bounded model nodes from root websocket events', () => {
    const withFailure = appendLiveFlowEvent([event], { ...event, id: 'flow-2', ts: event.ts + 1, status: 502, latency_ms: 90 })
    expect(deriveLiveFlowModels(withFailure)).toEqual([{
      id: 'gpt-5',
      name: 'gpt-5',
      requestCount: 2,
      failures: 1,
      latestStatus: 502,
      latestLatencyMs: 90,
      lastSeenAt: event.ts + 1,
    }])
  })
})
