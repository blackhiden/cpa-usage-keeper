import { describe, expect, it } from 'vitest';
import i18n, { SUPPORTED_LANGUAGES } from '../index';

const REQUIRED_KEYS = [
  'title',
  'period_label',
  'period_today', 'period_yesterday', 'period_current_week', 'period_previous_week',
  'period_current_month', 'period_previous_month',
  'period_trigger_today', 'period_trigger_yesterday', 'period_trigger_current_week',
  'period_trigger_previous_week', 'period_trigger_current_month', 'period_trigger_previous_month',
  'metric_label',
  'metric_total_tokens', 'metric_request_count', 'metric_cache_read_rate',
  'metric_ttft_average', 'metric_latency_average', 'metric_peak_tpm', 'metric_peak_rpm',
  'metric_short_total_tokens', 'metric_short_request_count', 'metric_short_cache_read_rate',
  'metric_short_ttft_average', 'metric_short_latency_average', 'metric_short_peak_tpm',
  'metric_short_peak_rpm',
  'rank', 'model', 'providers', 'updated_at', 'stale', 'loading',
  'empty_title', 'empty_description', 'error_generic', 'retry',
  'metric_overall', 'metric_short_overall', 'provider_all', 'provider_filter',
  'link_periods', 'cost', 'cost_partial', 'trend', 'trend_change',
  'trend_unavailable', 'requests_short', 'tokens_short', 'viewer_scope',
];

const bundle = (language: string) => i18n.getResourceBundle(language, 'translation').model_ranking;

describe('model ranking translations', () => {
  it.each(SUPPORTED_LANGUAGES)('includes every model_ranking key in %s', (language) => {
    const namespace = bundle(language);
    expect(namespace).toBeDefined();
    expect(typeof namespace).toBe('object');
    for (const key of REQUIRED_KEYS) {
      expect(namespace[key], `${language}:model_ranking.${key}`).toEqual(expect.any(String));
      expect(namespace[key].length, `${language}:model_ranking.${key}`).toBeGreaterThan(0);
    }
  });

  it('has identical key sets across all locales', () => {
    const enKeys = Object.keys(bundle('en')).sort();
    for (const language of SUPPORTED_LANGUAGES) {
      if (language === 'en') continue;
      expect(Object.keys(bundle(language)).sort(), `${language} model_ranking keys`).toEqual(enKeys);
    }
  });

  it('keeps interpolation parameters in sync', () => {
    const extractParams = (text: string) => (text.match(/\{\{[^{}]+\}\}/g) ?? []).sort();
    for (const language of SUPPORTED_LANGUAGES) {
      expect(extractParams(bundle(language).updated_at), `${language}:updated_at params`).toEqual(['{{time}}']);
    }
  });
});