// @vitest-environment happy-dom

import { renderToStaticMarkup } from 'react-dom/server';
import { describe, expect, it, vi } from 'vitest';
import { KeyLiveFlowPage } from '../KeyLiveFlowPage';

vi.mock('react-i18next', async (importOriginal) => ({
  ...await importOriginal<typeof import('react-i18next')>(),
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'en' } }),
}));

vi.mock('@/features/key-viewer/KeyViewerShell', () => ({
  KeyViewerShell: ({ children }: { children: React.ReactNode }) => <div data-key-viewer-shell>{children}</div>,
}));

vi.mock('@/features/usage/LiveFlowPanel', () => ({
  LiveFlowPanel: () => <div data-live-flow-panel />,
}));

describe('KeyLiveFlowPage', () => {
  it('renders the privacy notice with global view warning', () => {
    const html = renderToStaticMarkup(
      <KeyLiveFlowPage onNavigate={vi.fn()} />
    );
    
    // Privacy notice should be present with role="alert"
    expect(html).toContain('role="alert"');
    
    // Privacy notice should have the title and message translation keys
    expect(html).toContain('key_viewer.live_flow_privacy_title');
    expect(html).toContain('key_viewer.live_flow_privacy_notice');
  });

  it('renders the LiveFlowPanel component', () => {
    const html = renderToStaticMarkup(
      <KeyLiveFlowPage onNavigate={vi.fn()} />
    );
    
    expect(html).toContain('data-live-flow-panel');
  });

  it('wraps content in KeyViewerShell with liveflow active page', () => {
    const html = renderToStaticMarkup(
      <KeyLiveFlowPage onNavigate={vi.fn()} />
    );
    
    expect(html).toContain('data-key-viewer-shell');
  });

  it('displays privacy notice before LiveFlowPanel', () => {
    const html = renderToStaticMarkup(
      <KeyLiveFlowPage onNavigate={vi.fn()} />
    );
    
    const privacyIndex = html.indexOf('role="alert"');
    const panelIndex = html.indexOf('data-live-flow-panel');
    
    expect(privacyIndex).toBeGreaterThan(-1);
    expect(panelIndex).toBeGreaterThan(-1);
    expect(privacyIndex).toBeLessThan(panelIndex);
  });
});
