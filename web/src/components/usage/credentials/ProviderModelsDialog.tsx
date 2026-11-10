import { useTranslation } from 'react-i18next'

interface ProviderModelsDialogProps {
  open: boolean
  onClose: () => void
  providerId: string
}

export function ProviderModelsDialog({ open, onClose, providerId }: ProviderModelsDialogProps) {
  const { t } = useTranslation()

  if (!open) return null

  return (
    <div
      className="modal-backdrop"
      onClick={onClose}
      role="dialog"
      aria-labelledby="models-dialog-title"
      aria-modal="true"
    >
      <div className="modal-content" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 id="models-dialog-title">{t('usage_stats.provider_models_dialog_title', { provider: providerId })}</h2>
          <button type="button" onClick={onClose} aria-label="Close">×</button>
        </div>
        <div className="modal-body">
          <p>Models dialog for {providerId} - Implementation pending</p>
        </div>
      </div>
    </div>
  )
}
