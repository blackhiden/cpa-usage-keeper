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
  useTranslation: () => ({ t: (key: string, values?: Record<string, number>) => (
    key === 'usage_stats.credentials_count' ? String(values?.count ?? 0) : key
  ) }),
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

  it('renders one aggregate provider row, starts sorted by success rate, and links to its encoded detail path', async () => {
    await act(async () => root.render(<ProviderSummarySection />))
    await act(async () => undefined)

    expect(api.listProviders).toHaveBeenCalledWith({ sort: 'success_rate', order: 'desc' }, expect.any(AbortSignal))
    expect(container.textContent).toContain('OpenAI Team')
    expect(container.textContent).toContain('3')
    expect(container.textContent).toContain('95.00%')
    const detail = container.querySelector<HTMLAnchorElement>('[data-provider-detail-action="true"]')
    expect(detail?.getAttribute('href')).toBe('/cpa/ai-provider/OpenAI%20Team')
    expect(detail?.textContent).toContain('usage_stats.provider_detail_action')
  })
})
