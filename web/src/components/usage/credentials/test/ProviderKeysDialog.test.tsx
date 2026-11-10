import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ProviderKeysDialog } from '../ProviderKeysDialog'
import type { ProviderKeysResponse, UsageIdentity } from '@/lib/types'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'usage_stats.provider_keys_dialog_title') return `Keys for ${params?.provider}`
      if (key === 'usage_stats.provider_keys_dialog_subtitle') return `${params?.count} keys`
      if (key === 'usage_stats.provider_keys_toggle_view') return 'Toggle view'
      if (key === 'usage_stats.view_list') return 'List view'
      if (key === 'usage_stats.view_cards') return 'Card view'
      if (key === 'common.loading') return 'Loading...'
      if (key === 'usage_stats.provider_keys_empty') return 'No keys found'
      if (key === 'usage_stats.credentials_column_name') return 'Name'
      if (key === 'usage_stats.credentials_metric_requests') return 'Requests'
      if (key === 'usage_stats.credentials_metric_success') return 'Success'
      if (key === 'usage_stats.credentials_metric_tokens') return 'Tokens'
      if (key === 'usage_stats.credentials_metric_cache') return 'Cache'
      if (key === 'usage_stats.credentials_sort_label') return 'Sort by'
      if (key === 'usage_stats.credentials_sort_priority') return 'Priority'
      if (key === 'usage_stats.credentials_sort_total_requests') return 'Total Requests'
      if (key === 'usage_stats.credentials_sort_success_rate') return 'Success Rate'
      if (key === 'usage_stats.credentials_sort_total_tokens') return 'Total Tokens'
      if (key === 'usage_stats.credentials_sort_last_used') return 'Last Used'
      if (key === 'usage_stats.previous_page') return 'Previous'
      if (key === 'usage_stats.next_page') return 'Next'
      if (key === 'usage_stats.rows_per_page') return 'Rows per page'
      return key
    },
  }),
}))

vi.mock('@/lib/api', async () => {
  const actual = await vi.importActual('@/lib/api')
  return {
    ...actual,
    listProviderKeys: vi.fn(),
  }
})

const { listProviderKeys } = await import('@/lib/api')

function createMockKey(id: string, name: string): UsageIdentity {
  return {
    id,
    name,
    alias: null,
    displayName: name,
    auth_type: 'api_key',
    auth_type_name: 'API Key',
    identity: `key_${id}`,
    type: 'openai',
    provider: 'openai',
    prefix: 'sk-',
    priority: 0,
    disabled: false,
    total_requests: 100,
    success_count: 95,
    failure_count: 5,
    input_tokens: 1000,
    output_tokens: 500,
    reasoning_tokens: 0,
    cache_read_tokens: 200,
    total_tokens: 1500,
    last_aggregated_usage_event_id: '',
    first_used_at: '2024-01-01T00:00:00Z',
    last_used_at: '2024-01-10T00:00:00Z',
    stats_updated_at: '2024-01-10T00:00:00Z',
    is_deleted: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-10T00:00:00Z',
  }
}

function createMockResponse(keys: UsageIdentity[]): ProviderKeysResponse {
  return {
    provider: 'openai',
    keys,
    total_count: keys.length,
    page: 1,
    page_size: 20,
    total_pages: 1,
  }
}

describe('ProviderKeysDialog', () => {
  let container: HTMLDivElement
  let root: Root

  beforeEach(() => {
    container = document.createElement('div')
    document.body.appendChild(container)
    root = createRoot(container)
    vi.clearAllMocks()
  })

  afterEach(() => {
    act(() => root.unmount())
    document.body.removeChild(container)
  })

  it('renders dialog with title and subtitle', async () => {
    const mockKeys = [createMockKey('1', 'Key 1'), createMockKey('2', 'Key 2')]
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse(mockKeys))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('Keys for openai')
      expect(document.body.textContent).toContain('2 keys')
    })
  })

  it('displays loading state initially', async () => {
    vi.mocked(listProviderKeys).mockImplementation(() => new Promise(() => {}))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    expect(document.body.textContent).toContain('Loading...')
  })

  it('displays error state on fetch failure', async () => {
    vi.mocked(listProviderKeys).mockRejectedValue(new Error('Network error'))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('Network error')
    })
  })

  it('displays empty state when no keys are returned', async () => {
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse([]))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('No keys found')
    })
  })

  it('renders cards in card view by default', async () => {
    const mockKeys = [createMockKey('1', 'Key 1'), createMockKey('2', 'Key 2')]
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse(mockKeys))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      const cards = document.querySelectorAll('[data-credential-card]')
      expect(cards.length).toBe(2)
    })
  })

  it('toggles between card and list view', async () => {
    const mockKeys = [createMockKey('1', 'Key 1')]
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse(mockKeys))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('List view')
    })

    const toggleButton = Array.from(document.querySelectorAll('button')).find(
      (btn) => btn.textContent === 'List view'
    )
    expect(toggleButton).toBeTruthy()

    await act(async () => {
      toggleButton?.click()
    })

    expect(document.body.textContent).toContain('Card view')
  })

  it('calls listProviderKeys with correct parameters', async () => {
    const mockKeys = [createMockKey('1', 'Key 1')]
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse(mockKeys))

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(listProviderKeys).toHaveBeenCalledWith(
        'openai',
        expect.objectContaining({
          sort: 'success_rate',
          order: 'desc',
          page: 1,
          page_size: 20,
        }),
        expect.any(AbortSignal)
      )
    })
  })

  it('handles pagination controls', async () => {
    const mockKeys = Array.from({ length: 25 }, (_, i) => createMockKey(`${i}`, `Key ${i}`))
    vi.mocked(listProviderKeys).mockResolvedValue({
      ...createMockResponse(mockKeys.slice(0, 20)),
      total_count: 25,
      total_pages: 2,
    })

    await act(async () => {
      root.render(<ProviderKeysDialog open providerId="openai" onClose={vi.fn()} />)
    })

    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('1 / 2')
    })
  })

  it('does not fetch when dialog is closed', async () => {
    vi.mocked(listProviderKeys).mockResolvedValue(createMockResponse([]))

    await act(async () => {
      root.render(<ProviderKeysDialog open={false} providerId="openai" onClose={vi.fn()} />)
    })

    expect(listProviderKeys).not.toHaveBeenCalled()
  })
})


