import { describe, expect, it } from 'vitest';
import i18n, { SUPPORTED_LANGUAGES } from '../index';

describe('ranking translations', () => {
  it.each(SUPPORTED_LANGUAGES)('includes tab_ranking in usage_stats for %s', (language) => {
    const usageStats = i18n.getResourceBundle(language, 'translation').usage_stats;
    expect(usageStats.tab_ranking).toEqual(expect.any(String));
    expect(usageStats.tab_ranking.length).toBeGreaterThan(0);
  });

  it.each(SUPPORTED_LANGUAGES)('includes the ranking namespace for %s', (language) => {
    const ranking = i18n.getResourceBundle(language, 'translation').ranking;
    expect(ranking).toBeDefined();
    expect(typeof ranking).toBe('object');
  });

  it.each(SUPPORTED_LANGUAGES)('includes all core ranking keys in %s', (language) => {
    const ranking = i18n.getResourceBundle(language, 'translation').ranking;
    
    // Core ranking keys that must exist
    const requiredKeys = [
      'title', 'api_key', 'participant', 'rank', 'score',
      'metric_label', 'metric_overall', 'metric_total_tokens', 'metric_request_count',
      'metric_cache_read_rate', 'metric_ttft_average', 'metric_latency_average',
      'metric_peak_tpm', 'metric_peak_rpm',
      'metric_short_overall', 'metric_short_total_tokens', 'metric_short_request_count',
      'metric_short_cache_read_rate', 'metric_short_ttft_average', 'metric_short_latency_average',
      'metric_short_peak_tpm', 'metric_short_peak_rpm',
      'period_today', 'period_yesterday', 'period_current_month', 'period_previous_month',
      'period_trigger_today', 'period_trigger_yesterday', 'period_trigger_current_month', 'period_trigger_previous_month',
      'updated_at', 'stale',
      'score_explanation_label',
      'local_empty_title', 'local_empty_description',
      'empty_title', 'empty_description',
      'leaderboard_loading', 'metadata_loading',
      'offline_title', 'offline_description',
      'refresh_failed',
      'never_synced', 'last_sync', 'last_complete_day', 'last_error',
      'status_loading', 'status_joining', 'status_active', 'status_paused', 'status_deleted',
      'join', 'join_retry', 'join_confirm_title', 'join_confirm_body', 'join_confirm_action',
      'joining_description',
      'participation_title', 'profile_action',
      'display_name', 'display_name_hint',
      'avatar', 'avatar_option',
      'profile_error_required', 'profile_error_too_long', 'profile_error_characters', 'profile_error_phone',
      'privacy_title', 'privacy_description',
      'sync_now', 'pause', 'pause_confirm_title', 'pause_confirm_body', 'pause_confirm_action',
      'paused_description', 'resume',
      'exit', 'exit_confirm_title', 'exit_confirm_body', 'exit_confirm_action',
      'deleted_title', 'deleted_description',
      'action_sync_success', 'action_pause_success', 'action_resume_success', 'action_exit_success',
      'error_generic', 'error_invalid_profile', 'error_profile_locked', 'error_deleted',
      'error_sync_in_progress', 'error_participation_state', 'error_center_incompatible',
      'error_rate_limited_generic', 'error_rate_limited_seconds', 'error_rate_limited_minutes',
      'local_profile_edit', 'local_profile_edit_label', 'local_profile_alias', 'local_profile_alias_hint',
      'local_profile_save', 'local_profile_save_failed'
    ];

    for (const key of requiredKeys) {
      expect(ranking[key], `${language}:ranking.${key}`).toEqual(expect.any(String));
      expect(ranking[key].length, `${language}:ranking.${key}`).toBeGreaterThan(0);
    }
  });

  it('has identical key sets across all locales for ranking namespace', () => {
    const enKeys = Object.keys(i18n.getResourceBundle('en', 'translation').ranking).sort();
    
    for (const language of SUPPORTED_LANGUAGES) {
      if (language === 'en') continue;
      const keys = Object.keys(i18n.getResourceBundle(language, 'translation').ranking).sort();
      expect(keys, `${language} ranking keys should match en`).toEqual(enKeys);
    }
  });

  it('has matching interpolation parameters across locales for ranking namespace', () => {
    const enRanking = i18n.getResourceBundle('en', 'translation').ranking;
    const extractParams = (text: string) => (text.match(/\{\{[^{}]+\}\}/g) ?? []).sort();

    for (const [key, enValue] of Object.entries(enRanking)) {
      if (typeof enValue !== 'string') continue;
      const enParams = extractParams(enValue);

      for (const language of SUPPORTED_LANGUAGES) {
        if (language === 'en') continue;
        const translated = i18n.getResourceBundle(language, 'translation').ranking[key];
        if (typeof translated === 'string') {
          const translatedParams = extractParams(translated);
          expect(translatedParams, `${language}:ranking.${key} params`).toEqual(enParams);
        }
      }
    }
  });
});
