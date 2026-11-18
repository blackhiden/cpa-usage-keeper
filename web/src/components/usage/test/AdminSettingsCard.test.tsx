// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'

const api = vi.hoisted(() => ({
  fetchAdminSettings: vi.fn(),
  updateAdminSettings: vi.fn(),
  sendTelegramTestMessage: vi.fn(),
}))

vi.mock('@/lib/api', async (original) => ({ ...await original<typeof import('@/lib/api')>(), ...api }))
import { AdminSettingsCard } from '../AdminSettingsCard'

describe('AdminSettingsCard', () => {
  let root: Root
  let container: HTMLDivElement

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    void i18n.changeLanguage('en')
    api.fetchAdminSettings.mockReset().mockResolvedValue({
      api_key_configured: true,
      request_timeout_seconds: 30,
      telegram: { bot_token_configured: true, chat_id: '-1001234567890', message_thread_id: 7, enabled: true, muted_models: [] },
    })
    api.updateAdminSettings.mockReset().mockResolvedValue({
      api_key_configured: true,
      request_timeout_seconds: 45,
      telegram: { bot_token_configured: true, chat_id: '-1001234567890', message_thread_id: 7, enabled: true, muted_models: [] },
    })
    api.sendTelegramTestMessage.mockReset().mockResolvedValue({ sent: true })
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

  it('allows a Request Timeout up to 600 seconds and rejects 601', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const timeoutInput = container.querySelector<HTMLInputElement>('input[type="number"]')!
    expect(timeoutInput.max).toBe('600')
    const saveButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Save Admin Settings')
    const setValue = (value: string) => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(timeoutInput, value)
      timeoutInput.dispatchEvent(new Event('input', { bubbles: true }))
      timeoutInput.dispatchEvent(new Event('change', { bubbles: true }))
    }
    await act(async () => setValue('600'))
    expect(saveButton?.disabled).toBe(false)
    await act(async () => setValue('601'))
    expect(saveButton?.disabled).toBe(true)
  })

  it('shows the configured telegram token state without rendering the token', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    expect(container.textContent).toContain('A bot token is configured.')
    const passwordInputs = container.querySelectorAll<HTMLInputElement>('input[type="password"]')
    expect(passwordInputs).toHaveLength(2)
    expect(passwordInputs[1]?.value).toBe('')
  })

  it('sends only changed telegram fields', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const chatIDInput = container.querySelector<HTMLInputElement>('input[type="text"]')!
    const numberInputs = container.querySelectorAll<HTMLInputElement>('input[type="number"]')
    const topicInput = numberInputs[1]!
    const setValue = (input: HTMLInputElement, value: string) => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(input, value)
      input.dispatchEvent(new Event('input', { bubbles: true }))
      input.dispatchEvent(new Event('change', { bubbles: true }))
    }
    await act(async () => {
      setValue(chatIDInput, '-2009876543210')
      setValue(topicInput, '9')
    })
    const saveButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Save Admin Settings')
    await act(async () => saveButton?.click())
    expect(api.updateAdminSettings).toHaveBeenCalledWith({
      request_timeout_seconds: 30,
      telegram: { chat_id: '-2009876543210', message_thread_id: 9 },
    })
  })

  it('blocks saving while the chat id is invalid', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const chatIDInput = container.querySelector<HTMLInputElement>('input[type="text"]')!
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(chatIDInput, 'not-a-chat')
      chatIDInput.dispatchEvent(new Event('input', { bubbles: true }))
      chatIDInput.dispatchEvent(new Event('change', { bubbles: true }))
    })
    expect(container.textContent).toContain('Enter a numeric chat ID or @username.')
    const saveButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Save Admin Settings')
    expect(saveButton?.disabled).toBe(true)
  })

  it('saves the enable toggle and muted models', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const checkbox = container.querySelector<HTMLInputElement>('input[type="checkbox"]')!
    await act(async () => checkbox.click())
    const muteInput = container.querySelectorAll<HTMLInputElement>('input[type="text"]')[1]!
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, 'value')?.set?.call(muteInput, 'legacy-model')
      muteInput.dispatchEvent(new Event('input', { bubbles: true }))
      muteInput.dispatchEvent(new Event('change', { bubbles: true }))
    })
    const addButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Add')
    await act(async () => addButton?.click())
    const saveButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Save Admin Settings')
    await act(async () => saveButton?.click())
    expect(api.updateAdminSettings).toHaveBeenCalledWith({
      request_timeout_seconds: 30,
      telegram: { enabled: false, muted_models: ['legacy-model'] },
    })
  })

  it('sends a telegram test message', async () => {
    await act(async () => root.render(<AdminSettingsCard />))
    const sendButton = [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === 'Send test message')
    await act(async () => sendButton?.click())
    expect(api.sendTelegramTestMessage).toHaveBeenCalledTimes(1)
    expect(container.textContent).toContain('Test message sent.')
  })
})