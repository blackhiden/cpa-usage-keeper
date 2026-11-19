import type { ModelRankingPeriod } from './types';

export interface ModelRankingPeriodOption {
  value: ModelRankingPeriod;
  labelKey: string;
  triggerLabelKey: string;
}

// Display order matches the requested six-period dropdown.
export const MODEL_RANKING_PERIOD_OPTIONS: ReadonlyArray<ModelRankingPeriodOption> = [
  { value: 'today', labelKey: 'model_ranking.period_today', triggerLabelKey: 'model_ranking.period_trigger_today' },
  { value: 'yesterday', labelKey: 'model_ranking.period_yesterday', triggerLabelKey: 'model_ranking.period_trigger_yesterday' },
  { value: 'current_week', labelKey: 'model_ranking.period_current_week', triggerLabelKey: 'model_ranking.period_trigger_current_week' },
  { value: 'previous_week', labelKey: 'model_ranking.period_previous_week', triggerLabelKey: 'model_ranking.period_trigger_previous_week' },
  { value: 'current_month', labelKey: 'model_ranking.period_current_month', triggerLabelKey: 'model_ranking.period_trigger_current_month' },
  { value: 'previous_month', labelKey: 'model_ranking.period_previous_month', triggerLabelKey: 'model_ranking.period_trigger_previous_month' },
];

export const MODEL_RANKING_PERIOD_STORAGE_KEY = 'cpa-usage-keeper-model-ranking-period';
export const MODEL_RANKING_DEFAULT_PERIOD: ModelRankingPeriod = 'today';

const validPeriods = new Set<string>(MODEL_RANKING_PERIOD_OPTIONS.map((option) => option.value));

export function isModelRankingPeriod(value: unknown): value is ModelRankingPeriod {
  return typeof value === 'string' && validPeriods.has(value);
}

export function readStoredModelRankingPeriod(storage: Pick<Storage, 'getItem'> | undefined): ModelRankingPeriod {
  if (!storage) return MODEL_RANKING_DEFAULT_PERIOD;
  try {
    const stored = storage.getItem(MODEL_RANKING_PERIOD_STORAGE_KEY);
    return isModelRankingPeriod(stored) ? stored : MODEL_RANKING_DEFAULT_PERIOD;
  } catch {
    return MODEL_RANKING_DEFAULT_PERIOD;
  }
}

export function storeModelRankingPeriod(storage: Pick<Storage, 'setItem'> | undefined, period: ModelRankingPeriod): void {
  if (!storage) return;
  try {
    storage.setItem(MODEL_RANKING_PERIOD_STORAGE_KEY, period);
  } catch {
    // Persisting the selection is best-effort; the view still works without it.
  }
}