// @vitest-environment happy-dom

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';
import { ModelRankingSection } from '../ModelRankingSection';
import { MODEL_RANKING_PERIOD_STORAGE_KEY } from '../period';
import type { ModelRankingResponse } from '../types';

const api = vi.hoisted(() => ({ fetchModelRankingLeaderboard: vi.fn() }));

vi.mock('../api', () => ({
  fetchModelRankingLeaderboard: api.fetchModelRankingLeaderboard,
  ModelRankingApiError: class ModelRankingApiError extends Error {
    status: number;
    code: string;

    constructor(code: string, status: number) {
      super(code);
      this.status = status;
      this.code = code;
    }
  },
}));

const response = (overrides: Partial<ModelRankingResponse> = {}): ModelRankingResponse => ({
  period: 'today',
  period_key: '2026-11-16',
  metric: 'total_tokens',
  start: '2026-11-16T00:00:00Z',
  end: '2026-11-17T00:00:00Z',
  timezone: 'UTC',
  generated_at: '2026-11-16T04:00:00Z',
  entries: [{
    rank: 1,
    model: 'gpt-5',
    providers: ['openai', 'azure'],
    request_count: 3,
    success_count: 2,
    failure_count: 1,
    input_tokens: 200,
    cache_read_tokens: 50,
    total_tokens: 100,
    value: 100,
    cost_usd: 0.125,
    unavailable_cost_count: 0,
    top_spender: { display_name: 'Alpha Key', masked_key: 'sk-*********7890' },
    provider_contributions: [
      { provider: 'openai', request_count: 2, total_tokens: 70 },
      { provider: 'azure', request_count: 1, total_tokens: 30 },
    ],
    trend: { change: 25, percent: 33.3, previous_value: 75, available: true },
  }],
  ...overrides,
});

const optionTexts = () => Array.from(document.querySelectorAll('[role="option"]'), (node) => node.textContent);

