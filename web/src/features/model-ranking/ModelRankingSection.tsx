import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/Button';
import { LoadingSpinner } from '@/components/ui/LoadingSpinner';
import { Select } from '@/components/ui/Select';
import { formatModelRankingValue } from './format';
import { useModelRankingData } from './hooks/useModelRankingData';
import {
  isModelRankingPeriod,
  MODEL_RANKING_DEFAULT_PERIOD,
  MODEL_RANKING_PERIOD_OPTIONS,
  readStoredModelRankingPeriod,
  storeModelRankingPeriod,
} from './period';
import { MODEL_RANKING_METRICS, type ModelRankingMetric, type ModelRankingPeriod, type ModelRankingResponse } from './types';
import styles from './ModelRankingSection.module.scss';

const DEFAULT_METRIC: ModelRankingMetric = 'total_tokens';

export interface ModelRankingSectionProps {
  viewerScope?: boolean;
  linkedPeriod?: ModelRankingPeriod;
  onPeriodChange?: (period: ModelRankingPeriod) => void;
}

const METRIC_OPTIONS: ReadonlyArray<{ value: ModelRankingMetric; labelKey: string; triggerLabelKey: string }> = [
  { value: 'total_tokens', labelKey: 'model_ranking.metric_total_tokens', triggerLabelKey: 'model_ranking.metric_short_total_tokens' },
  { value: 'request_count', labelKey: 'model_ranking.metric_request_count', triggerLabelKey: 'model_ranking.metric_short_request_count' },
  { value: 'cache_read_rate', labelKey: 'model_ranking.metric_cache_read_rate', triggerLabelKey: 'model_ranking.metric_short_cache_read_rate' },
  { value: 'ttft_average', labelKey: 'model_ranking.metric_ttft_average', triggerLabelKey: 'model_ranking.metric_short_ttft_average' },
  { value: 'latency_average', labelKey: 'model_ranking.metric_latency_average', triggerLabelKey: 'model_ranking.metric_short_latency_average' },
  { value: 'peak_tpm', labelKey: 'model_ranking.metric_peak_tpm', triggerLabelKey: 'model_ranking.metric_short_peak_tpm' },
  { value: 'peak_rpm', labelKey: 'model_ranking.metric_peak_rpm', triggerLabelKey: 'model_ranking.metric_short_peak_rpm' },
  { value: 'overall', labelKey: 'model_ranking.metric_overall', triggerLabelKey: 'model_ranking.metric_short_overall' },
];

const resolveStorage = (): Storage | undefined => (typeof window === 'undefined' ? undefined : window.localStorage);

const isMetric = (value: string): value is ModelRankingMetric =>
  (MODEL_RANKING_METRICS as readonly string[]).includes(value);

const formatGeneratedAt = (value: string, language: string): string => {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(language, { dateStyle: 'medium', timeStyle: 'short', hour12: false }).format(date);
};

