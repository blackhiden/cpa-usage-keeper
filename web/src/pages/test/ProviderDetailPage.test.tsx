// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ProviderDetailPage } from '../ProviderDetailPage'

const api = vi.hoisted(() => ({ listProviderKeys: vi.fn(), logout: vi.fn() }))
vi.mock('@/lib/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/lib/api')>(),
  ...api,
}))
vi.mock('@/components/dashboard/DashboardHeader', () => ({
  DashboardHeader: () => <header>Header</header>,
}))
vi.mock('@/components/usage/credentials/AiProviderCredentialsSection', () => ({
  AiProviderCredentialsSection: (props: { total: number; sort: string; page: number; pageSize: number }) => (
    <div data-provider-key-table>{props.total}:{props.sort}:{props.page}:{props.pageSize}</div>
  ),
}))
vi.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: () => undefined },
  useTranslation: () => ({ t: (key: string) => key }),
}))

globalThis.IS_REACT_ACT_ENVIRONMENT = true

describe('ProviderDetailPage', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    window.__APP_BASE_PATH__ = '/cpa'
    api.listProviderKeys.mockResolvedValue({ provider: 'openai', keys: [], total_count: 5, page: 1, page_size: 20, total_pages: 1 })
    api.logout.mockResolvedValue(undefined)
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

  it('loads the provider keys with success-rate sorting and renders a safe return link', async () => {
    await act(async () => root.render(<ProviderDetailPage providerId="openai" onAuthRequired={vi.fn()} />))
    await act(async () => undefined)

    expect(api.listProviderKeys).toHaveBeenCalledWith('openai', {
      sort: 'success_rate', order: 'desc', page: 1, page_size: 20,
    }, expect.any(AbortSignal))
    expect(container.querySelector('[data-provider-key-table]')?.textContent).toBe('5:success_rate:1:20')
    expect(container.querySelector<HTMLAnchorElement>('a')?.getAttribute('href')).toBe('/cpa/ai-provider')
  })

  it('preserves the CPAMC embed marker on the return link', async () => {
    window.history.replaceState(null, '', '/cpa/ai-provider/openai?embed=cpamc')
    await act(async () => root.render(<ProviderDetailPage providerId="openai" onAuthRequired={vi.fn()} />))
    await act(async () => undefined)

    expect(container.querySelector<HTMLAnchorElement>('a')?.getAttribute('href')).toBe('/cpa/ai-provider?embed=cpamc')
  })
})
