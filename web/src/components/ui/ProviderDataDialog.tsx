import { type ReactNode } from 'react';
import { Modal } from './Modal';
import { IconGrid3x3, IconList } from './icons';
import styles from './ProviderDataDialog.module.scss';

export interface ProviderDataDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  viewMode: 'card' | 'list';
  onViewModeChange: (mode: 'card' | 'list') => void;
}

export function ProviderDataDialog({
  open,
  onClose,
  title,
  children,
  viewMode,
  onViewModeChange,
}: ProviderDataDialogProps) {
  const handleCardClick = () => {
    if (viewMode !== 'card') {
      onViewModeChange('card');
    }
  };

  const handleListClick = () => {
    if (viewMode !== 'list') {
      onViewModeChange('list');
    }
  };

  const headerContent = (
    <div className={styles.header}>
      <span className={styles.title}>{title}</span>
      <div className={styles.viewToggle} role="group" aria-label="View mode">
        <button
          type="button"
          className={`${styles.toggleButton} ${viewMode === 'card' ? styles.active : ''}`}
          onClick={handleCardClick}
          aria-label="Card view"
          aria-pressed={viewMode === 'card'}
        >
          <IconGrid3x3 size={16} />
        </button>
        <button
          type="button"
          className={`${styles.toggleButton} ${viewMode === 'list' ? styles.active : ''}`}
          onClick={handleListClick}
          aria-label="List view"
          aria-pressed={viewMode === 'list'}
        >
          <IconList size={16} />
        </button>
      </div>
    </div>
  );

  return (
    <Modal open={open} onClose={onClose} title={headerContent} width={1200}>
      {children}
    </Modal>
  );
}