// ModelRankingSection renders its own leaderboard card on /ranking. It owns its
// period and metric selectors so the API-key leaderboard is never reloaded or
// reset by a model-period change.
export function ModelRankingSection({ viewerScope = false, linkedPeriod, onPeriodChange }: ModelRankingSectionProps) {
  const { t, i18n } = useTranslation();
  const [period, setPeriod] = useState<ModelRankingPeriod>(() => readStoredModelRankingPeriod(resolveStorage()));
  const [metric, setMetric] = useState<ModelRankingMetric>(DEFAULT_METRIC);
  const [provider, setProvider] = useState('');
  const [knownProviders, setKnownProviders] = useState<string[]>([]);
  const [periodsLinked, setPeriodsLinked] = useState<boolean>(() => {
    try { return window.localStorage.getItem('cpa-usage-keeper-model-ranking-link-periods') === 'true'; } catch { return false; }
  });
  const effectivePeriod = periodsLinked && linkedPeriod ? linkedPeriod : period;
  const [lastLeaderboard, setLastLeaderboard] = useState<ModelRankingResponse | null>(null);
  const { leaderboard, loading, error, refresh } = useModelRankingData({
    enabled: true,
    period: effectivePeriod,
    metric,
    provider,
    viewerScope,
  });
  const displayLeaderboard = leaderboard ?? lastLeaderboard;
  if (leaderboard && leaderboard !== lastLeaderboard) {
    setLastLeaderboard(leaderboard);
    setKnownProviders((current) => [...new Set([...current, ...leaderboard.entries.flatMap((entry) => entry.providers)])].sort());
  }

  const periodOptions = useMemo(
    () => MODEL_RANKING_PERIOD_OPTIONS.map((option) => ({
      value: option.value,
      label: t(option.labelKey),
      triggerLabel: t(option.triggerLabelKey),
    })),
    [t],
  );
  const metricOptions = useMemo(
    () => METRIC_OPTIONS.map((option) => ({
      value: option.value,
      label: t(option.labelKey),
      triggerLabel: t(option.triggerLabelKey),
    })),
    [t],
  );
  const currentPeriodLabel = periodOptions.find((option) => option.value === effectivePeriod)?.triggerLabel ?? effectivePeriod;
  const currentMetricLabel = metricOptions.find((option) => option.value === metric)?.triggerLabel ?? metric;
  const rows = useMemo(() => displayLeaderboard?.entries.slice(0, 100) ?? [], [displayLeaderboard]);

  const handlePeriodChange = useCallback((value: string) => {
    const next = periodsLinked && linkedPeriod ? linkedPeriod : isModelRankingPeriod(value) ? value : MODEL_RANKING_DEFAULT_PERIOD;
    storeModelRankingPeriod(resolveStorage(), next);
    setPeriod(next);
    onPeriodChange?.(next);
  }, [linkedPeriod, onPeriodChange, periodsLinked]);

  const handleLinkedChange = useCallback((checked: boolean) => {
    setPeriodsLinked(checked);
    try { window.localStorage.setItem('cpa-usage-keeper-model-ranking-link-periods', String(checked)); } catch { /* best-effort */ }
  }, []);

  const providerOptions = useMemo(() => {
    const providers = [...new Set([...knownProviders, ...(displayLeaderboard?.entries ?? []).flatMap((entry) => entry.providers)])].sort();
    return [{ value: '', label: t('model_ranking.provider_all'), triggerLabel: t('model_ranking.provider_all') }, ...providers.map((value) => ({ value, label: value }))];
  }, [displayLeaderboard, knownProviders, t]);

  const handleMetricChange = useCallback((value: string) => {
    if (isMetric(value)) setMetric(value);
  }, []);

  return (
    <article className={`card ${styles.sectionCard}`} aria-busy={loading} data-model-ranking-section>
      <header className={styles.header}>
        <h2 className={`${styles.title} keeper-card-title`} data-model-ranking-title>
          {t('model_ranking.title')}
        </h2>
        {viewerScope ? <span className={styles.scopeLabel} data-model-ranking-viewer-scope>{t('model_ranking.viewer_scope')}</span> : null}
        {metric === 'overall' && leaderboard?.score_explanation?.texts?.[i18n.language as 'en'|'zh'|'zh-TW'] ? <p className={styles.explanation} data-model-ranking-overall-explanation>{leaderboard.score_explanation.texts[i18n.language as 'en'|'zh'|'zh-TW']}</p> : null}
        <div className={styles.controls} data-model-ranking-controls>
          <Select
            value={effectivePeriod}
            options={periodOptions}
            onChange={handlePeriodChange}
            className={styles.periodSelect}
            ariaLabel={`${t('model_ranking.period_label')}: ${currentPeriodLabel}`}
            dropdownMinWidth={200}
            fullWidth={false}
            dataAttributes={{ 'data-model-ranking-period': period }}
          />
          {!viewerScope ? <label className={styles.linkPeriods}><input type="checkbox" checked={periodsLinked} onChange={(event) => handleLinkedChange(event.currentTarget.checked)} data-model-ranking-link-periods />{t('model_ranking.link_periods')}</label> : null}
          <Select value={provider} options={providerOptions} onChange={setProvider} ariaLabel={t('model_ranking.provider_filter')} dropdownMinWidth={180} fullWidth={false} dataAttributes={{ 'data-model-ranking-provider-filter': provider || 'all' }} />
          <Select
            value={metric}
            options={metricOptions}
            onChange={handleMetricChange}
            className={styles.metricSelect}
            ariaLabel={`${t('model_ranking.metric_label')}: ${currentMetricLabel}`}
            dropdownMinWidth={240}
            fullWidth={false}
            id="model-ranking-metric"
            dataAttributes={{ 'data-model-ranking-metric': metric }}
          />
        </div>
      </header>
      {displayLeaderboard ? (
        <div className={styles.meta}>
          {displayLeaderboard.stale ? (
            <span className={styles.staleBadge} data-model-ranking-stale>
              {t('model_ranking.stale')} · {displayLeaderboard.period_key}
            </span>
          ) : null}
          <span data-model-ranking-updated-at>
            {t('model_ranking.updated_at', { time: formatGeneratedAt(displayLeaderboard.generated_at, i18n.language) })}
          </span>
        </div>
      ) : null}
      {loading && !displayLeaderboard ? (
        <div className={styles.loadingState}>
          <LoadingSpinner size={20} />
          <span>{t('model_ranking.loading')}</span>
        </div>
      ) : error && !displayLeaderboard ? (
        <div className={styles.errorState} role="alert" data-model-ranking-error>
          <p>{t('model_ranking.error_generic')}</p>
          <Button variant="secondary" appearance="action" onClick={() => void refresh()}>
            {t('model_ranking.retry')}
          </Button>
        </div>
      ) : rows.length === 0 ? (
        <div className={styles.emptyState} data-model-ranking-empty>
          <strong>{t('model_ranking.empty_title')}</strong>
          <p>{t('model_ranking.empty_description')}</p>
        </div>
      ) : (
        <div className={styles.tableScroll}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th className={styles.rankColumn}>{t('model_ranking.rank')}</th>
                <th>{t('model_ranking.model')}</th>
                <th>{t('model_ranking.providers')}</th>
                <th className={styles.numberCell}>{t(`model_ranking.metric_short_${metric}`)}</th>
                <th className={styles.numberCell}>{t('model_ranking.cost')}</th>
                <th className={styles.numberCell}>{t('model_ranking.trend')}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((entry) => (
                <tr key={`${entry.rank}:${entry.model}`} data-model-ranking-row>
                  <td className={styles.rankColumn}>
                    <span className={styles.rankBadge}>{entry.rank}</span>
                  </td>
                  <td>
                    <strong className={styles.modelName}>{entry.model}</strong>
                  </td>
                  <td>
                    <div className={styles.providers}>
                      {entry.providers.length > 0
                        ? (entry.provider_contributions ?? entry.providers.map((name) => ({ provider: name, request_count: 0, total_tokens: 0 }))).map((share) => (
                          <span className={styles.providerBadge} key={share.provider} data-model-ranking-provider>
                            {share.provider}: {share.request_count} {t('model_ranking.requests_short')} · {share.total_tokens} {t('model_ranking.tokens_short')}
                          </span>
                        ))
                        : <span className={styles.noProvider}>—</span>}
                    </div>
                  </td>
                  <td className={`${styles.numberCell} ${styles.valueCell}`.trim()}>
                    {formatModelRankingValue(metric, entry)}
                  </td>
                  <td className={styles.numberCell}>{(entry.unavailable_cost_count ?? 0) > 0 ? t('model_ranking.cost_partial', { cost: (entry.cost_usd ?? 0).toFixed(4), count: entry.unavailable_cost_count }) : `$${(entry.cost_usd ?? 0).toFixed(4)}`}</td>
                  <td className={styles.numberCell}>{entry.trend?.available ? t('model_ranking.trend_change', { change: entry.trend.change, percent: entry.trend.percent?.toFixed(1) ?? '—' }) : t('model_ranking.trend_unavailable')}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </article>
  );
}