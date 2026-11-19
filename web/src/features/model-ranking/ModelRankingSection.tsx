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
import { MODEL_RANKING_METRICS, type ModelRankingMetric, type ModelRankingPeriod } from './types';
import styles from './ModelRankingSection.module.scss';

const DEFAULT_METRIC: ModelRankingMetric = 'total_tokens';

const METRIC_OPTIONS: ReadonlyArray<{ value: ModelRankingMetric; labelKey: string; triggerLabelKey: string }> = [
  { value: 'total_tokens', labelKey: 'model_ranking.metric_total_tokens', triggerLabelKey: 'model_ranking.metric_short_total_tokens' },
  { value: 'request_count', labelKey: 'model_ranking.metric_request_count', triggerLabelKey: 'model_ranking.metric_short_request_count' },
  { value: 'cache_read_rate', labelKey: 'model_ranking.metric_cache_read_rate', triggerLabelKey: 'model_ranking.metric_short_cache_read_rate' },
  { value: 'ttft_average', labelKey: 'model_ranking.metric_ttft_average', triggerLabelKey: 'model_ranking.metric_short_ttft_average' },
  { value: 'latency_average', labelKey: 'model_ranking.metric_latency_average', triggerLabelKey: 'model_ranking.metric_short_latency_average' },
  { value: 'peak_tpm', labelKey: 'model_ranking.metric_peak_tpm', triggerLabelKey: 'model_ranking.metric_short_peak_tpm' },
  { value: 'peak_rpm', labelKey: 'model_ranking.metric_peak_rpm', triggerLabelKey: 'model_ranking.metric_short_peak_rpm' },
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
export function ModelRankingSection() {
  const { t, i18n } = useTranslation();
  const [period, setPeriod] = useState<ModelRankingPeriod>(() => readStoredModelRankingPeriod(resolveStorage()));
  const [metric, setMetric] = useState<ModelRankingMetric>(DEFAULT_METRIC);
  const { leaderboard, loading, error, refresh } = useModelRankingData({
    enabled: true,
    period,
    metric,
  });

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
  const currentPeriodLabel = periodOptions.find((option) => option.value === period)?.triggerLabel ?? period;
  const currentMetricLabel = metricOptions.find((option) => option.value === metric)?.triggerLabel ?? metric;
  const rows = useMemo(() => leaderboard?.entries.slice(0, 100) ?? [], [leaderboard]);

  const handlePeriodChange = useCallback((value: string) => {
    const next = isModelRankingPeriod(value) ? value : MODEL_RANKING_DEFAULT_PERIOD;
    storeModelRankingPeriod(resolveStorage(), next);
    setPeriod(next);
  }, []);

  const handleMetricChange = useCallback((value: string) => {
    if (isMetric(value)) setMetric(value);
  }, []);

  return (
    <article className={`card ${styles.sectionCard}`} aria-busy={loading} data-model-ranking-section>
      <header className={styles.header}>
        <h2 className={`${styles.title} keeper-card-title`} data-model-ranking-title>
          {t('model_ranking.title')}
        </h2>
        <div className={styles.controls} data-model-ranking-controls>
          <Select
            value={period}
            options={periodOptions}
            onChange={handlePeriodChange}
            className={styles.periodSelect}
            ariaLabel={`${t('model_ranking.period_label')}: ${currentPeriodLabel}`}
            dropdownMinWidth={200}
            fullWidth={false}
            dataAttributes={{ 'data-model-ranking-period': period }}
          />
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
      {leaderboard ? (
        <div className={styles.meta}>
          {leaderboard.stale ? (
            <span className={styles.staleBadge} data-model-ranking-stale>
              {t('model_ranking.stale')} · {leaderboard.period_key}
            </span>
          ) : null}
          <span data-model-ranking-updated-at>
            {t('model_ranking.updated_at', { time: formatGeneratedAt(leaderboard.generated_at, i18n.language) })}
          </span>
        </div>
      ) : null}
      {loading && !leaderboard ? (
        <div className={styles.loadingState}>
          <LoadingSpinner size={20} />
          <span>{t('model_ranking.loading')}</span>
        </div>
      ) : error && !leaderboard ? (
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
                        ? entry.providers.map((provider) => (
                          <span className={styles.providerBadge} key={provider} data-model-ranking-provider>
                            {provider}
                          </span>
                        ))
                        : <span className={styles.noProvider}>—</span>}
                    </div>
                  </td>
                  <td className={`${styles.numberCell} ${styles.valueCell}`.trim()}>
                    {formatModelRankingValue(metric, entry)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </article>
  );
}