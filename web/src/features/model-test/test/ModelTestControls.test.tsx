// @vitest-environment happy-dom

import { act } from 'react'
import { createRoot, type Root } from 'react-dom/client'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '@/i18n'
import { ApiError } from '@/lib/api'

const api = vi.hoisted(() => ({
  testModel: vi.fn(),
  notifyModelTestFailure: vi.fn(),
}))

vi.mock('@/lib/api', async (original) => ({ ...await original<typeof import('@/lib/api')>(), ...api }))
import { ModelTestControls } from '../ModelTestControls'

describe('ModelTestControls notify action', () => {
  let root: Root
  let container: HTMLDivElement

  const findButton = (label: string) => [...container.querySelectorAll<HTMLButtonElement>('button')].find((button) => button.textContent === label)

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true
    void i18n.changeLanguage('en')
    api.testModel.mockReset()
    api.notifyModelTestFailure.mockReset().mockResolvedValue({ sent: true })
    container = document.createElement('div')
    document.body.append(container)
    root = createRoot(container)
  })

  afterEach(async () => {
    await act(async () => root.unmount())
    container.remove()
  })

  const renderAndFail = async () => {
    api.testModel.mockResolvedValue({ success: false, mode: 'streaming', status_code: 502, error_code: 'test_failed', reason: 'upstream timeout' })
    await act(async () => root.render(<ModelTestControls model="gpt-test" />))
    await act(async () => findButton('Test')?.click())
  }

  it('keeps the notify action hidden after a successful test', async () => {
    api.testModel.mockResolvedValue({ success: true, mode: 'streaming' })
    await act(async () => root.render(<ModelTestControls model="gpt-test" />))
    await act(async () => findButton('Test')?.click())
    expect(findButton('Notify admin')).toBeUndefined()
  })

  it('sends only the bounded failure fields', async () => {
    await renderAndFail()
    const notifyButton = findButton('Notify admin')
    expect(notifyButton).toBeDefined()
    await act(async () => notifyButton?.click())
    expect(api.notifyModelTestFailure).toHaveBeenCalledWith({
      model: 'gpt-test',
      mode: 'streaming',
      status_code: 502,
      error_code: 'test_failed',
      reason: 'upstream timeout',
    })
    expect(container.textContent).toContain('Administrator notified.')
    expect(findButton('Notify admin')).toBeUndefined()
  })

  it('reports an unconfigured bot with its own message', async () => {
    await renderAndFail()
    api.notifyModelTestFailure.mockRejectedValue(new ApiError('not configured', 409, 'telegram_not_configured'))
    await act(async () => findButton('Notify admin')?.click())
    expect(container.textContent).toContain('The administrator has not configured Telegram notifications.')
  })

  it('reports rate limiting with its own message', async () => {
    await renderAndFail()
    api.notifyModelTestFailure.mockRejectedValue(new ApiError('rate limited', 429, 'notify_rate_limited'))
    await act(async () => findButton('Notify admin')?.click())
    expect(container.textContent).toContain('A notification was sent recently. Please wait before sending another.')
  })
})
