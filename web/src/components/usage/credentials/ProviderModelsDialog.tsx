import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { ProviderModelCard } from './ProviderModelCard'
import { ApiError, listProviderModels, type ListProviderModelsOptions } from '@/lib/api'
import type { ProviderModel } from '@/lib/types'
import styles from './ProviderModelsDialog.module.scss'

interface ProviderModelsDialogProps {
  open: boolean
  onClose: () => void
  providerId: string
}

type ViewMode = 'card' | 'list'

const DEFAULT_PAGE_SIZE = 20

export function ProviderModelsDialog({ open, onClose, providerId }: ProviderModelsDialogProps) {
  const { t } = useTranslation()
  const [models, setModels] = useState<ProviderModel[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [viewMode, setViewMode] = useState<ViewMode>('card')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(DEFAULT_PAGE_SIZE)
  const [totalPages, setTotalPages] = useState(1)
  const [totalCount, setTotalCount] = useState(0)
  const [sort, setSort] = useState<string>('total_requests')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')

  const loadModels = useCallback(async (signal: AbortSignal) => {
    setLoading(true)
    setError(null)
    
    const options: ListProviderModelsOptions = {
      sort,
      order,
      page,
      page_size: pageSize,
    }

    try {
      const response = await listProviderModels(providerId, options, signal)
      if (signal.aborted) return
      
      setModels(response.models)
      setTotalCount(response.total_count)
      setTotalPages(Math.max(response.total_pages, 1))
    } catch (err) {
      if (signal.aborted) return
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError(t('usage_stats.failed_to_load_models'))
      }
    } finally {
      if (!signal.aborted) {
        setLoading(false)
      }
    }
  }, [providerId, page, pageSize, sort, order, t])

  useEffect(() => {
    if (!open) return

    const controller = new AbortController()
    void loadModels(controller.signal)

    return () => controller.abort()
  }, [open, loadModels])

  const handleViewModeToggle = useCallback(() => {
    setViewMode((current) => (current === 'card' ? 'list' : 'card'))
  }, [])

  const handleSortChange = useCallback((newSort: string) => {
    if (newSort === sort) {
      setOrder((current) => (current === 'asc' ? 'desc' : 'asc'))
    } else {
      setSort(newSort)
      setOrder('desc')
    }
    setPage(1)
  }, [sort])

  const handlePageChange = useCallback((newPage: number) => {
    setPage(newPage)
  }, [])

  const formatNumber = (num: number): string => {
    if (num >= 1_000_000) {
      return `${(num / 1_000_000).toFixed(1)}M`
    }
    if (num >= 1_000) {
      return `${(num / 1_000).toFixed(1)}K`
    }
    return num.toString()
  }

  const renderCardView = () => (
    <div className={styles.cardGrid}>
      {models.map((model) => (
        <ProviderModelCard key={model.model} model={model} />
      ))}
    </div>
  )

  const renderListView = () => (
    <div className={styles.tableContainer}>
      <table className={styles.table}>
        <thead>
          <tr>
            <th onClick={() => handleSortChange('model')} className={styles.sortable}>
              {t('usage_stats.model')}
              {sort === 'model' && <span className={styles.sortIndicator}>{order === 'asc' ? '↑' : '↓'}</span>}
            </th>
            <th onClick={() => handleSortChange('total_requests')} className={styles.sortable}>
              {t('usage_stats.requests')}
              {sort === 'total_requests' && <span className={styles.sortIndicator}>{order === 'asc' ? '↑' : '↓'}</span>}
            </th>
            <th onClick={() => handleSortChange('success_rate')} className={styles.sortable}>
              {t('usage_stats.success_rate')}
              {sort === 'success_rate' && <span className={styles.sortIndicator}>{order === 'asc' ? '↑' : '↓'}</span>}
            </th>
            <th onClick={() => handleSortChange('total_tokens')} className={styles.sortable}>
              {t('usage_stats.tokens')}
              {sort === 'total_tokens' && <span className={styles.sortIndicator}>{order === 'asc' ? '↑' : '↓'}</span>}
            </th>
          </tr>
        </thead>
        <tbody>
          {models.map((model) => (
            <tr key={model.model}>
              <td className={styles.modelName}>{model.model}</td>
              <td>{formatNumber(model.total_requests)}</td>
              <td>
                <span className={model.success_rate >= 0.95 ? styles.successHigh : model.success_rate >= 0.8 ? styles.successMedium : styles.successLow}>
                  {(model.success_rate * 100).toFixed(1)}%
                </span>
              </td>
              <td>{formatNumber(model.total_tokens)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )

  const renderPagination = () => {
    if (totalPages <= 1) return null

    const pages: number[] = []
    const maxVisible = 5
    let start = Math.max(1, page - Math.floor(maxVisible / 2))
    const end = Math.min(totalPages, start + maxVisible - 1)
    
    if (end - start < maxVisible - 1) {
      start = Math.max(1, end - maxVisible + 1)
    }

    for (let i = start; i <= end; i++) {
      pages.push(i)
    }

    return (
      <div className={styles.pagination}>
        <button
          type="button"
          onClick={() => handlePageChange(page - 1)}
          disabled={page <= 1 || loading}
          className={styles.paginationButton}
        >
          {t('common.previous')}
        </button>
        {start > 1 && (
          <>
            <button type="button" onClick={() => handlePageChange(1)} className={styles.paginationButton}>
              1
            </button>
            {start > 2 && <span className={styles.paginationEllipsis}>...</span>}
          </>
        )}
        {pages.map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => handlePageChange(p)}
            disabled={loading}
            className={`${styles.paginationButton} ${p === page ? styles.paginationButtonActive : ''}`.trim()}
          >
            {p}
          </button>
        ))}
        {end < totalPages && (
          <>
            {end < totalPages - 1 && <span className={styles.paginationEllipsis}>...</span>}
            <button type="button" onClick={() => handlePageChange(totalPages)} className={styles.paginationButton}>
              {totalPages}
            </button>
          </>
        )}
        <button
          type="button"
          onClick={() => handlePageChange(page + 1)}
          disabled={page >= totalPages || loading}
          className={styles.paginationButton}
        >
          {t('common.next')}
        </button>
      </div>
    )
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={t('usage_stats.provider_models_title', { provider: providerId })}
      width={820}
      className={styles.dialog}
    >
      <div className={styles.content}>
        <div className={styles.controls}>
          <div className={styles.info}>
            {t('usage_stats.total_models', { count: totalCount })}
          </div>
          <button
            type="button"
            onClick={handleViewModeToggle}
            className={styles.viewToggle}
            aria-label={t('usage_stats.toggle_view')}
          >
            {viewMode === 'card' ? t('usage_stats.list_view') : t('usage_stats.card_view')}
          </button>
        </div>

        {error && (
          <div className={styles.error} role="alert">
            {error}
          </div>
        )}

        {loading && !models.length && (
          <div className={styles.loading}>
            {t('common.loading')}...
          </div>
        )}

        {!loading && !error && models.length === 0 && (
          <div className={styles.empty}>
            {t('usage_stats.no_models_found')}
          </div>
        )}

        {models.length > 0 && (
          <>
            {viewMode === 'card' ? renderCardView() : renderListView()}
            {renderPagination()}
          </>
        )}
      </div>
    </Modal>
  )
}
