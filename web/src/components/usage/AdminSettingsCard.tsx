import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'
import { Card } from '@/components/ui/Card'
import { Input } from '@/components/ui/Input'
import { fetchAdminSettings, updateAdminSettings, type ApiError } from '@/lib/api'
import styles from './AdminSettingsCard.module.scss'

export function AdminSettingsCard() {
  const { t } = useTranslation()
  const [configured, setConfigured] = useState(false)
  const [timeout, setTimeout] = useState(30)
  const [key, setKey] = useState('')
  const [clearKey, setClearKey] = useState(false)
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
    }).catch(() => {
      if (!controller.signal.aborted) setError(t('admin_settings.load_failed'))
    }).finally(() => {
      if (!controller.signal.aborted) setLoading(false)
    })
    return () => controller.abort()
  }, [t])

  const save = async () => {
    setSaving(true)
    setError('')
    setNotice('')
    try {
      const next = await updateAdminSettings({
        request_timeout_seconds: timeout,
        ...(key || clearKey ? { api_key: key } : {}),
      })
      setConfigured(next.api_key_configured)
      setTimeout(next.request_timeout_seconds)
      setKey('')
      setClearKey(false)
      setNotice(t('admin_settings.saved'))
    } catch (failure) {
      const status = (failure as ApiError).status
      setError(status === 400 ? t('admin_settings.invalid_settings') : t('admin_settings.save_failed'))
    } finally {
      setSaving(false)
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
            max={120}
            step={1}
            label={t('admin_settings.request_timeout')}
            hint={t('admin_settings.timeout_hint')}
            value={timeout}
            onChange={(event) => setTimeout(Number(event.target.value))}
          />
        </div>
        <Button type="button" variant="primary" appearance="action" loading={saving} disabled={saving || timeout < 1 || timeout > 120 || !Number.isInteger(timeout)} onClick={() => void save()}>
          {t('admin_settings.save')}
        </Button>
      </>}
      {error && <p className={styles.error} role="alert">{error}</p>}
      {notice && <p className={styles.notice} role="status" aria-live="polite">{notice}</p>}
    </Card>
  )
}