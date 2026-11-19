// @vitest-environment happy-dom

import { act } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import i18n from '@/i18n';

vi.mock('react-chartjs-2', () => ({
  Bar: () => null,
  Chart: () => null,
  Doughnut: () => null,
  Line: () => null,
  Scatter: () => null,
}));

vi.mock('@/features/usage', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/features/usage')>(),
  LiveFlowPanel: () => <div data-live-flow-page-panel />,
}));

vi.mock('@/lib/api', async (importOriginal) => ({
  ...await importOriginal<typeof import('@/lib/api')>(),
  fetchStatus: async () => ({
    timezone: 'UTC',
    cpa_public_url: 'https://cpa.example.test',
    cpa_request_log_access_enabled: true,
  }),
  fetchVersion: async () => ({ version: 'test' }),
  fetchCpaApiKeyOptions: async () => ({ options: [] }),
  fetchCpaApiKeySettings: async () => ({ items: [] }),
  fetchAuthSessions: async () => ({ items: [] }),
  fetchPricingModels: async () => ({ models: [], config_revision: 0 }),
  fetchPricingModelOptions: async () => ({ models: [] }),
  fetchCurrentPricingRecalculation: async () => null,
  fetchAdminSettings: async () => ({ api_key_configured: false, request_timeout_seconds: 30 }),
  fetchUsageEventModelFilterOptions: async () => ({ models: [] }),
  fetchUsageEventSourceFilterOptions: async () => ({ sources: [] }),
}));

import { UsagePage } from '../UsagePage';

const json = (body: unknown) => new Response(JSON.stringify(body), {
  status: 200,
  headers: { 'Content-Type': 'application/json' },
});

const keyBoard = {
  period: 'today',
  period_key: '2026-11-16',
  metric: 'overall',
  generated_at: '2026-11-16T04:00:00Z',
  stale: false,
  entries: [],
};

const modelBoard = {
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
  }],
};

describe('UsagePage model ranking composition', () => {
  let container: HTMLDivElement;
  let root: Root;
  let fetchMock: ReturnType<typeof vi.spyOn>;

  beforeEach(async () => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    await i18n.changeLanguage('en');
    localStorage.clear();
    window.history.replaceState(null, '', '/ranking');
    fetchMock = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url.includes('/ranking/local/leaderboards')) return json(keyBoard);
      if (url.includes('/model-ranking/leaderboards')) return json(modelBoard);
      return json({});
    });
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

  it('renders the API-key card and the model card on the ranking tab', async () => {
    await act(async () => root.render(<UsagePage />));
    await act(async () => { await Promise.resolve(); });
    await act(async () => { await Promise.resolve(); });

    expect(container.querySelector('[data-ranking-toolbar]')).not.toBeNull();
    expect(container.querySelector('[data-model-ranking-section]')).not.toBeNull();
    expect(container.querySelector('[data-model-ranking-row]')?.textContent).toContain('gpt-5');
    expect(Array.from(container.querySelectorAll('[data-model-ranking-provider]'), (node) => node.textContent?.split(':')[0])).toEqual(['openai', 'azure']);

    const targets = fetchMock.mock.calls.map((call) => String(call[0]));
    expect(targets.some((target) => target.includes('/ranking/local/leaderboards'))).toBe(true);
    expect(targets.some((target) => target.includes('/model-ranking/leaderboards'))).toBe(true);
  });
});