// @vitest-environment happy-dom

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
  MODEL_RANKING_POLL_INTERVAL_MS,
  useModelRankingData,
  type ModelRankingDataAPI,
} from '../hooks/useModelRankingData';
import type { ModelRankingMetric, ModelRankingPeriod, ModelRankingResponse } from '../types';

const board = (period: ModelRankingPeriod, metric: ModelRankingMetric, key = '2026-11-16'): ModelRankingResponse => ({
  period,
  period_key: key,
  metric,
  start: '2026-11-16T00:00:00Z',
  end: '2026-11-17T00:00:00Z',
  timezone: 'UTC',
  generated_at: '2026-11-16T04:00:00Z',
  entries: [{
    rank: 1,
    model: 'gpt-5',
    providers: ['openai'],
    request_count: 1,
    success_count: 1,
    failure_count: 0,
    input_tokens: 10,
    cache_read_tokens: 0,
    total_tokens: 10,
    value: 10,
  }],
});

interface ProbeProps {
  api: ModelRankingDataAPI;
  period: ModelRankingPeriod;
  metric: ModelRankingMetric;
  enabled?: boolean;
}

function Probe({ api, period, metric, enabled = true }: ProbeProps) {
  const state = useModelRankingData({ enabled, period, metric, api });
  return (
    <div
      data-loading={String(state.loading)}
      data-error={state.error ? 'error' : ''}
      data-key={state.leaderboard?.period_key ?? ''}
      data-stale={String(Boolean(state.leaderboard?.stale))}
      data-model={state.leaderboard?.entries[0]?.model ?? ''}
    />
  );
}

describe('useModelRankingData', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  const renderProbe = async (props: ProbeProps) => {
    await act(async () => root.render(<Probe {...props} />));
  };
  const probe = () => container.firstElementChild as HTMLElement;

  it('loads the selected period and metric', async () => {
    const api: ModelRankingDataAPI = { leaderboard: vi.fn(async (period, metric) => board(period, metric)) };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    expect(probe().getAttribute('data-model')).toBe('gpt-5');
    expect(probe().getAttribute('data-loading')).toBe('false');
    expect(probe().getAttribute('data-error')).toBe('');
    expect(api.leaderboard).toHaveBeenCalledWith('today', 'total_tokens', expect.any(AbortSignal));
  });

  it('aborts a stale request when the selection changes', async () => {
    const pending = Promise.withResolvers<ModelRankingResponse>();
    const leaderboard = vi.fn((period: ModelRankingPeriod, metric: ModelRankingMetric) => (
      period === 'today' ? pending.promise : Promise.resolve(board(period, metric, 'k2'))
    ));
    const api: ModelRankingDataAPI = { leaderboard };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    const firstSignal = leaderboard.mock.calls[0]?.[2];
    expect(firstSignal?.aborted).toBe(false);
    await renderProbe({ api, period: 'yesterday', metric: 'total_tokens' });
    expect(firstSignal?.aborted).toBe(true);
    expect(probe().getAttribute('data-key')).toBe('k2');
  });

  it('shows the cached board while a re-selected period refreshes', async () => {
    let calls = 0;
    const pending = Promise.withResolvers<ModelRankingResponse>();
    const api: ModelRankingDataAPI = {
      leaderboard: vi.fn((period, metric) => {
        calls += 1;
        if (calls === 3) return pending.promise;
        return Promise.resolve(board(period, metric, period === 'today' ? 'k1' : 'k2'));
      }),
    };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    expect(probe().getAttribute('data-key')).toBe('k1');
    await renderProbe({ api, period: 'yesterday', metric: 'total_tokens' });
    expect(probe().getAttribute('data-key')).toBe('k2');
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    expect(probe().getAttribute('data-key')).toBe('k1');
    expect(probe().getAttribute('data-loading')).toBe('true');
  });

  it('keeps the last board and marks it stale when a silent refresh fails', async () => {
    vi.useFakeTimers();
    let fail = false;
    const api: ModelRankingDataAPI = {
      leaderboard: vi.fn(async (period, metric) => {
        if (fail) throw new Error('offline');
        return board(period, metric);
      }),
    };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(probe().getAttribute('data-stale')).toBe('false');

    fail = true;
    await act(async () => { await vi.advanceTimersByTimeAsync(MODEL_RANKING_POLL_INTERVAL_MS); });
    expect(probe().getAttribute('data-stale')).toBe('true');
    expect(probe().getAttribute('data-model')).toBe('gpt-5');
    expect(probe().getAttribute('data-error')).toBe('');
  });

  it('does not poll while the document is hidden', async () => {
    vi.useFakeTimers();
    const visibility = vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('hidden');
    const api: ModelRankingDataAPI = { leaderboard: vi.fn(async (period, metric) => board(period, metric)) };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    await act(async () => { await vi.advanceTimersByTimeAsync(0); });
    expect(api.leaderboard).toHaveBeenCalledTimes(1);
    await act(async () => { await vi.advanceTimersByTimeAsync(MODEL_RANKING_POLL_INTERVAL_MS); });
    expect(api.leaderboard).toHaveBeenCalledTimes(1);
    visibility.mockRestore();
  });

  it('surfaces an initial failure', async () => {
    const api: ModelRankingDataAPI = { leaderboard: vi.fn(async () => { throw new Error('boom'); }) };
    await renderProbe({ api, period: 'today', metric: 'total_tokens' });
    expect(probe().getAttribute('data-error')).toBe('error');
    expect(probe().getAttribute('data-model')).toBe('');
  });

  it('does not fetch while disabled', async () => {
    const api: ModelRankingDataAPI = { leaderboard: vi.fn(async (period, metric) => board(period, metric)) };
    await renderProbe({ api, period: 'today', metric: 'total_tokens', enabled: false });
    expect(api.leaderboard).not.toHaveBeenCalled();
  });
});