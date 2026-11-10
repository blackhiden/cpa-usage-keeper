import { useTranslation } from 'react-i18next'
import type { UsageIdentity } from '@/lib/types'
import { ProviderBrandIcon } from '@/components/ProviderBrandIcon'
import { CredentialCardShell, MetricPill, RequestMetric, TonePercent, successRateTone, formatCredentialNumber, cacheReadRateTone } from './CredentialSectionShell'
import { buildAiProviderCredentialRows } from './credentialViewModels'

interface ProviderKeyCardProps {
  keyData: UsageIdentity
}

export function ProviderKeyCard({ keyData }: ProviderKeyCardProps) {
  const { t } = useTranslation()
  
  // Transform the key data using the existing view model builder
  const [row] = buildAiProviderCredentialRows([keyData])
  
  if (!row) {
    return null
  }

  return (
    <CredentialCardShell
      identityId={row.identity.id}
      icon={<ProviderBrandIcon providerType={row.identity.type} size={40} />}
      title={row.displayName}
      badges={row.priorityLabel ? <span>{row.priorityLabel}</span> : undefined}
      metrics={
        <>
          <MetricPill label={t('usage_stats.credentials_metric_requests')} value={<RequestMetric total={row.totalRequests} success={row.successCount} failure={row.failureCount} />} />
          <MetricPill label={t('usage_stats.credentials_metric_success')} value={<TonePercent value={row.successRate} tone={successRateTone(row.successRate)} />} />
          <MetricPill label={t('usage_stats.credentials_metric_tokens')} value={formatCredentialNumber(row.totalTokens)} />
          <MetricPill label={t('usage_stats.credentials_metric_cache')} value={<TonePercent value={row.cacheReadRate} tone={cacheReadRateTone(row.cacheReadRate)} />} />
        </>
      }
      health={
        row.lastUsedText ? (
          <div style={{ fontSize: '0.875rem', color: 'var(--text-secondary)' }}>
            {t('usage_stats.credentials_last_used')}: {row.lastUsedText}
          </div>
        ) : null
      }
    />
  )
}
