import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { AiProviderCredentialsSection } from '@/components/usage/credentials/AiProviderCredentialsSection'
import { buildAiProviderCredentialRows, type AiProviderCredentialRow } from '@/components/usage/credentials/credentialViewModels'
import { cpamcEmbedSearch } from '@/embed/cpamcEmbed'
import { DashboardHeader } from '@/components/dashboard/DashboardHeader'
import { ApiError, appPath, listProviderKeys, logout, type ListProviderKeysOptions, type UsageIdentityPageSort } from '@/lib/api'
import styles from './UsagePage.module.scss'

interface ProviderDetailPageProps {
  providerId: string
  onAuthRequired: () => void
}

const DEFAULT_PAGE_SIZE = 20
const DEFAULT_SORT: UsageIdentityPageSort = 'success_rate'
const PROVIDER_DETAIL_LOAD_FAILED = 'PROVIDER_DETAIL_LOAD_FAILED'

export function ProviderDetailPage({ providerId, onAuthRequired }: ProviderDetailPageProps) {
  const { t } = useTranslation()
  const [rows, setRows] = useState<AiProviderCredentialRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE)
  const [totalPages, setTotalPages] = useState(1)
  const [sort, setSort] = useState<UsageIdentityPageSort>(DEFAULT_SORT)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [reloadRevision, setReloadRevision] = useState(0)
  const [loggingOut, setLoggingOut] = useState(false)

  useEffect(() => {
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
        setRows(buildAiProviderCredentialRows(response.keys))
        setTotal(response.total_count)
        setTotalPages(Math.max(response.total_pages, 1))
      })
      .catch((loadError: unknown) => {
        if (controller.signal.aborted) return
        if (loadError instanceof ApiError && loadError.status === 401) {
          onAuthRequired()
          return
        }
        setError(loadError instanceof Error ? loadError.message : PROVIDER_DETAIL_LOAD_FAILED)
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false)
      })

    return () => controller.abort()
  }, [onAuthRequired, page, pageSize, providerId, reloadRevision, sort])

  const handlePageSizeChange = useCallback((nextPageSize: number) => {
    setPageSize(nextPageSize)
    setPage(1)
  }, [])

  const handleSortChange = useCallback((nextSort: UsageIdentityPageSort) => {
    setSort(nextSort)
    setPage(1)
  }, [])

  const handleLogout = useCallback(async () => {
    setLoggingOut(true)
    try {
      await logout()
    } finally {
      onAuthRequired()
      setLoggingOut(false)
    }
  }, [onAuthRequired])

  return (
    <div className={`${styles.pageShell} ${styles.standalone}`.trim()} data-keeper-page="provider-detail">
      <div className={styles.pageFrame}>
        <DashboardHeader onLogout={() => void handleLogout()} loggingOut={loggingOut} />
        <main className={styles.contentColumn}>
          <div className={styles.container}>
            {error && (
              <div className={styles.errorBox} role="alert">
                <span>{error === PROVIDER_DETAIL_LOAD_FAILED ? t('usage_stats.provider_detail_load_failed') : error}</span>
                <button type="button" onClick={() => setReloadRevision((current) => current + 1)}>{t('common.retry')}</button>
              </div>
            )}
            <div>
              <a className="btn btn-ghost btn-sm" href={appPath('/ai-provider') + cpamcEmbedSearch()}>
                <span>← {t('usage_stats.provider_detail_back')}</span>
              </a>
            </div>
            <AiProviderCredentialsSection
              rows={rows}
              total={total}
              page={page}
              totalPages={totalPages}
              pageSize={pageSize}
              activeOnly={false}
              sort={sort}
              loading={loading}
              title={t('usage_stats.provider_detail_title', { provider: providerId })}
              subtitle={t('usage_stats.provider_detail_subtitle', { count: total })}
              allowManagement={false}
              onPageChange={setPage}
              onPageSizeChange={handlePageSizeChange}
              onActiveOnlyChange={() => undefined}
              onSortChange={handleSortChange}
            />
          </div>
        </main>
      </div>
    </div>
  )
}
