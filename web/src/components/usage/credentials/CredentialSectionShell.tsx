import type { CSSProperties, ReactNode } from 'react'
import styles from './CredentialSections.module.scss'
import { formatCompactNumber } from '@/utils/usage'
import { Select } from '@/components/ui/Select'
import { CREDENTIAL_PAGE_SIZE_OPTIONS } from './credentialViewModels'
import type { CredentialViewMode } from './credentialListPreferences'

// Fork compatibility: provider dialogs still import these from the shared shell.
export { CredentialCardGrid, CredentialCardShell, MetricPill } from './cards/CredentialCardShell'
export { RequestMetric, TonePercent, successRateTone, cacheReadRateTone } from './credentialMetrics'

type CredentialSectionStyle = CSSProperties

interface CredentialSectionShellProps {
  title: string
  /** Fork compatibility: provider detail routes still supply a subtitle. */
  subtitle?: ReactNode
  /** Optional for fork callers that render the protected table layout directly. */
  viewMode?: CredentialViewMode
  contentClassName?: string
  titleHelp?: ReactNode
  titleExtra?: ReactNode
  actions?: ReactNode
  style?: CredentialSectionStyle
  children: ReactNode
}

export function CredentialSectionShell({ title, subtitle, viewMode = 'list', contentClassName = '', titleHelp, titleExtra, actions, style, children }: CredentialSectionShellProps) {
  const heading = <h3 className={`${styles.credentialSectionTitle} keeper-card-title`}>{title}</h3>
  return (
    <section className={`${styles.credentialSectionCard} keeper-card-surface`} data-view-mode={viewMode} style={style}>
      <div className={styles.credentialSectionHeader}>
        <div className={styles.credentialSectionTitleBlock}>
          <div className={`${styles.credentialSectionTitleRow} keeper-card-title-track`}>
            {titleHelp ? <div className={styles.credentialSectionHeading}>{heading}{titleHelp}</div> : heading}
            {titleExtra}
          </div>
          {subtitle && <p className="keeper-card-subtitle">{subtitle}</p>}
        </div>
        {actions && <div className={styles.credentialSectionActions}>{actions}</div>}
      </div>
      <div className={`${styles.credentialRows} ${contentClassName}`.trim()} data-view-mode={viewMode}>{children}</div>
    </section>
  )
}

export function CredentialPriorityBadge({ children }: { children: ReactNode }) {
  return <span className={styles.credentialPriorityBadge}>{children}</span>
}

const CREDENTIAL_PAGE_SIZE_SELECT_OPTIONS = CREDENTIAL_PAGE_SIZE_OPTIONS.map((option) => ({
  value: String(option),
  label: String(option),
}))

export function CredentialsPagination({
  className,
  controlsClassName,
  page,
  total,
  totalPages,
  pageSize,
  sortValue,
  sortOptions,
  sortLabel,
  previousLabel,
  nextLabel,
  rowsPerPageLabel,
  onPageChange,
  onPageSizeChange,
  onSortChange,
}: {
  className?: string
  controlsClassName?: string
  page: number
  total?: number
  totalPages: number
  pageSize: number
  sortValue?: string
  sortOptions?: Array<{ value: string; label: string }>
  sortLabel?: string
  previousLabel: string
  nextLabel: string
  rowsPerPageLabel: string
  onPageChange: (page: number) => void
  onPageSizeChange: (pageSize: number) => void
  onSortChange?: (sort: string) => void
}) {
  if (total === 0) {
    return null
  }

  const selectedSortLabel = sortOptions?.find((option) => option.value === sortValue)?.label

  return (
    <div className={`${styles.credentialPagination} ${className ?? ''}`.trim()} onFocusCapture={(event) => {
      // 局部滚动条的边缘按钮需完整可见；Select 的 portal 焦点不属于此滚动容器。
      const bar = event.currentTarget
      if (bar.contains(event.target) && bar.scrollWidth > bar.clientWidth) {
        event.target.scrollIntoView({ block: 'nearest', inline: 'nearest' })
      }
    }}>
      <div className={`${styles.credentialPaginationControls} ${controlsClassName ?? ''}`.trim()}>
        {sortOptions && sortOptions.length > 0 && sortLabel && onSortChange && (
          <div className={styles.credentialPageSizeControl}>
            <span>{sortLabel}</span>
            <div className={styles.credentialPaginationSortControl}>
              <span
                className={styles.credentialPaginationSortSizer}
                data-credential-pagination-sort-sizer="true"
                aria-hidden="true"
              >
                {sortOptions.map((option) => <span key={option.value}>{option.label}</span>)}
              </span>
              <Select
                value={sortValue ?? ''}
                options={sortOptions}
                onChange={onSortChange}
                className={`${styles.credentialPaginationSelect} ${styles.credentialPaginationSortSelect}`}
                ariaLabel={selectedSortLabel ? `${sortLabel}: ${selectedSortLabel}` : sortLabel}
                fullWidth
                dropdownMinWidth={180}
              />
            </div>
          </div>
        )}
        <div className={styles.credentialPageSizeControl}>
          <span>{rowsPerPageLabel}</span>
          <Select
            value={String(pageSize)}
            options={CREDENTIAL_PAGE_SIZE_SELECT_OPTIONS}
            onChange={(value) => onPageSizeChange(Number(value))}
            className={`${styles.credentialPaginationSelect} ${styles.credentialPaginationPageSizeSelect}`}
            ariaLabel={`${rowsPerPageLabel}: ${pageSize}`}
            fullWidth={false}
            dropdownMinWidth={72}
          />
        </div>
        <button type="button" onClick={() => onPageChange(page - 1)} disabled={page <= 1}>{previousLabel}</button>
        <span className={styles.credentialPaginationPage}>{page} / {totalPages}</span>
        <button type="button" onClick={() => onPageChange(page + 1)} disabled={page >= totalPages}>{nextLabel}</button>
      </div>
    </div>
  )
}

