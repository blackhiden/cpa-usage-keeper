// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

// React Flow needs a measurable viewport; stub it so nodes still render in tests.
vi.mock('@xyflow/react', async () => {
  const actual = await vi.importActual<typeof import('@xyflow/react')>('@xyflow/react')
  return {
    ...actual,
    ReactFlow: ({ nodes }: { nodes: Array<{ id: string; className?: string; zIndex?: number }> }) => (
      <div data-topology-canvas>
        {nodes.map((node) => (
          <div key={node.id} data-node-id={node.id} className={node.className} data-z-index={node.zIndex} />
        ))}
      </div>
    ),
    Background: () => null,
    Controls: () => null,
  }
})

import { FlowTopology, liveFlowNodeZIndex, liveFlowRingRadius, type Provider } from '../FlowTopology'

const provider = (id: string, activeRequests = 0): Provider => ({ id, name: id, activeRequests })

describe('live flow topology layout', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  it('keeps neighbouring blocks apart as the model count grows', () => {
    const blockWidth = 190
    for (const count of [2, 3, 4, 5, 8, 12, 20]) {
      const radius = liveFlowRingRadius(count)
      const chord = 2 * radius * Math.sin(Math.PI / count)
      expect(chord, `count=${count} radius=${radius}`).toBeGreaterThanOrEqual(blockWidth)
    }
  })

  it('never shrinks below the base radius for small counts', () => {
    expect(liveFlowRingRadius(2)).toBeGreaterThanOrEqual(250)
    expect(liveFlowRingRadius(6)).toBeGreaterThanOrEqual(250)
  })

  it('grows the radius once blocks would overlap at the base radius', () => {
    expect(liveFlowRingRadius(20)).toBeGreaterThan(250)
  })

  it('collapses the ring for a single block', () => {
    expect(liveFlowRingRadius(1)).toBe(0)
  })

  it('raises active blocks above idle ones', () => {
    expect(liveFlowNodeZIndex(true)).toBeGreaterThan(liveFlowNodeZIndex(false))
  })

  it('marks an in-flight block as active and raises it', async () => {
    await act(async () => root.render(
      <FlowTopology providers={[provider('idle'), provider('busy', 2)]} />,
    ))

    const busy = container.querySelector('[data-node-id="provider-busy"]')
    const idle = container.querySelector('[data-node-id="provider-idle"]')
    expect(busy?.className).toBe('live-flow-node-active')
    expect(idle?.className || null).toBeNull()
    expect(Number(busy?.getAttribute('data-z-index'))).toBeGreaterThan(Number(idle?.getAttribute('data-z-index')))
  })

  it('raises a block that has a pending live trail', async () => {
    await act(async () => root.render(
      <FlowTopology
        providers={[provider('trailing')]}
        trails={{ t1: { id: 't1', modelID: 'trailing', state: 'pending', status: 0, timestamp: 1 } }}
      />,
    ))

    expect(container.querySelector('[data-node-id="provider-trailing"]')?.className).toBe('live-flow-node-active')
  })

  it('does not raise a block whose trail already completed', async () => {
    await act(async () => root.render(
      <FlowTopology
        providers={[provider('done')]}
        trails={{ t1: { id: 't1', modelID: 'done', state: 'success', status: 200, timestamp: 1 } }}
      />,
    ))

    expect(container.querySelector('[data-node-id="provider-done"]')?.className || null).toBeNull()
  })
})
