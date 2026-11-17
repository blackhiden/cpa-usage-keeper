import { useCallback, useRef } from 'react';
import { KeyViewerShell } from '@/features/key-viewer/KeyViewerShell';
import type { KeyViewerPath } from '@/features/key-viewer/navigation';
import type { AuthSessionAPIKeySummary } from '@/lib/types';
import { ModelCatalog, type ModelCatalogHandle } from '@/components/usage/models/ModelCatalog';
import { useModelCatalogData } from '@/components/usage/models/useModelCatalogData';

interface KeyModelsPageProps {
  apiKey?: AuthSessionAPIKeySummary;
  onNavigate: (path: KeyViewerPath) => void;
  onAuthRequired?: () => void;
}

export function KeyModelsPage({ apiKey, onNavigate, onAuthRequired }: KeyModelsPageProps) {
  const catalog = useModelCatalogData({ onAuthRequired });
  const catalogRef = useRef<ModelCatalogHandle | null>(null);
  const refresh = useCallback(() => void catalogRef.current?.refresh(), []);

  return (
    <KeyViewerShell
      activePage="models"
      apiKey={apiKey}
      loading={catalog.loading && !catalog.loaded}
      onRefresh={refresh}
      refreshing={catalog.loading}
      onNavigate={onNavigate}
      onAuthRequired={onAuthRequired}
    >
      <ModelCatalog ref={catalogRef} catalog={catalog} enableTesting />
    </KeyViewerShell>
  );
}