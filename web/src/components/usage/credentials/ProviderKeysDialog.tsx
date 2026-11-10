import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { ApiError, listProviderKeys, type ListProviderKeysOptions, type UsageIdentityPageSort } from '@/lib/api'
import type { UsageIdentity } from '@/lib/types'
import { buildAiProviderCredentialRows, type AiProviderCredentialRow } from './credentialViewModels'
import { ProviderKeyCard } from './ProviderKeyCard'
import { CredentialCardGrid, CredentialsPagination, CredentialRowShell, CredentialTableHeader, formatCredentialNumber, formatCredentialPercent, MetricPill, RequestMetric, successRateTone, TonePercent } from './CredentialSectionShell'
import { ProviderBrandIcon } from '@/components/ProviderBrandIcon'
import styles from './CredentialSections.module.scss'

interface ProviderKeysDialogProps {
  open: boolean
  onClose: () => void
  providerId: string
}

type ViewMode = 'card' | 'list'

const DEFAULT_PAGE_SIZE = 20
const DEFAULT_SORT: UsageIdentityPageSort = 'success_rate'

export function ProviderKeysDialog({ open, onClose, providerId }: ProviderKeysDialogProps) {
  const { t } = useTranslation()
  const [keys, setKeys] = useState<UsageIdentity[]>([])
  const [rows, setRows] = useState<AiProviderCredentialRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [totalPages, setTotalPages] = useState(1)
  const [sort, setSort] = useState<UsageIdentityPageSort>(DEFAULT_SORT)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('card')

  useEffect(() => {
    if (!open) return

    const controller = new AbortController()
    const options: ListProviderKeysOptions = {
      sort,
      order: 'desc',
      page,
      page_size: pageSize,
    }

    setLoading(true)
    setError('')
    void listProviderKeys(providerId, options, controller.signal)
      .then((response) => {
        if (controller.signal.aborted) return
        setKeys(response.keys)
        setRows(buildAiProviderCredentialRows(response.keys))
        setTotal(response.total_count)
        setTotalPages(Math.max(response.total_pages, 1))
      })
      .catch((loadError: unknown) => {
        if (controller.signal.aborted) return
        if (loadError instanceof ApiError) {
          setError(loadError.message)
        } else {
          setError(loadError instanceof Error ? loadError.message : 'Failed to load keys')
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })

    return () => controller.abort()
  }, [open, page, pageSize, providerId, sort])

  const handlePageSizeChange = (nextPageSize: number) => {
    setPageSize(nextPageSize)
    setPage(1)
  }

  const handleSortChange = (nextSort: UsageIdentityPageSort) => {
    setSort(nextSort)
    setPage(1)
  }

  const handleViewModeToggle = () => {
    setViewMode((current) => (current === 'card' ? 'list' : 'card'))
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('usage_stats.provider_keys_dialog_title', { provider: providerId })}
      width={920}
    >
      <div className={styles.providerKeysDialog}>
        <div className={styles.providerKeysDialogHeader}>
          <p className={styles.providerKeysDialogSubtitle}>
            {t('usage_stats.provider_keys_dialog_subtitle', { count: total })}
          </p>
          <Button
            type="button"
            variant="secondary"
            appearance="ghost"
            onClick={handleViewModeToggle}
            aria-label={t('usage_stats.provider_keys_toggle_view')}
          >
            {viewMode === 'card' ? t('usage_stats.view_list') : t('usage_stats.view_cards')}
          </Button>
        </div>

        {error && (
          <div className={styles.errorBox} role="alert">
            <span>{error}</span>
          </div>
        )}

        {loading && (
          <div className={styles.loadingContainer}>
            <span>{t('common.loading')}</span>
          </div>
        )}

        {!loading && !error && rows.length === 0 && (
          <div className={styles.emptyState}>
            <span>{t('usage_stats.provider_keys_empty')}</span>
          </div>
        )}

        {!loading && !error && rows.length > 0 && (
          <>
            {viewMode === 'card' ? (
              <CredentialCardGrid>
                {rows.map((row) => (
                  <ProviderKeyCard key={row.identity.id} keyData={row.identity} />
                ))}
              </CredentialCardGrid>
            ) : (
              <div className={styles.providerKeysTableView}>
                <CredentialTableHeader
                  nameLabel={t('usage_stats.credentials_column_name')}
                  totalRequestsLabel={t('usage_stats.credentials_metric_requests')}
                  successRateLabel={t('usage_stats.credentials_metric_success')}
                  totalTokensLabel={t('usage_stats.credentials_metric_tokens')}
                  cacheReadRateLabel={t('usage_stats.credentials_metric_cache')}
                  sideLabel=""
                />
                {rows.map((row) => (
                  <CredentialRowShell
                    key={row.identity.id}
                    icon={<ProviderBrandIcon providerType={row.identity.type} size={40} />}
                    title={row.displayName}
                    subtitle={row.maskedIdentity}
                    badges={row.priorityLabel ? <span className={styles.credentialPriorityBadge}>{row.priorityLabel}</span> : null}
                    metrics={
                      <>
                        <MetricPill label="" value={<RequestMetric total={row.totalRequests} success={row.successCount} failure={row.failureCount} />} />
                        <MetricPill label="" value={<TonePercent value={row.successRate} tone={successRateTone(row.successRate)} />} />
                        <MetricPill label="" value={formatCredentialNumber(row.totalTokens)} />
                        <MetricPill label="" value={formatCredentialPercent(row.cacheReadRate)} />
                      </>
                    }
                    side={null}
                  />
                ))}
              </div>
            )}

            <CredentialsPagination
              page={page}
              total={total}
              totalPages={totalPages}
              pageSize={pageSize}
              sortValue={sort}
              sortLabel={t('usage_stats.credentials_sort_label')}
              sortOptions={[
                { value: 'priority', label: t('usage_stats.credentials_sort_priority') },
                { value: 'total_requests', label: t('usage_stats.credentials_sort_total_requests') },
                { value: 'success_rate', label: t('usage_stats.credentials_sort_success_rate') },
                { value: 'total_tokens', label: t('usage_stats.credentials_sort_total_tokens') },
                { value: 'last_used_at', label: t('usage_stats.credentials_sort_last_used') },
              ]}
              previousLabel={t('usage_stats.previous_page')}
              nextLabel={t('usage_stats.next_page')}
              rowsPerPageLabel={t('usage_stats.rows_per_page')}
              onPageChange={setPage}
              onPageSizeChange={handlePageSizeChange}
              onSortChange={(nextSort) => handleSortChange(nextSort as UsageIdentityPageSort)}
            />
          </>
        )}
      </div>
    </Modal>
  )
}

