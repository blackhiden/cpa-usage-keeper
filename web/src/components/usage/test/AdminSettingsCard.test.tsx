// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'

const api = vi.hoisted(() => ({
  fetchAdminSettings: vi.fn(),
  updateAdminSettings: vi.fn(),
}))

vi.mock('@/lib/api', async (original) => ({ ...await original<typeof import('@/lib/api')>(), ...api }))
import { AdminSettingsCard } from '../AdminSettingsCard'

describe('AdminSettingsCard', () => {
  let root: Root
  let container: HTMLDivElement

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    void i18n.changeLanguage('en')
    api.fetchAdminSettings.mockReset().mockResolvedValue({ api_key_configured: true, request_timeout_seconds: 30 })
    api.updateAdminSettings.mockReset().mockResolvedValue({ api_key_configured: true, request_timeout_seconds: 45 })
    container = document.createElement('div')
    document.body.append(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  it('shows configured state without retrieving or displaying the saved key', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    expect(container.textContent).toContain('A test key is configured.')
    expect(container.querySelector<HTMLInputElement>('input[type="password"]')?.value).toBe('')
    expect(container.textContent).not.toContain('secret')
  })

  it('sends only the edited key and timeout', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const keyInput = container.querySelector<HTMLInputElement>('input[type="password"]')!
    await act(async () => {
      const setter = Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set
      setter?.call(keyInput, 'new-test-key')
      keyInput.dispatchEvent(new Event('input', { bubbles: true }))
      keyInput.dispatchEvent(new Event('change', { bubbles: true }))
    })
    const saveButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Save Admin Settings')
    await act(async () => saveButton?.click())
    expect(api.updateAdminSettings).toHaveBeenCalledWith({ request_timeout_seconds: 30, api_key: 'new-test-key' })
  })
})