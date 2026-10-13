// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'

vi.mock('../FlowTopology', () => ({
  FlowTopology: () => <div data-live-flow-canvas />,
}))

import { LiveFlowPanel } from '../LiveFlowPanel'

class TestWebSocket {
  static instances: TestWebSocket[] = []
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null

  constructor(readonly url: string) {
    TestWebSocket.instances.push(this)
  }

  close() {
    this.onclose?.()
  }
}

describe('LiveFlowPanel', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(async () => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    await i18n.changeLanguage('en')
    TestWebSocket.instances = []
    vi.stubGlobal('WebSocket', TestWebSocket)
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
    vi.unstubAllGlobals()
  })

  it('mounts the card and canvas before the first event arrives', async () => {
    await act(async () => root.render(<LiveFlowPanel />))
    expect(container.querySelector('[data-live-flow-panel]')).not.toBeNull()
    expect(container.querySelector('[data-live-flow-canvas]')).not.toBeNull()
    expect(container.textContent).toContain('Waiting for the next CPA request')
  })

  it('renders a model when the root websocket supplies a live event', async () => {
    await act(async () => root.render(<LiveFlowPanel />))
    const socket = TestWebSocket.instances[0]
    await act(async () => socket.onopen?.())
    await act(async () => socket.onmessage?.(new MessageEvent('message', {
      data: JSON.stringify({ id: 'flow-1', ts: 1_768_000_000_000, method: 'POST', path: '/v1/responses', model: 'gpt-5', status: 200, latency_ms: 42 }),
    })))
    expect(container.textContent).toContain('gpt-5')
    expect(container.textContent).toContain('POST · 200')
  })
})
