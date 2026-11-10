// @vitest-environment happy-dom

import { act, type ReactNode } from 'react';
import { createRoot } from 'react-dom/client';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ProviderDataDialog } from '../ProviderDataDialog';

vi.mock('react-i18next', () => {
  const t = (key: string) => key;
  return {
    initReactI18next: { type: '3rdParty', init: () => undefined },
    useTranslation: () => ({ t }),
  };
});

describe('ProviderDataDialog', () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    vi.useFakeTimers();
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    container = document.createElement('div');
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(async () => {
    await act(async () => root.unmount());
    container.remove();
    document.body.style.cssText = '';
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  async function render(children: ReactNode) {
    await act(async () => root.render(children));
  }

  it('renders dialog when open is true', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="Test Dialog"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Dialog Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const dialog = document.body.querySelector('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog?.textContent).toContain('Test Dialog');
    expect(dialog?.textContent).toContain('Dialog Content');
  });

  it('is hidden when open is false', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={false}
        onClose={onClose}
        title="Test Dialog"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Dialog Content</div>
      </ProviderDataDialog>
    );

    const dialog = document.body.querySelector('[role="dialog"]');
    expect(dialog).toBeNull();
  });

  it('displays accessibility attributes', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="Accessible Dialog"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const dialog = document.body.querySelector('[role="dialog"]');
    expect(dialog).toBeTruthy();
    expect(dialog?.getAttribute('aria-modal')).toBe('true');

    const viewToggle = document.body.querySelector('[role="group"]');
    expect(viewToggle).toBeTruthy();
    expect(viewToggle?.getAttribute('aria-label')).toBe('View mode');
  });

  it('switches from card to list view when list button is clicked', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="View Toggle Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const listButton = document.body.querySelector('[aria-label="List view"]') as HTMLButtonElement;
    expect(listButton).toBeTruthy();
    expect(listButton.getAttribute('aria-pressed')).toBe('false');

    await act(async () => {
      listButton.click();
    });

    expect(onViewModeChange).toHaveBeenCalledWith('list');
    expect(onViewModeChange).toHaveBeenCalledTimes(1);
  });

  it('switches from list to card view when card button is clicked', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="View Toggle Test"
        viewMode="list"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const cardButton = document.body.querySelector('[aria-label="Card view"]') as HTMLButtonElement;
    expect(cardButton).toBeTruthy();
    expect(cardButton.getAttribute('aria-pressed')).toBe('false');

    await act(async () => {
      cardButton.click();
    });

    expect(onViewModeChange).toHaveBeenCalledWith('card');
    expect(onViewModeChange).toHaveBeenCalledTimes(1);
  });

  it('does not call onViewModeChange when clicking already active button', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="View Toggle Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const cardButton = document.body.querySelector('[aria-label="Card view"]') as HTMLButtonElement;
    expect(cardButton.getAttribute('aria-pressed')).toBe('true');

    await act(async () => {
      cardButton.click();
    });

    expect(onViewModeChange).not.toHaveBeenCalled();
  });

  it('triggers onClose when ESC key is pressed', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="ESC Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    await act(async () => {
      document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    });

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('triggers onClose when clicking outside the dialog', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="Click Outside Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const overlay = document.body.querySelector('.modal-overlay');
    expect(overlay).toBeTruthy();

    await act(async () => {
      overlay!.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }));
    });

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('triggers onClose when close button is clicked', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="Close Button Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const closeButton = document.body.querySelector('.modal-close-floating') as HTMLButtonElement;
    expect(closeButton).toBeTruthy();

    await act(async () => {
      closeButton.click();
    });

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('applies correct aria-pressed state to view toggle buttons', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="ARIA Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const cardButton = document.body.querySelector('[aria-label="Card view"]') as HTMLButtonElement;
    const listButton = document.body.querySelector('[aria-label="List view"]') as HTMLButtonElement;

    expect(cardButton.getAttribute('aria-pressed')).toBe('true');
    expect(listButton.getAttribute('aria-pressed')).toBe('false');

    // Re-render with list mode
    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="ARIA Test"
        viewMode="list"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    expect(cardButton.getAttribute('aria-pressed')).toBe('false');
    expect(listButton.getAttribute('aria-pressed')).toBe('true');
  });

  it('renders with max-width of 1200px', async () => {
    const onClose = vi.fn();
    const onViewModeChange = vi.fn();

    await render(
      <ProviderDataDialog
        open={true}
        onClose={onClose}
        title="Width Test"
        viewMode="card"
        onViewModeChange={onViewModeChange}
      >
        <div>Content</div>
      </ProviderDataDialog>
    );

    await act(async () => {
      vi.runOnlyPendingTimers();
    });

    const dialog = document.body.querySelector('[role="dialog"]') as HTMLElement;
    expect(dialog.style.width).toBe('1200px');
  });
});
