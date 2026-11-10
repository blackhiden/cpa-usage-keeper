import { useTranslation } from 'react-i18next'

interface ProviderKeysDialogProps {
  open: boolean
  onClose: () => void
  providerId: string
}

export function ProviderKeysDialog({ open, onClose, providerId }: ProviderKeysDialogProps) {
  const { t } = useTranslation()

  if (!open) return null

  return (
    <div
      className="modal-backdrop"
      onClick={onClose}
      role="dialog"
      aria-labelledby="keys-dialog-title"
      aria-modal="true"
    >
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 id="keys-dialog-title">{t('usage_stats.provider_keys_dialog_title', { provider: providerId })}</h2>
          <button type="button" onClick={onClose} aria-label="Close">×</button>
        </div>
        <div className="modal-body">
          <p>Keys dialog for {providerId} - Implementation pending</p>
        </div>
      </div>
    </div>
  )
}
