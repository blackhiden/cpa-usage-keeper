import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Select } from '@/components/ui/Select'
import { LIVE_FLOW_PERIODS } from '../periods'
import type { LiveFlowPeriod } from '../types'

export interface LiveFlowPeriodSelectProps {
  period: LiveFlowPeriod
  onPeriodChange: (period: LiveFlowPeriod) => void
  className?: string
}

export function LiveFlowPeriodSelect({ period, onPeriodChange, className }: LiveFlowPeriodSelectProps) {
  const { t } = useTranslation()
  const options = useMemo(
    () => LIVE_FLOW_PERIODS.map((option) => ({
      value: option.value,
      label: t(option.labelKey),
      triggerLabel: t(option.triggerLabelKey),
    })),
    [t],
  )
  const currentLabel = options.find((option) => option.value === period)?.triggerLabel ?? period

  return (
    <Select
      value={period}
      options={options}
      onChange={(value) => onPeriodChange(value as LiveFlowPeriod)}
      className={className}
      ariaLabel={`${t('usage_stats.live_flow_period_label')}: ${currentLabel}`}
      dropdownMinWidth={200}
      fullWidth={false}
      dataAttributes={{ 'data-live-flow-period': period }}
    />
  )
}