export function formatCredentialNumber(value: number): string {
  return formatCompactNumber(value)
}

export function formatCredentialPercent(value: number | null): string {
  if (value === null) {
    return '—'
  }
  return `${value.toFixed(2)}%`
}

export function credentialToneClassName(prefix: string, tone: string): string {
  return styles[`${prefix}${capitalize(tone)}`] ?? ''
}

export function capitalize(value: string): string {
  return value.charAt(0).toUpperCase() + value.slice(1)
}

// Fork-specific components for Provider Summary table layout
interface CredentialRowShellProps {
  icon: ReactNode
  title: ReactNode
  subtitle?: ReactNode
  badges: ReactNode
  metrics: ReactNode
  metricsTitle?: string
  side: ReactNode
  rowClassName?: string
}

interface CredentialTableHeaderProps {
  nameLabel: string
  totalRequestsLabel: string
  successRateLabel: string
  totalTokensLabel: string
  cacheReadRateLabel: string
  lastUsedLabel?: string
  sideLabel: string
  rowClassName?: string
}

export function CredentialRowShell({ icon, title, subtitle, badges, metrics, metricsTitle, side, rowClassName }: CredentialRowShellProps) {
  return (
    <article className={`${styles.credentialRow} ${rowClassName ?? ''}`.trim()}>
      <div className={styles.credentialIdentityBlock}>
        {icon}
        <div className={styles.credentialIdentityContent}>
          <div className={styles.credentialNameRow}>
            <div className={styles.credentialNameMain}>
              <span className={styles.credentialDisplayName}>{title}</span>
            </div>
            {badges && <div className={styles.credentialBadges}>{badges}</div>}
          </div>
          {subtitle && <span className={styles.credentialIdentityText}>{subtitle}</span>}
        </div>
      </div>
      <div className={styles.credentialMetricGroup} title={metricsTitle}>{metrics}</div>
      <div className={styles.credentialSidePanel}>{side}</div>
    </article>
  )
}

export function CredentialTableHeader({ nameLabel, totalRequestsLabel, successRateLabel, totalTokensLabel, cacheReadRateLabel, lastUsedLabel, sideLabel, rowClassName }: CredentialTableHeaderProps) {
  return (
    <div className={`${styles.credentialTableHeader} ${rowClassName ?? ''}`.trim()}>
      <span className={styles.credentialTableHeaderName}>{nameLabel}</span>
      <div className={styles.credentialMetricHeaderGroup}>
        <span className={styles.credentialMetricHeaderCell}>{totalRequestsLabel}</span>
        <span className={styles.credentialMetricHeaderCell}>{successRateLabel}</span>
        <span className={styles.credentialMetricHeaderCell}>{totalTokensLabel}</span>
        <span className={styles.credentialMetricHeaderCell}>{cacheReadRateLabel}</span>
        {lastUsedLabel && <span className={styles.credentialMetricHeaderCell}>{lastUsedLabel}</span>}
      </div>
      <span className={styles.credentialTableHeaderSide}>{sideLabel}</span>
    </div>
  )
}
