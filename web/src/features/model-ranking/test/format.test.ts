import { describe, expect, it } from 'vitest';
import { formatModelRankingValue } from '../format';
import type { ModelRankingEntry } from '../types';

const entry: ModelRankingEntry = {
  rank: 1,
  model: 'gpt-5',
  providers: ['openai'],
  request_count: 2,
  success_count: 1,
  failure_count: 1,
  input_tokens: 200,
  cache_read_tokens: 50,
  total_tokens: 100,
  value: 100,
  rate_numerator: 50,
  rate_denominator: 200,
};

describe('formatModelRankingValue', () => {
  it('formats counts compactly', () => {
    expect(formatModelRankingValue('total_tokens', { ...entry, value: 999 })).toBe('999');
    expect(formatModelRankingValue('total_tokens', { ...entry, value: 1_500 })).toBe('1.50K');
    expect(formatModelRankingValue('request_count', { ...entry, value: 2_000_000 })).toBe('2.00M');
  });

  it('formats cache read rate as a percentage', () => {
    expect(formatModelRankingValue('cache_read_rate', entry)).toBe('25.00%');
    expect(formatModelRankingValue('cache_read_rate', { ...entry, rate_numerator: 0, rate_denominator: 0, value: 250_000 })).toBe('25.00%');
  });

  it('formats TTFT and latency as durations', () => {
    expect(formatModelRankingValue('ttft_average', { ...entry, rate_numerator: 250, rate_denominator: 1 })).toBe('250ms');
    expect(formatModelRankingValue('latency_average', { ...entry, rate_numerator: 1_500, rate_denominator: 1 })).toBe('1.50s');
    expect(formatModelRankingValue('latency_average', { ...entry, rate_numerator: 0, rate_denominator: 0, value: 750_000 })).toBe('750ms');
  });

  it('converts five-minute peak buckets to per-minute values', () => {
    expect(formatModelRankingValue('peak_tpm', { ...entry, value: 500 })).toBe('100');
    expect(formatModelRankingValue('peak_rpm', { ...entry, value: 1_000 })).toBe('200');
  });
});