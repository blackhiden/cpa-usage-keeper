import { useTranslation } from 'react-i18next'
import type { ProviderModel } from '@/lib/types'
import styles from './ProviderModelCard.module.scss'

interface ProviderModelCardProps {
  model: ProviderModel
}

export function ProviderModelCard({ model }: ProviderModelCardProps) {
  const { t } = useTranslation()

  const formatNumber = (num: number): string => {
    if (num >= 1_000_000) {
      return `${(num / 1_000_000).toFixed(1)}M`
    }
    if (num >= 1_000) {
      return `${(num / 1_000).toFixed(1)}K`
    }
    return num.toString()
  }

  const successRatePercent = (model.success_rate * 100).toFixed(1)
  const successRateClass = model.success_rate >= 0.95 ? styles.successHigh : model.success_rate >= 0.8 ? styles.successMedium : styles.successLow

  return (
    <div className={styles.card}>
      <div className={styles.header}>
        <h3 className={styles.modelName}>{model.model}</h3>
      </div>
      <div className={styles.stats}>
        <div className={styles.statItem}>
          <span className={styles.statLabel}>{t('usage_stats.total_requests')}</span>
          <span className={styles.statValue}>{formatNumber(model.total_requests)}</span>
        </div>
        <div className={styles.statItem}>
          <span className={styles.statLabel}>{t('usage_stats.success_rate')}</span>
          <span className={`${styles.statValue} ${successRateClass}`.trim()}>{successRatePercent}%</span>
        </div>
        <div className={styles.statItem}>
          <span className={styles.statLabel}>{t('usage_stats.total_tokens')}</span>
          <span className={styles.statValue}>{formatNumber(model.total_tokens)}</span>
        </div>
      </div>
      {model.last_used_at && (
        <div className={styles.footer}>
          <span className={styles.lastUsed}>
            {t('usage_stats.last_used')}: {new Date(model.last_used_at).toLocaleString()}
          </span>
        </div>
      )}
    </div>
  )
}
