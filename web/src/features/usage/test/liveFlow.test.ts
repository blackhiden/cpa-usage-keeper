import { describe, expect, it } from 'vitest'
import { appendLiveFlowEvent, applyLiveFlowEvent, deriveLiveFlowModels, LIVE_FLOW_COMPLETED_TRAIL_DURATION_MS, LIVE_FLOW_PENDING_COLOR, parseLiveFlowEvent, parseLiveFlowRelayError, toLiveFlowTrail } from '../liveFlow'
import { liveFlowFitViewOptions } from '../FlowTopology'

describe('live flow event helpers', () => {
  const event = {
    id: 'flow-1',
    ts: 1_768_000_000_000,
    event: 'completed' as const,
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

  it('preserves a prefixed OpenAI-compatible requested model ID', () => {
    const eventWithPrefix = { ...event, model: 'kr/claude-sonnet-4.5' }
    expect(parseLiveFlowEvent(eventWithPrefix)?.model).toBe('kr/claude-sonnet-4.5')
    expect(deriveLiveFlowModels([eventWithPrefix])[0]?.name).toBe('kr/claude-sonnet-4.5')
  })

  it('derives bounded model nodes from root websocket events', () => {
    const withFailure = appendLiveFlowEvent([event], { ...event, id: 'flow-2', ts: event.ts + 1, status: 502, latency_ms: 90 })
    expect(deriveLiveFlowModels(withFailure)).toEqual([{
      id: 'gpt-5',
      name: 'gpt-5',
      requestCount: 2,
      activeRequests: 0,
      failures: 1,
      latestStatus: 502,
      latestLatencyMs: 90,
      lastSeenAt: event.ts + 1,
    }])
  })

  it('renders started events as in-flight work until their completion arrives', () => {
    const started = { ...event, event: 'started' as const, status: 0, latency_ms: 0 }
    const active = applyLiveFlowEvent({}, started)
    expect(active).toEqual({ 'flow-1': 'gpt-5' })
    expect(deriveLiveFlowModels([started], active)).toEqual([{
      id: 'gpt-5',
      name: 'gpt-5',
      requestCount: 0,
      activeRequests: 1,
      failures: 0,
      latestStatus: 0,
      latestLatencyMs: 0,
      lastSeenAt: 0,
    }])

    expect(applyLiveFlowEvent(active, event)).toEqual({})
    expect(toLiveFlowTrail(started)).toEqual({
      id: 'flow-1',
      modelID: 'gpt-5',
      state: 'pending',
      status: 0,
      timestamp: 1_768_000_000_000,
    })
  })

  it('converts the same request trail to a retained success or failure result', () => {
    const started = { ...event, event: 'started' as const, status: 0, latency_ms: 0 }
    expect(toLiveFlowTrail(event)).toMatchObject({ id: 'flow-1', state: 'success', modelID: 'gpt-5' })
    expect(toLiveFlowTrail({ ...event, status: 502 })).toMatchObject({ id: 'flow-1', state: 'failure', modelID: 'gpt-5' })
    expect(LIVE_FLOW_COMPLETED_TRAIL_DURATION_MS).toBe(25_000)
    expect(toLiveFlowTrail(started)).toMatchObject({ id: 'flow-1', state: 'pending' })
    expect(LIVE_FLOW_PENDING_COLOR).toBe('var(--live-flow-pending-color)')
  })

  it('recognizes a relay error without accepting it as a CPA event', () => {
    const relayError = { kind: 'keeper_live_flow_error', code: 'upstream_unreachable', message: 'unreachable', retryable: true }
    expect(parseLiveFlowRelayError(relayError)).toEqual(relayError)
    expect(parseLiveFlowEvent(relayError)).toBeNull()
  })

  it('fits the initial graph approximately two zoom-out steps below the default view', () => {
    expect(liveFlowFitViewOptions.maxZoom).toBe(0.7)
  })
})
