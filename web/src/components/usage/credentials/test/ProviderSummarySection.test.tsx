// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ProviderSummarySection } from '../ProviderSummarySection'

const api = vi.hoisted(() => ({ listProviders: vi.fn() }))
vi.mock('@/lib/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/lib/api')>(),
  ...api,
}))
vi.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: () => undefined },
  useTranslation: () => ({ t: (key: string, values?: Record<string, number | string>) => {
    if (key === 'usage_stats.credentials_count') return String(values?.count ?? 0)
    if (key === 'usage_stats.provider_keys_dialog_title') return `Keys for ${values?.provider}`
    if (key === 'usage_stats.provider_models_dialog_title') return `Models for ${values?.provider}`
    return key
  } }),
}))

globalThis.IS_REACT_ACT_ENVIRONMENT = true

describe('ProviderSummarySection', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    window.__APP_BASE_PATH__ = '/cpa'
    api.listProviders.mockResolvedValue({
      total_count: 1,
      providers: [{
        provider: 'OpenAI Team',
        provider_type: 'openai',
        key_count: 3,
        total_requests: 1000,
        success_count: 950,
        failure_count: 50,
        success_rate: 0.95,
        total_tokens: 123456,
        last_used_at: '2026-10-20T10:30:00Z',
      }],
    })
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
    delete window.__APP_BASE_PATH__
    vi.clearAllMocks()
  })

  it('renders one aggregate provider row, starts sorted by success rate, and shows Keys and Models buttons', async () => {
    await act(async () => root.render(<ProviderSummarySection />))
    await act(async () => undefined)

    expect(api.listProviders).toHaveBeenCalledWith({ sort: 'success_rate', order: 'desc' }, expect.any(AbortSignal))
    expect(container.textContent).toContain('OpenAI Team')
    expect(container.textContent).toContain('3')
    expect(container.textContent).toContain('95.00%')
    
    const keysButton = container.querySelector<HTMLButtonElement>('[data-provider-keys-action="true"]')
    const modelsButton = container.querySelector<HTMLButtonElement>('[data-provider-models-action="true"]')
    
    expect(keysButton).toBeTruthy()
    expect(modelsButton).toBeTruthy()
    expect(keysButton?.textContent).toContain('usage_stats.provider_keys_action')
    expect(modelsButton?.textContent).toContain('usage_stats.provider_models_action')
  })

  it('opens keys dialog when Keys button is clicked', async () => {
    await act(async () => root.render(<ProviderSummarySection />))
    await act(async () => undefined)

    const keysButton = container.querySelector<HTMLButtonElement>('[data-provider-keys-action="true"]')
    expect(keysButton).toBeTruthy()

    await act(async () => keysButton?.click())

    expect(container.textContent).toContain('Keys for OpenAI Team')
  })

  it('opens models dialog when Models button is clicked', async () => {
    await act(async () => root.render(<ProviderSummarySection />))
    await act(async () => undefined)

    const modelsButton = container.querySelector<HTMLButtonElement>('[data-provider-models-action="true"]')
    expect(modelsButton).toBeTruthy()

    await act(async () => modelsButton?.click())

    expect(container.textContent).toContain('Models for OpenAI Team')
  })

  it('closes keys dialog when close handler is called', async () => {
    await act(async () => root.render(<ProviderSummarySection />))
    await act(async () => undefined)

    const keysButton = container.querySelector<HTMLButtonElement>('[data-provider-keys-action="true"]')
    await act(async () => keysButton?.click())

    expect(container.textContent).toContain('Keys for OpenAI Team')

    const closeButton = container.querySelector<HTMLButtonElement>('[aria-label="Close"]')
    await act(async () => closeButton?.click())

    expect(container.textContent).not.toContain('Keys for OpenAI Team')
  })
})
