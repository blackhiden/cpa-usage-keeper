// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'

vi.mock('../../usage/FlowTopology', () => ({
  FlowTopology: ({ providers = [] }: { providers?: { id: string }[] }) => (
    <div data-live-flow-canvas data-provider-ids={providers.map((provider) => provider.id).join(',')} />
  ),
}))

const fetchLiveFlowModels = vi.fn()
vi.mock('../api', () => ({ fetchLiveFlowModels: (...args: unknown[]) => fetchLiveFlowModels(...args) }))

class TestWebSocket {
  static instances: TestWebSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null
  constructor(readonly url: string) { TestWebSocket.instances.push(this) }
  close() { this.onclose?.() }
}

import { LiveFlowPanel } from '../../usage/LiveFlowPanel'

const responseFor = (period: string, models: { model: string; request_count: number }[]) => ({
  period,
  period_key: '2026-11-16',
  start: '2026-11-16T00:00:00Z',
  end: '2026-11-17T00:00:00Z',
  timezone: 'Asia/Jakarta',
  total_models: models.length,
  models: models.map((model) => ({
    model: model.model,
    request_count: model.request_count,
    success_count: model.request_count,
    failure_count: 0,
    total_tokens: 0,
    last_seen_at: '2026-11-16T09:00:00Z',
    last_latency_ms: 50,
    last_status: 200,
  })),
})

describe('LiveFlowPanel history', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(async () => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    await i18n.changeLanguage('en')
    TestWebSocket.instances = []
    fetchLiveFlowModels.mockReset()
    vi.stubGlobal('WebSocket', TestWebSocket)
    window.localStorage.clear()
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
    vi.unstubAllGlobals()
  })

  it('renders historical models fetched for the selected period', async () => {
    fetchLiveFlowModels.mockResolvedValue(responseFor('today', [{ model: 'gpt-5', request_count: 3 }]))
    await act(async () => root.render(<LiveFlowPanel />))
    await act(async () => { await Promise.resolve() })
    expect(container.querySelector('[data-live-flow-canvas]')?.getAttribute('data-provider-ids')).toBe('gpt-5')
  })

  it('keeps the canvas usable when the history request fails', async () => {
    fetchLiveFlowModels.mockRejectedValue(new Error('boom'))
    await act(async () => root.render(<LiveFlowPanel />))
    await act(async () => { await Promise.resolve() })
    expect(container.querySelector('[data-live-flow-canvas]')).not.toBeNull()
    expect(container.textContent).toContain('Period history is unavailable')
  })

  it('restores a stored period selection on mount', async () => {
    window.localStorage.setItem('cpa-usage-keeper-live-flow-period', 'previous_month')
    fetchLiveFlowModels.mockResolvedValue(responseFor('previous_month', []))
    await act(async () => root.render(<LiveFlowPanel />))
    await act(async () => { await Promise.resolve() })
    expect(container.querySelector('[data-live-flow-history]')?.getAttribute('data-history-period')).toBe('previous_month')
    expect(fetchLiveFlowModels.mock.calls[0][0]).toBe('previous_month')
  })
})
