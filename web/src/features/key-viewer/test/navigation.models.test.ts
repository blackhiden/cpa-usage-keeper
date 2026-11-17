import { describe, expect, it } from 'vitest';
import { isKeyViewerPath, KEY_VIEWER_PAGE_PATHS } from '../navigation';

describe('API-key viewer Models navigation', () => {
  it('allows the catalog route and maps it to its navigation item', () => {
    expect(isKeyViewerPath('/models')).toBe(true);
    expect(KEY_VIEWER_PAGE_PATHS.models).toBe('/models');
  });

  it('does not allow pricing routes', () => {
    expect(isKeyViewerPath('/pricing')).toBe(false);
  });
});