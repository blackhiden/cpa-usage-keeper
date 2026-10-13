import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { liveFlowWebSocketURL } from '@/lib/api'
import { FlowTopology, type Provider } from './FlowTopology'
import { appendLiveFlowEvent, deriveLiveFlowModels, formatLiveFlowTime, parseLiveFlowEvent, type LiveFlowEvent } from './liveFlow'
import styles from './LiveFlowPanel.module.scss'

type ConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'unavailable'

const reconnectDelayMs = 2_000

function connectionLabelKey(state: ConnectionState): string {
  return `usage_stats.live_flow_state_${state}`
}

export function LiveFlowPanel() {
  const { t, i18n } = useTranslation()
  const [events, setEvents] = useState<LiveFlowEvent[]>([])
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting')
  const [selectedModel, setSelectedModel] = useState<string>()

  useEffect(() => {
    let active = true
    let socket: WebSocket | null = null
    let reconnectTimer: number | null = null

    const connect = () => {
      if (!active) return
      setConnectionState((state) => state === 'connected' ? 'reconnecting' : 'connecting')
      socket = new WebSocket(liveFlowWebSocketURL())
      socket.onopen = () => active && setConnectionState('connected')
      socket.onmessage = (message) => {
        try {
          const event = parseLiveFlowEvent(JSON.parse(String(message.data)))
          if (!event || !active) return
          setEvents((current) => appendLiveFlowEvent(current, event))
          setSelectedModel(event.model?.trim() || 'unresolved')
        } catch {
          // Ignore malformed upstream frames; the root stream is best-effort.
        }
      }
      socket.onerror = () => active && setConnectionState('unavailable')
      socket.onclose = () => {
        if (!active) return
        setConnectionState((state) => state === 'unavailable' ? 'unavailable' : 'reconnecting')
        reconnectTimer = window.setTimeout(connect, reconnectDelayMs)
      }
    }

    connect()
    return () => {
      active = false
      if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
      socket?.close()
    }
  }, [])

  const models = useMemo(() => deriveLiveFlowModels(events), [events])
  const providers = useMemo<Provider[]>(() => models.map((model, index) => ({
    id: model.id,
    name: model.name,
    requestCount: model.requestCount,
    failures: model.failures,
    latestStatus: model.latestStatus,
    latestLatencyMs: model.latestLatencyMs,
    isActive: model.id === selectedModel,
    color: model.latestStatus >= 400 ? 'var(--error-color)' : index % 2 === 0 ? 'var(--success-color)' : 'var(--primary-active)',
  })), [models, selectedModel])

  const latestEvent = events.at(-1)

  return (
    <section className={styles.panel} aria-labelledby="live-flow-title" data-live-flow-panel>
      <header className={styles.header}>
        <div>
          <p className={styles.eyebrow}>{t('usage_stats.live_flow_eyebrow')}</p>
          <h2 id="live-flow-title" className={styles.title}>{t('usage_stats.live_flow_title')}</h2>
          <p className={styles.subtitle}>{t('usage_stats.live_flow_subtitle')}</p>
        </div>
        <div className={styles.status} data-state={connectionState} aria-live="polite">
          <span className={styles.statusDot} aria-hidden="true" />
          {t(connectionLabelKey(connectionState))}
        </div>
      </header>

      <div className={styles.canvas}>
        <FlowTopology providers={providers} activeProvider={selectedModel} />
        {providers.length === 0 && (
          <div className={styles.emptyState} role="status">
            <strong>{connectionState === 'unavailable' ? t('usage_stats.live_flow_unavailable_title') : t('usage_stats.live_flow_waiting_title')}</strong>
            <span>{connectionState === 'unavailable' ? t('usage_stats.live_flow_unavailable_description') : t('usage_stats.live_flow_waiting_description')}</span>
          </div>
        )}
      </div>

      <div className={styles.footer}>
        <div className={styles.summary}>
          <span>{t('usage_stats.live_flow_events', { count: events.length })}</span>
          {latestEvent && <span>{t('usage_stats.live_flow_latest', { value: formatLiveFlowTime(latestEvent.ts, i18n.language) })}</span>}
        </div>
        <div className={styles.feed} aria-label={t('usage_stats.live_flow_recent_events')}>
          {events.slice(-6).reverse().map((event) => (
            <div key={`${event.id}-${event.ts}`} className={styles.feedItem} data-failed={event.status >= 400}>
              <time dateTime={new Date(event.ts).toISOString()}>{formatLiveFlowTime(event.ts, i18n.language)}</time>
              <span className={styles.feedModel}>{event.model || t('usage_stats.live_flow_unresolved_model')}</span>
              <span>{event.method} · {event.status}</span>
              <span>{event.latency_ms}ms</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
