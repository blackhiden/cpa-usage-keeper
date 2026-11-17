export type KeyViewerPage = 'overview' | 'realtime' | 'analysis' | 'ranking' | 'events' | 'liveflow' | 'models';
export type KeyViewerPath = '/key-overview' | '/key-realtime' | '/key-analysis' | '/key-ranking' | '/key-events' | '/key-live-flow' | '/models';

export const KEY_VIEWER_PAGE_PATHS: Record<KeyViewerPage, KeyViewerPath> = {
  overview: '/key-overview',
  realtime: '/key-realtime',
  analysis: '/key-analysis',
  ranking: '/key-ranking',
  events: '/key-events',
  liveflow: '/key-live-flow',
  models: '/models',
};

const KEY_VIEWER_PATHS = new Set<KeyViewerPath>(Object.values(KEY_VIEWER_PAGE_PATHS));

export const isKeyViewerPath = (path: string): path is KeyViewerPath => KEY_VIEWER_PATHS.has(path as KeyViewerPath);
