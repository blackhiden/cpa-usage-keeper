import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { fetchAdminSettings, sendTelegramTestMessage, updateAdminSettings, type ApiError } from '@/lib/api'
import type { TelegramAdminSettings, UpdateAdminSettingsRequest, UpdateTelegramAdminSettings } from '@/lib/types'
import styles from './AdminSettingsCard.module.scss'

const MAX_TIMEOUT_SECONDS = 120
const CHAT_ID_PATTERN = /^(-?[0-9]{1,20}|@[A-Za-z0-9_]{5,32})$/

function telegramValues(telegram?: TelegramAdminSettings) {
  return {
    configured: telegram?.bot_token_configured ?? false,
    chatID: telegram?.chat_id ?? '',
    threadID: telegram?.message_thread_id ? String(telegram.message_thread_id) : '',
    enabled: telegram?.enabled ?? true,
    muted: telegram?.muted_models ?? [],
  }
}

function sameModels(left: string[], right: string[]): boolean {
  if (left.length !== right.length) return false
  return left.every((value, index) => value === right[index])
}

export function AdminSettingsCard() {
  const { t } = useTranslation()
  const [configured, setConfigured] = useState(false)
  const [timeout, setTimeout] = useState(30)
  const [key, setKey] = useState('')
  const [clearKey, setClearKey] = useState(false)
  const [telegramConfigured, setTelegramConfigured] = useState(false)
  const [telegramToken, setTelegramToken] = useState('')
  const [clearTelegramToken, setClearTelegramToken] = useState(false)
  const [telegramChatID, setTelegramChatID] = useState('')
  const [telegramThreadID, setTelegramThreadID] = useState('')
  const [savedChatID, setSavedChatID] = useState('')
  const [savedThreadID, setSavedThreadID] = useState('')
  const [telegramEnabled, setTelegramEnabled] = useState(true)
  const [savedTelegramEnabled, setSavedTelegramEnabled] = useState(true)
  const [mutedModels, setMutedModels] = useState<string[]>([])
  const [savedMutedModels, setSavedMutedModels] = useState<string[]>([])
  const [mutedDraft, setMutedDraft] = useState('')
  const [testState, setTestState] = useState<'idle' | 'sending' | 'sent' | 'failed'>('idle')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    void fetchAdminSettings(controller.signal).then((settings) => {
      if (controller.signal.aborted) return
      setConfigured(settings.api_key_configured)
      setTimeout(settings.request_timeout_seconds)
      const telegram = telegramValues(settings.telegram)
      setTelegramConfigured(telegram.configured)
      setTelegramChatID(telegram.chatID)
      setTelegramThreadID(telegram.threadID)
      setSavedChatID(telegram.chatID)
      setSavedThreadID(telegram.threadID)
      setTelegramEnabled(telegram.enabled)
      setSavedTelegramEnabled(telegram.enabled)
      setMutedModels(telegram.muted)
      setSavedMutedModels(telegram.muted)
    }).catch(() => {
      if (!controller.signal.aborted) setError(t('admin_settings.load_failed'))
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false)
    })
    return () => controller.abort()
  }, [t])

  const chatIDValid = telegramChatID.trim() === '' || CHAT_ID_PATTERN.test(telegramChatID.trim())
  const threadIDValid = telegramThreadID.trim() === '' || (/^[0-9]+$/.test(telegramThreadID.trim()) && Number(telegramThreadID) > 0)
  const timeoutValid = Number.isInteger(timeout) && timeout >= 1 && timeout <= MAX_TIMEOUT_SECONDS

  const save = async () => {
    setSaving(true)
    setError('')
    setNotice('')
    const telegram: UpdateTelegramAdminSettings = {}
    if (telegramToken) telegram.bot_token = telegramToken
    else if (clearTelegramToken) telegram.bot_token = ''
    const chatID = telegramChatID.trim()
    if (chatID !== savedChatID) telegram.chat_id = chatID
    const threadID = telegramThreadID.trim() === '' ? 0 : Number(telegramThreadID.trim())
    if (threadID !== (Number(savedThreadID) || 0)) telegram.message_thread_id = threadID
    if (telegramEnabled !== savedTelegramEnabled) telegram.enabled = telegramEnabled
    if (!sameModels(mutedModels, savedMutedModels)) telegram.muted_models = mutedModels
    const payload: UpdateAdminSettingsRequest = {
      request_timeout_seconds: timeout,
      ...(key || clearKey ? { api_key: key } : {}),
      ...(Object.keys(telegram).length > 0 ? { telegram } : {}),
    }
    try {
      const next = await updateAdminSettings(payload)
      setConfigured(next.api_key_configured)
      setTimeout(next.request_timeout_seconds)
      const nextTelegram = telegramValues(next.telegram)
      setTelegramConfigured(nextTelegram.configured)
      setTelegramChatID(nextTelegram.chatID)
      setTelegramThreadID(nextTelegram.threadID)
      setSavedChatID(nextTelegram.chatID)
      setSavedThreadID(nextTelegram.threadID)
      setTelegramEnabled(nextTelegram.enabled)
      setSavedTelegramEnabled(nextTelegram.enabled)
      setMutedModels(nextTelegram.muted)
      setSavedMutedModels(nextTelegram.muted)
      setKey('')
      setClearKey(false)
      setTelegramToken('')
      setClearTelegramToken(false)
      setNotice(t('admin_settings.saved'))
    } catch (failure) {
      const status = (failure as ApiError).status
      setError(status === 400 ? t('admin_settings.invalid_settings') : t('admin_settings.save_failed'))
    } finally {
      setSaving(false)
    }
  }

  const addMutedModel = () => {
    const value = mutedDraft.trim()
    if (!value) return
    if (mutedModels.some((entry) => entry.toLowerCase() === value.toLowerCase())) {
      setMutedDraft('')
      return
    }
    setMutedModels([...mutedModels, value])
    setMutedDraft('')
  }

  const removeMutedModel = (model: string) => {
    setMutedModels(mutedModels.filter((entry) => entry !== model))
  }

  const sendTestMessage = async () => {
    if (testState === 'sending') return
    setTestState('sending')
    try {
      await sendTelegramTestMessage()
      setTestState('sent')
    } catch {
      setTestState('failed')
    }
  }

  return (
    <Card title={t('admin_settings.title')} className={styles.card}>
      <p className={styles.description}>{t('admin_settings.description')}</p>
      {loading ? <p role="status">{t('admin_settings.loading')}</p> : <>
        <div className={styles.fields}>
          <div className={styles.keyField}>
            <Input
              type="password"
              autoComplete="new-password"
              label={t('admin_settings.api_key')}
              placeholder={configured ? t('admin_settings.key_saved_placeholder') : t('admin_settings.key_empty_placeholder')}
              value={key}
              onChange={(event) => { setKey(event.target.value); setClearKey(false) }}
              aria-label={t('admin_settings.api_key')}
            />
            {configured && <div className={styles.keyActions}>
              <span>{t('admin_settings.key_configured')}</span>
              <Button type="button" variant="ghost" appearance="action" size="sm" onClick={() => { setKey(''); setClearKey(true) }}>
                {t('admin_settings.clear_key')}
              </Button>
            </div>}
          </div>
          <Input
            type="number"
            min={1}
            max={MAX_TIMEOUT_SECONDS}
            step={1}
            label={t('admin_settings.request_timeout')}
            hint={t('admin_settings.timeout_hint')}
            value={timeout}
            onChange={(event) => setTimeout(Number(event.target.value))}
          />
        </div>
        <section className={styles.telegram}>
          <h3 className={styles.sectionTitle}>{t('admin_settings.telegram_title')}</h3>
          <p className={styles.sectionDescription}>{t('admin_settings.telegram_description')}</p>
          <div className={styles.fields}>
            <div className={styles.keyField}>
              <Input
                type="password"
                autoComplete="new-password"
                label={t('admin_settings.telegram_bot_token')}
                placeholder={telegramConfigured ? t('admin_settings.telegram_token_saved_placeholder') : t('admin_settings.telegram_token_empty_placeholder')}
                value={telegramToken}
                onChange={(event) => { setTelegramToken(event.target.value); setClearTelegramToken(false) }}
                aria-label={t('admin_settings.telegram_bot_token')}
              />
              {telegramConfigured && <div className={styles.keyActions}>
                <span>{t('admin_settings.telegram_token_configured')}</span>
                <Button type="button" variant="ghost" appearance="action" size="sm" onClick={() => { setTelegramToken(''); setClearTelegramToken(true) }}>
                  {t('admin_settings.telegram_clear_token')}
                </Button>
              </div>}
            </div>
            <Input
              type="text"
              autoComplete="off"
              label={t('admin_settings.telegram_chat_id')}
              hint={t('admin_settings.telegram_chat_id_hint')}
              error={chatIDValid ? undefined : t('admin_settings.telegram_chat_id_invalid')}
              value={telegramChatID}
              onChange={(event) => setTelegramChatID(event.target.value)}
            />
          </div>
          <Input
            type="number"
            min={1}
            step={1}
            autoComplete="off"
            label={t('admin_settings.telegram_topic_id')}
            hint={t('admin_settings.telegram_topic_id_hint')}
            error={threadIDValid ? undefined : t('admin_settings.telegram_topic_id_invalid')}
            value={telegramThreadID}
            onChange={(event) => setTelegramThreadID(event.target.value)}
          />
          <label className={styles.toggle}>
            <input type="checkbox" checked={telegramEnabled} onChange={(event) => setTelegramEnabled(event.target.checked)} />
            <span>{t('admin_settings.telegram_enabled')}</span>
          </label>
          <p className={styles.sectionDescription}>{t('admin_settings.telegram_enabled_hint')}</p>
          <div className={styles.muteRow}>
            <Input
              type="text"
              autoComplete="off"
              label={t('admin_settings.telegram_muted_models')}
              hint={t('admin_settings.telegram_muted_models_hint')}
              value={mutedDraft}
              onChange={(event) => setMutedDraft(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault()
                  addMutedModel()
                }
              }}
            />
            <Button type="button" variant="secondary" appearance="action" size="sm" disabled={!mutedDraft.trim()} onClick={addMutedModel}>
              {t('admin_settings.telegram_add_muted_model')}
            </Button>
          </div>
          {mutedModels.length > 0 && <ul className={styles.mutedList}>
            {mutedModels.map((model) => <li key={model} className={styles.mutedItem}>
              <span>{model}</span>
              <Button type="button" variant="ghost" appearance="action" size="sm" onClick={() => removeMutedModel(model)}>
                {t('admin_settings.telegram_remove_muted_model')}
              </Button>
            </li>)}
          </ul>}
          <div className={styles.testRow}>
            <Button type="button" variant="secondary" appearance="action" size="sm" disabled={!telegramConfigured || testState === 'sending'} loading={testState === 'sending'} onClick={() => void sendTestMessage()}>
              {t('admin_settings.telegram_send_test')}
            </Button>
            {testState === 'sent' && <span className={styles.inlineNotice}>{t('admin_settings.telegram_test_sent')}</span>}
            {testState === 'failed' && <span className={styles.inlineError}>{t('admin_settings.telegram_test_failed')}</span>}
          </div>
        </section>
        <Button type="button" variant="primary" appearance="action" loading={saving} disabled={saving || !timeoutValid || !chatIDValid || !threadIDValid} onClick={() => void save()}>
          {t('admin_settings.save')}
        </Button>
      </>}
      {error && <p className={styles.error} role="alert">{error}</p>}
      {notice && <p className={styles.notice} role="status" aria-live="polite">{notice}</p>}
    </Card>
  )
}