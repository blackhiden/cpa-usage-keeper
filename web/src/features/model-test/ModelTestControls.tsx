import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { testModel } from '@/lib/api'
import type { ModelTestMode, ModelTestResult } from '@/lib/types'
import { Button } from '@/components/ui/Button'
import { Select, type SelectOption } from '@/components/ui/Select'
import styles from './ModelTestControls.module.scss'

interface ModelTestControlsProps {
  model: string
}

const MAX_REASON_LENGTH = 240

export function ModelTestControls({ model }: ModelTestControlsProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<ModelTestMode>('regular')
  const [result, setResult] = useState<ModelTestResult | null>(null)
  const [errorCode, setErrorCode] = useState('')
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
    </div>
  )
}