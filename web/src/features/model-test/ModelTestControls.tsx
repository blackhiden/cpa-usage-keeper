import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { notifyModelTestFailure, testModel, type ApiError } from '@/lib/api'
import type { ModelTestMode, ModelTestResult } from '@/lib/types'
import { Button } from '@/components/ui/Button'
import { Select, type SelectOption } from '@/components/ui/Select'
import styles from './ModelTestControls.module.scss'

interface ModelTestControlsProps {
  model: string
}

type NotifyState = 'idle' | 'sending' | 'sent' | 'failed'

const MAX_REASON_LENGTH = 240

export function ModelTestControls({ model }: ModelTestControlsProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ModelTestMode>('streaming')
  const [result, setResult] = useState<ModelTestResult | null>(null)
  const [errorCode, setErrorCode] = useState('')
  const [notifyState, setNotifyState] = useState<NotifyState>('idle')
  const [notifyError, setNotifyError] = useState('')
  const [running, setRunning] = useState(false)
  const controller = useRef<AbortController | null>(null)
  const options: SelectOption[] = [
    { value: 'regular', label: t('usage_stats.model_test_mode_regular') },
    { value: 'streaming', label: t('usage_stats.model_test_mode_streaming') },
  ]

  useEffect(() => () => controller.current?.abort(), [])

  const runTest = async () => {
    if (running) return
    controller.current?.abort()
    const next = new AbortController()
    controller.current = next
    setRunning(true)
    setResult(null)
    setErrorCode('')
    setNotifyState('idle')
    setNotifyError('')
    try {
      const response = await testModel({ model, mode }, next.signal)
      if (controller.current !== next) return
      setResult(response)
      setErrorCode(response.error_code ?? '')
    } catch {
      if (!next.signal.aborted && controller.current === next) setErrorCode('request_failed')
    } finally {
      if (controller.current === next) {
        controller.current = null
        setRunning(false)
      }
    }
  }

  const reason = errorCode === 'admin_key_required'
    ? t('usage_stats.model_test_admin_key_required')
    : result?.reason?.slice(0, MAX_REASON_LENGTH) || (errorCode ? t('usage_stats.model_test_failure_generic') : '')

  const failed = Boolean(result && !result.success)
  const notifying = notifyState === 'sending'

  const notifyAdmin = async () => {
    if (!failed || notifying || notifyState === 'sent') return
    setNotifyState('sending')
    setNotifyError('')
    try {
      await notifyModelTestFailure({
        model,
        mode,
        ...(result?.status_code ? { status_code: result.status_code } : {}),
        ...(errorCode ? { error_code: errorCode } : {}),
        ...(result?.reason ? { reason: result.reason.slice(0, MAX_REASON_LENGTH) } : {}),
      })
      setNotifyState('sent')
    } catch (failure) {
      const code = (failure as ApiError).code
      setNotifyState('failed')
      setNotifyError(code === 'telegram_not_configured'
        ? t('usage_stats.model_test_notify_not_configured')
        : code === 'notify_rate_limited'
          ? t('usage_stats.model_test_notify_rate_limited')
          : t('usage_stats.model_test_notify_failed'))
    }
  }

  return (
    <div className={styles.controls}>
      <Select value={mode} options={options} onChange={(value) => setMode(value as ModelTestMode)} ariaLabel={t('usage_stats.model_test_mode_label')} fullWidth={false} />
      <Button type="button" variant="secondary" appearance="action" size="sm" disabled={running} loading={running} onClick={() => void runTest()}>
        {running ? t('usage_stats.model_test_testing') : t('usage_stats.model_test_test')}
      </Button>
      {running && <Button type="button" variant="ghost" appearance="action" size="sm" onClick={() => { controller.current?.abort(); setRunning(false) }}>{t('common.cancel')}</Button>}
      {result && <span className={result.success ? styles.success : styles.failure} role="status" aria-live="polite">
        {result.success ? t('usage_stats.model_test_success') : t('usage_stats.model_test_failed')}
      </span>}
      {reason && <span className={styles.reason} role="status">{reason}</span>}
      {failed && notifyState !== 'sent' && <Button type="button" variant="secondary" appearance="action" size="sm" disabled={notifying} loading={notifying} onClick={() => void notifyAdmin()}>
        {t('usage_stats.model_test_notify_admin')}
      </Button>}
      {notifyState === 'sent' && <span className={styles.success} role="status" aria-live="polite">{t('usage_stats.model_test_notify_sent')}</span>}
      {notifyError && <span className={styles.reason} role="alert">{notifyError}</span>}
    </div>
  )
}