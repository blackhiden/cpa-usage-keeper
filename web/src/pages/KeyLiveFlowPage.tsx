import { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { AuthSessionAPIKeySummary } from '@/lib/types';
import { KeyViewerShell } from '@/features/key-viewer/KeyViewerShell';
import type { KeyViewerPath } from '@/features/key-viewer/navigation';
import { LiveFlowPanel } from '@/features/usage/LiveFlowPanel';
import styles from './KeyLiveFlowPage.module.scss';

interface KeyLiveFlowPageProps {
  apiKey?: AuthSessionAPIKeySummary;
  onNavigate: (path: KeyViewerPath) => void;
  onAuthRequired?: () => void;
}

export function KeyLiveFlowPage({ apiKey, onNavigate, onAuthRequired }: KeyLiveFlowPageProps) {
  const { t } = useTranslation();
  const [refreshKey, setRefreshKey] = useState(0);

  const handleRefresh = useCallback(() => {
    setRefreshKey((key) => key + 1);
  }, []);

  return (
    <KeyViewerShell
      activePage="liveflow"
      apiKey={apiKey}
      onNavigate={onNavigate}
      onAuthRequired={onAuthRequired}
      onRefresh={handleRefresh}
    >
      <div className={styles.pageContent}>
        <div className={styles.privacyNotice} role="alert" aria-live="polite">
          <svg
            className={styles.privacyIcon}
            width="20"
            height="20"
            viewBox="0 0 20 20"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            aria-hidden="true"
          >
            <path
              fillRule="evenodd"
              clipRule="evenodd"
              d="M18 10C18 14.4183 14.4183 18 10 18C5.58172 18 2 14.4183 2 10C2 5.58172 5.58172 2 10 2C14.4183 2 18 5.58172 18 10ZM11 6C11 6.55228 10.5523 7 10 7C9.44772 7 9 6.55228 9 6C9 5.44772 9.44772 5 10 5C10.5523 5 11 5.44772 11 6ZM9 9C9 8.44772 9.44772 8 10 8C10.5523 8 11 8.44772 11 9V14C11 14.5523 10.5523 15 10 15C9.44772 15 9 14.5523 9 14V9Z"
              fill="currentColor"
            />
          </svg>
          <div className={styles.privacyContent}>
            <strong className={styles.privacyTitle}>{t('key_viewer.live_flow_privacy_title')}</strong>
            <p className={styles.privacyText}>{t('key_viewer.live_flow_privacy_notice')}</p>
          </div>
        </div>

        <LiveFlowPanel key={refreshKey} />
      </div>
    </KeyViewerShell>
  );
}
