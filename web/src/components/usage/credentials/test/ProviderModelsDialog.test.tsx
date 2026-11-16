// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ProviderModelsDialog } from '../ProviderModelsDialog'
import type { ProviderModelsResponse } from '@/lib/types'

const api = vi.hoisted(() => ({ listProviderModels: vi.fn() }))
vi.mock('@/lib/api', async () => {
  const actual = await vi.importActual('@/lib/api')
  return { ...actual, listProviderModels: api.listProviderModels }
})

vi.mock('react-i18next', () => {
  // The real i18next `t` is stable across renders. Returning a fresh function
  // per call makes any component that lists `t` in a useCallback dependency
  // re-fetch on every render, so this mock must be stable too.
  const t = (key: string, params?: Record<string, unknown>) => {
    if (key === 'usage_stats.provider_models_dialog_title' && params?.provider) {
      return `Models for ${params.provider}`
    }
    if (key === 'usage_stats.provider_models_dialog_subtitle' && typeof params?.count === 'number') {
      return `${params.count} models`
    }
    if (key === 'usage_stats.model') return 'Model'
    if (key === 'usage_stats.requests') return 'Requests'
    if (key === 'usage_stats.success_rate') return 'Success Rate'
    if (key === 'usage_stats.tokens') return 'Tokens'
    if (key === 'usage_stats.total_requests') return 'Total Requests'
    if (key === 'usage_stats.total_tokens') return 'Total Tokens'
    if (key === 'usage_stats.last_used') return 'Last used'
    if (key === 'usage_stats.view_list') return 'List'
    if (key === 'usage_stats.view_cards') return 'Cards'
    if (key === 'usage_stats.provider_models_toggle_view') return 'Toggle view mode'
    if (key === 'usage_stats.no_models_found') return 'No models found'
    if (key === 'usage_stats.failed_to_load_models') return 'Failed to load models'
    if (key === 'common.loading') return 'Loading'
    if (key === 'common.previous') return 'Previous'
    if (key === 'common.next') return 'Next'
    return key
  }
  const translation = { t }
  return { useTranslation: () => translation }
})

globalThis.IS_REACT_ACT_ENVIRONMENT = true

describe('ProviderModelsDialog', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    container = document.createElement('div')
    container.id = 'test-container'
    document.body.appendChild(container)
    root = createRoot(container)
    api.listProviderModels.mockReset()
  })

  afterEach(() => {
    act(() => root.unmount())
    document.body.removeChild(container)
  })

  const mockModelsResponse: ProviderModelsResponse = {
    provider: 'openai',
    models: [
      {
        model: 'gpt-4o',
        total_requests: 1500,
        success_count: 1450,
        failure_count: 50,
        success_rate: 0.967,
        input_tokens: 50000,
        total_tokens: 75000,
        last_used_at: '2026-11-10T00:00:00Z',
      },
      {
        model: 'gpt-4o-mini',
        total_requests: 3200,
        success_count: 3100,
        failure_count: 100,
        success_rate: 0.969,
        input_tokens: 120000,
        total_tokens: 150000,
      },
    ],
    total_count: 2,
    page: 1,
    page_size: 20,
    total_pages: 1,
  }

  // The Modal renders through createPortal into document.body, so assertions
  // must read the body rather than the React root container.
  const viewport = () => document.body

  it('renders the dialog with models in card view', async () => {
    api.listProviderModels.mockResolvedValueOnce(mockModelsResponse)

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })
    await act(async () => undefined)

    expect(viewport().textContent).toContain('Models for openai')
    expect(viewport().textContent).toContain('2 models')
    expect(viewport().textContent).toContain('gpt-4o')
    expect(viewport().textContent).toContain('gpt-4o-mini')
  })

  it('toggles between card and list view', async () => {
    api.listProviderModels.mockResolvedValueOnce(mockModelsResponse)

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })
    await act(async () => undefined)

    const toggleButton = viewport().querySelector('button[aria-label="Toggle view mode"]') as HTMLButtonElement
    expect(toggleButton).toBeTruthy()
    expect(toggleButton.textContent).toContain('List')

    await act(async () => {
      toggleButton.click()
    })

    expect(toggleButton.textContent).toContain('Cards')
    expect(viewport().querySelector('table')).toBeTruthy()
  })

  it('displays loading state', async () => {
    api.listProviderModels.mockImplementationOnce(
      () => new Promise((resolve) => setTimeout(() => resolve(mockModelsResponse), 100))
    )

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })

    expect(viewport().textContent).toContain('Loading')
  })

  it('displays error state', async () => {
    // A non-ApiError rejection is rendered with the localized fallback message,
    // so the assertion targets the fallback rather than the raw error text.
    api.listProviderModels.mockRejectedValueOnce(new Error('Network error'))

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })
    await act(async () => undefined)

    expect(viewport().textContent).toContain('Failed to load models')
  })

  it('displays empty state when no models found', async () => {
    const emptyResponse: ProviderModelsResponse = {
      ...mockModelsResponse,
      models: [],
      total_count: 0,
    }
    api.listProviderModels.mockResolvedValueOnce(emptyResponse)

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })
    await act(async () => undefined)

    expect(viewport().textContent).toContain('No models found')
  })

  it('calls API with correct parameters', async () => {
    api.listProviderModels.mockResolvedValueOnce(mockModelsResponse)

    await act(async () => {
      root.render(
        <ProviderModelsDialog open={true} onClose={vi.fn()} providerId="openai" />
      )
    })
    await act(async () => undefined)

    expect(api.listProviderModels).toHaveBeenCalledWith(
      'openai',
      expect.objectContaining({
        sort: 'total_requests',
        order: 'desc',
        page: 1,
        page_size: 20,
      }),
      expect.any(AbortSignal)
    )
  })

  it('does not load models when dialog is closed', async () => {
    await act(async () => {
      root.render(
        <ProviderModelsDialog open={false} onClose={vi.fn()} providerId="openai" />
      )
    })

    expect(api.listProviderModels).not.toHaveBeenCalled()
  })
})
