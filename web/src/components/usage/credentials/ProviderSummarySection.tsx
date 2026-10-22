import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ProviderBrandIcon } from '@/components/ProviderBrandIcon'
import { cpamcEmbedSearch } from '@/embed/cpamcEmbed'
import { Select } from '@/components/ui/Select'
import { ApiError, appPath, listProviders } from '@/lib/api'
import type { ProviderSummary } from '@/lib/types'
import { getProviderDetailPath } from '@/lib/usageNavigation'
import styles from './CredentialSections.module.scss'
import { formatCredentialTimestamp } from './credentialViewModels'
import { CredentialRowShell, CredentialSectionShell, CredentialTableHeader, MetricPill, TonePercent, formatCredentialNumber, successRateTone } from './CredentialSectionShell'

export type ProviderSummarySort = 'total_requests' | 'success_rate' | 'total_tokens' | 'last_used_at'
const PROVIDERS_LOAD_FAILED = 'PROVIDERS_LOAD_FAILED'

interface ProviderSummarySectionProps {
  onAuthRequired?: () => void
  refreshRevision?: number
}

function successRate(provider: ProviderSummary): number | null {
  if (!Number.isFinite(provider.success_rate) || provider.total_requests <= 0) return null
  // The API contract returns a decimal ratio. Accept an already-percent value for compatibility.
  return provider.success_rate <= 1 ? provider.success_rate * 100 : provider.success_rate
}

export function ProviderSummarySection({ onAuthRequired, refreshRevision = 0 }: ProviderSummarySectionProps) {
  const { t } = useTranslation()
  const [providers, setProviders] = useState<ProviderSummary[]>([])
  const [total, setTotal] = useState(0)
  const [sort, setSort] = useState<ProviderSummarySort>('success_rate')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    void listProviders({ sort, order: 'desc' }, controller.signal)
      .then((response) => {
        if (controller.signal.aborted) return
        setProviders(response.providers)
        setTotal(response.total_count)
        setError('')
      })
      .catch((loadError: unknown) => {
        if (controller.signal.aborted) return
        if (loadError instanceof ApiError && loadError.status === 401) {
          onAuthRequired?.()
          return
        }
        setError(loadError instanceof Error ? loadError.message : PROVIDERS_LOAD_FAILED)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })

    return () => controller.abort()
  }, [onAuthRequired, refreshRevision, sort])

  const handleSortChange = (value: string) => {
    const nextSort = value as ProviderSummarySort
    if (nextSort === sort) return
    setLoading(true)
    setError('')
    setSort(nextSort)
  }

  return (
    <CredentialSectionShell
      title={t('usage_stats.providers_title')}
      subtitle={t('usage_stats.providers_subtitle')}
      countLabel={t('usage_stats.credentials_count', { count: total })}
      actions={(
        <Select
          value={sort}
          options={[
            { value: 'success_rate', label: t('usage_stats.credentials_sort_success_rate') },
            { value: 'total_requests', label: t('usage_stats.credentials_sort_total_requests') },
            { value: 'total_tokens', label: t('usage_stats.credentials_sort_total_tokens') },
            { value: 'last_used_at', label: t('usage_stats.credentials_sort_last_used') },
          ]}
          onChange={handleSortChange}
          ariaLabel={t('usage_stats.credentials_sort_label')}
          fullWidth={false}
          dropdownMinWidth={180}
        />
      )}
    >
      {error && <div className={styles.credentialInlineError} role="alert">{error === PROVIDERS_LOAD_FAILED ? t('usage_stats.providers_load_failed') : error}</div>}
      {loading && providers.length === 0 && <div className={styles.credentialEmptyState}>{t('common.loading')}</div>}
      {!loading && providers.length === 0 && !error && <div className={styles.credentialEmptyState}>{t('usage_stats.providers_empty')}</div>}
      {providers.length > 0 && (
        <CredentialTableHeader
          rowClassName={styles.aiProviderCredentialRow}
          nameLabel={t('usage_stats.credentials_column_name')}
          totalRequestsLabel={t('usage_stats.provider_key_count')}
          successRateLabel={t('usage_stats.total_requests')}
          totalTokensLabel={t('usage_stats.success_rate')}
          cacheReadRateLabel={t('usage_stats.total_tokens')}
          sideLabel={t('usage_stats.provider_action')}
        />
      )}
      {providers.map((provider) => (
        <CredentialRowShell
          key={provider.provider}
          icon={<ProviderBrandIcon providerType={provider.provider_type || provider.provider} size={30} />}
          title={provider.provider}
          subtitle={provider.provider_type || undefined}
          badges={null}
          metrics={(
            <>
              <MetricPill value={formatCredentialNumber(provider.key_count)} />
              <MetricPill value={formatCredentialNumber(provider.total_requests)} />
              <MetricPill value={<TonePercent value={successRate(provider)} tone={successRateTone(successRate(provider))} />} />
              <MetricPill value={formatCredentialNumber(provider.total_tokens)} />
            </>
          )}
          side={(
            <div className={styles.providerSummaryAction}>
              {provider.last_used_at && <span>{t('usage_stats.credentials_last_used')}: {formatCredentialTimestamp(provider.last_used_at) ?? provider.last_used_at}</span>}
              <a className={`btn btn-secondary btn-sm ${styles.providerSummaryDetailButton}`} href={appPath(getProviderDetailPath(provider.provider)) + cpamcEmbedSearch()} data-provider-detail-action="true">
                <span>{t('usage_stats.provider_detail_action')}</span>
              </a>
            </div>
          )}
          rowClassName={styles.aiProviderCredentialRow}
        />
      ))}
    </CredentialSectionShell>
  )
}
