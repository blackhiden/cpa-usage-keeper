import { describe, expect, it } from 'vitest';
import {
  isModelRankingPeriod,
  MODEL_RANKING_DEFAULT_PERIOD,
  MODEL_RANKING_PERIOD_OPTIONS,
  MODEL_RANKING_PERIOD_STORAGE_KEY,
  readStoredModelRankingPeriod,
  storeModelRankingPeriod,
} from '../period';
import type { ModelRankingPeriod } from '../types';

class MemoryStorage {
  private readonly values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

describe('model ranking periods', () => {
  it('offers exactly the six requested periods in display order', () => {
    const values = MODEL_RANKING_PERIOD_OPTIONS.map((option) => option.value);
    expect(values).toEqual([
      'today',
      'yesterday',
      'current_week',
      'previous_week',
      'current_month',
      'previous_month',
    ]);
    expect(new Set(values).size).toBe(6);
  });

  it('validates period values', () => {
    for (const option of MODEL_RANKING_PERIOD_OPTIONS) {
      expect(isModelRankingPeriod(option.value)).toBe(true);
    }
    for (const invalid of ['', 'current_day', 'this_week', 'TODAY', null, undefined, 1]) {
      expect(isModelRankingPeriod(invalid)).toBe(false);
    }
  });

  it('defaults to today and ignores a missing or invalid stored value', () => {
    expect(MODEL_RANKING_DEFAULT_PERIOD).toBe('today');
    expect(readStoredModelRankingPeriod(undefined)).toBe('today');
    const storage = new MemoryStorage();
    expect(readStoredModelRankingPeriod(storage)).toBe('today');
    storage.setItem(MODEL_RANKING_PERIOD_STORAGE_KEY, 'bogus');
    expect(readStoredModelRankingPeriod(storage)).toBe('today');
    storage.setItem(MODEL_RANKING_PERIOD_STORAGE_KEY, 'previous_week');
    expect(readStoredModelRankingPeriod(storage)).toBe('previous_week');
  });

  it('survives a throwing storage implementation', () => {
    const throwing = {
      getItem: () => { throw new Error('denied'); },
      setItem: () => { throw new Error('denied'); },
    };
    expect(readStoredModelRankingPeriod(throwing)).toBe('today');
    expect(() => storeModelRankingPeriod(throwing, 'yesterday')).not.toThrow();
  });

  it('persists the selected period under its dedicated key', () => {
    const storage = new MemoryStorage();
    storeModelRankingPeriod(storage, 'current_month' as ModelRankingPeriod);
    expect(storage.getItem(MODEL_RANKING_PERIOD_STORAGE_KEY)).toBe('current_month');
  });
});