describe('ModelRankingSection', () => {
  let container: HTMLDivElement;
  let root: Root;

  beforeEach(async () => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    await i18n.changeLanguage('en');
    localStorage.clear();
    api.fetchModelRankingLeaderboard.mockReset();
    api.fetchModelRankingLeaderboard.mockResolvedValue(response());
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    localStorage.clear();
    vi.restoreAllMocks();
  });

  const render = async () => {
    await act(async () => root.render(<ModelRankingSection />));
  };
  const section = () => container.querySelector<HTMLElement>('[data-model-ranking-section]')!;
  const periodSelect = () => container.querySelector<HTMLElement>('[data-model-ranking-period]')!;
  const metricSelect = () => container.querySelector<HTMLElement>('[data-model-ranking-metric]')!;

  it('renders the default today selection with alias-collapsed providers', async () => {
    await render();
    expect(section()).not.toBeNull();
    expect(periodSelect().getAttribute('data-model-ranking-period')).toBe('today');
    expect(metricSelect().getAttribute('data-model-ranking-metric')).toBe('total_tokens');
    expect(container.querySelectorAll('[data-model-ranking-row]')).toHaveLength(1);
    expect(container.querySelector('[data-model-ranking-row] strong')?.textContent).toBe('gpt-5');
    expect(Array.from(container.querySelectorAll('[data-model-ranking-provider]'), (node) => node.textContent?.split(':')[0])).toEqual(['openai', 'azure']);
    expect(container.querySelector('[data-model-ranking-updated-at]')?.textContent).toContain('Updated');
    expect(api.fetchModelRankingLeaderboard).toHaveBeenCalledWith('today', 'total_tokens', expect.any(AbortSignal), '', false);
    expect(container.querySelector('[data-model-ranking-row]')?.textContent).toContain('$0.1250');
    expect(container.querySelector('[data-model-ranking-row]')?.textContent).toContain('33.3%');
    expect(container.querySelector('thead')?.textContent).toContain('Top spender');
    const header = Array.from(container.querySelectorAll('thead th')).find((node) => node.textContent === 'Top spender');
    expect(header?.getAttribute('title')).toContain('selected metric');
    const spender = container.querySelector('[data-model-ranking-top-spender]');
    expect(spender?.textContent).toBe('Alpha Key');
    expect(spender?.getAttribute('title')).toBe('sk-*********7890');
  });

  it('renders a dash when a model has no attributable spender', async () => {
    api.fetchModelRankingLeaderboard.mockResolvedValue(response({
      entries: [{ ...response().entries[0], top_spender: undefined }],
    }));
    await render();
    expect(container.querySelector('[data-model-ranking-top-spender]')?.textContent).toBe('—');
  });

  it('offers exactly six periods and persists a period change', async () => {
    await render();
    await act(async () => container.querySelector<HTMLButtonElement>('[data-model-ranking-controls] button')!.click());
    expect(optionTexts()).toEqual(['Today', 'Yesterday', 'Current week', 'Last week', 'Current month', 'Last month']);
    const lastWeek = Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find((node) => node.textContent === 'Last week')!;
    await act(async () => lastWeek.click());
    expect(api.fetchModelRankingLeaderboard).toHaveBeenLastCalledWith('previous_week', 'total_tokens', expect.any(AbortSignal), '', false);
    expect(localStorage.getItem(MODEL_RANKING_PERIOD_STORAGE_KEY)).toBe('previous_week');
    expect(periodSelect().getAttribute('data-model-ranking-period')).toBe('previous_week');
  });

  it('switches the metric without touching the period', async () => {
    await render();
    const triggers = container.querySelectorAll<HTMLButtonElement>('[data-model-ranking-controls] button');
    await act(async () => triggers[2].click());
    expect(optionTexts()).toEqual(['Total tokens', 'Request count', 'Cache read rate', 'Average TTFT', 'Average latency', 'Peak TPM', 'Peak RPM', 'Overall score']);
    const requests = Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find((node) => node.textContent === 'Request count')!;
    await act(async () => requests.click());
    expect(api.fetchModelRankingLeaderboard).toHaveBeenLastCalledWith('today', 'request_count', expect.any(AbortSignal), '', false);
    expect(metricSelect().getAttribute('data-model-ranking-metric')).toBe('request_count');
    expect(container.querySelector('thead')?.textContent).toContain('Requests');
  });

  it('restores a stored period on mount', async () => {
    localStorage.setItem(MODEL_RANKING_PERIOD_STORAGE_KEY, 'previous_month');
    await render();
    expect(periodSelect().getAttribute('data-model-ranking-period')).toBe('previous_month');
    expect(api.fetchModelRankingLeaderboard).toHaveBeenCalledWith('previous_month', 'total_tokens', expect.any(AbortSignal), '', false);
  });

  it('renders the empty state for a period without model usage', async () => {
    api.fetchModelRankingLeaderboard.mockResolvedValue(response({ entries: [] }));
    await render();
    const empty = container.querySelector('[data-model-ranking-empty]');
    expect(empty).not.toBeNull();
    expect(empty?.textContent).toContain('No model usage in this period');
    expect(container.querySelectorAll('[data-model-ranking-row]')).toHaveLength(0);
  });

  it('renders an error state with a working retry', async () => {
    api.fetchModelRankingLeaderboard.mockRejectedValueOnce(new Error('boom'));
    await render();
    const error = container.querySelector('[data-model-ranking-error]');
    expect(error).not.toBeNull();
    expect(error?.textContent).toContain('Could not load the model ranking.');
    await act(async () => container.querySelector<HTMLButtonElement>('[data-model-ranking-error] button')!.click());
    expect(container.querySelector('[data-model-ranking-error]')).toBeNull();
    expect(container.querySelectorAll('[data-model-ranking-row]')).toHaveLength(1);
  });

  it('never calls the API-key leaderboard endpoint', async () => {
    const network = vi.spyOn(globalThis, 'fetch');
    await render();
    await act(async () => container.querySelector<HTMLButtonElement>('[data-model-ranking-controls] button')!.click());
    const option = Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find((node) => node.textContent === 'Yesterday')!;
    await act(async () => option.click());
    const targets = network.mock.calls.map((call) => String(call[0]));
    expect(targets.every((target) => !target.includes('/ranking/local/leaderboards'))).toBe(true);
    expect(targets).toHaveLength(0);
  });

  it('filters the model board by a selected provider', async () => {
    await render();
    const triggers = container.querySelectorAll<HTMLButtonElement>('[data-model-ranking-controls] button');
    await act(async () => triggers[1].click());
    expect(optionTexts()).toContain('openai');
    const provider = Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find((node) => node.textContent === 'openai')!;
    await act(async () => provider.click());
    expect(api.fetchModelRankingLeaderboard).toHaveBeenLastCalledWith('today', 'total_tokens', expect.any(AbortSignal), 'openai', false);
  });

  it('shows overall score explanation and unavailable costs', async () => {
    api.fetchModelRankingLeaderboard.mockResolvedValue(response({
      metric: 'overall',
      score_explanation: { version: 1, texts: { en: 'Normalized model score' } },
      entries: [{ ...response().entries[0], value: 80, unavailable_cost_count: 2 }],
    }));
    localStorage.setItem(MODEL_RANKING_PERIOD_STORAGE_KEY, 'today');
    await render();
    // Switch the metric to overall, which is the final option.
    const triggers = container.querySelectorAll<HTMLButtonElement>('[data-model-ranking-controls] button');
    await act(async () => triggers[2].click());
    const option = Array.from(document.querySelectorAll<HTMLElement>('[role="option"]')).find((node) => node.textContent === 'Overall score')!;
    await act(async () => option.click());
    expect(container.querySelector('[data-model-ranking-overall-explanation]')?.textContent).toContain('Normalized model score');
    expect(container.querySelector('[data-model-ranking-row]')?.textContent).toContain('unavailable');
  });
});