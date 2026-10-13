import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { liveFlowWebSocketURL } from '@/lib/api'
import { FlowTopology, type Provider } from './FlowTopology'
import { appendLiveFlowEvent, applyLiveFlowEvent, deriveLiveFlowModels, formatLiveFlowTime, liveFlowModelID, parseLiveFlowEvent, parseLiveFlowRelayError, toLiveFlowPulse, type LiveFlowActiveRequests, type LiveFlowEvent, type LiveFlowPulse, type LiveFlowRelayError } from './liveFlow'
import styles from './LiveFlowPanel.module.scss'

type ConnectionState = 'connecting' | 'connected' | 'reconnecting' | 'unavailable'

const reconnectDelayMs = 2_000

function connectionLabelKey(state: ConnectionState): string {
  return `usage_stats.live_flow_state_${state}`
}

function relayErrorMessageKey(code: string): string {
  return `usage_stats.live_flow_error_${code}`
}

export function LiveFlowPanel() {
  const { t, i18n } = useTranslation()
  const [events, setEvents] = useState<LiveFlowEvent[]>([])
  const [activeRequests, setActiveRequests] = useState<LiveFlowActiveRequests>({})
  const [pulses, setPulses] = useState<Record<string, LiveFlowPulse>>({})
  const pulseTimersRef = useRef<Record<string, number>>({})
  const [connectionState, setConnectionState] = useState<ConnectionState>('connecting')
  const [relayError, setRelayError] = useState<LiveFlowRelayError | null>(null)
  const [retryGeneration, setRetryGeneration] = useState(0)

  useEffect(() => {
    let active = true
    let socket: WebSocket | null = null
    let reconnectTimer: number | null = null
    let terminalRelayError = false
    let upstreamRelayError = false

    const connect = () => {
      if (!active) return
      terminalRelayError = false
      upstreamRelayError = false
      setConnectionState((state) => state === 'connected' ? 'reconnecting' : state)
      socket = new WebSocket(liveFlowWebSocketURL())
      socket.onopen = () => {
        if (!active) return
        setRelayError(null)
        setConnectionState('connected')
      }
      socket.onmessage = (message) => {
        try {
          const payload = JSON.parse(String(message.data))
          const error = parseLiveFlowRelayError(payload)
          if (error) {
            upstreamRelayError = true
            terminalRelayError = !error.retryable
            setRelayError(error)
            setConnectionState('unavailable')
            return
          }
          const event = parseLiveFlowEvent(payload)
          if (!event || !active) return
          const modelID = liveFlowModelID(event)
          const pulse = toLiveFlowPulse(event)
          setEvents((current) => appendLiveFlowEvent(current, event))
          setActiveRequests((current) => applyLiveFlowEvent(current, event))
          setPulses((current) => ({ ...current, [modelID]: pulse }))
          if (pulseTimersRef.current[modelID] !== undefined) {
            window.clearTimeout(pulseTimersRef.current[modelID])
          }
          pulseTimersRef.current[modelID] = window.setTimeout(() => {
            setPulses((current) => current[modelID]?.id === pulse.id
              ? Object.fromEntries(Object.entries(current).filter(([id]) => id !== modelID))
              : current)
            delete pulseTimersRef.current[modelID]
          }, 780)
        } catch {
          // Ignore malformed upstream frames; the root stream is best-effort.
        }
      }
      socket.onerror = () => active && setConnectionState('unavailable')
      socket.onclose = () => {
        if (!active || terminalRelayError) return
        setConnectionState(upstreamRelayError ? 'unavailable' : 'reconnecting')
        reconnectTimer = window.setTimeout(connect, reconnectDelayMs)
      }
    }

    connect()
    return () => {
      active = false
      if (reconnectTimer !== null) window.clearTimeout(reconnectTimer)
      Object.values(pulseTimersRef.current).forEach((timer) => window.clearTimeout(timer))
      pulseTimersRef.current = {}
      socket?.close()
    }
  }, [retryGeneration])

  const retry = () => setRetryGeneration((generation) => generation + 1)
  const unavailableMessage = relayError
    ? t(relayErrorMessageKey(relayError.code), { defaultValue: relayError.message })
    : t('usage_stats.live_flow_unavailable_description')

  const models = useMemo(() => deriveLiveFlowModels(events, activeRequests), [events, activeRequests])
  const providers = useMemo<Provider[]>(() => models.map((model, index) => ({
    id: model.id,
    name: model.name,
    requestCount: model.requestCount,
    activeRequests: model.activeRequests,
    failures: model.failures,
    latestStatus: model.latestStatus,
    latestLatencyMs: model.latestLatencyMs,
    isActive: model.activeRequests > 0,
    color: model.activeRequests > 0
      ? 'var(--warning-color)'
      : model.latestStatus >= 400
        ? 'var(--error-color)'
        : index % 2 === 0 ? 'var(--success-color)' : 'var(--primary-active)',
  })), [models])

  const latestEvent = events.at(-1)
  const activeRequestCount = Object.keys(activeRequests).length

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
        <FlowTopology providers={providers} pulses={pulses} />
        {providers.length === 0 && (
          <div className={styles.emptyState} role="status">
            <strong>{relayError || connectionState === 'unavailable' ? t('usage_stats.live_flow_unavailable_title') : t('usage_stats.live_flow_waiting_title')}</strong>
            <span>{relayError || connectionState === 'unavailable' ? unavailableMessage : t('usage_stats.live_flow_waiting_description')}</span>
            {(relayError || connectionState === 'unavailable') && <button type="button" className={styles.retryButton} onClick={retry}>{t('usage_stats.live_flow_retry')}</button>}
          </div>
        )}
      </div>

      <div className={styles.footer}>
        <div className={styles.summary}>
          <span>{t('usage_stats.live_flow_events', { count: events.filter((event) => event.event === 'completed').length })}</span>
          <span>{t('usage_stats.live_flow_inflight', { count: activeRequestCount })}</span>
          {latestEvent && <span>{t('usage_stats.live_flow_latest', { value: formatLiveFlowTime(latestEvent.ts, i18n.language) })}</span>}
        </div>
        <div className={styles.feed} aria-label={t('usage_stats.live_flow_recent_events')}>
          {events.slice(-6).reverse().map((event) => (
            <div key={`${event.id}-${event.event}-${event.ts}`} className={styles.feedItem} data-phase={event.event} data-failed={event.event === 'completed' && event.status >= 400}>
              <time dateTime={new Date(event.ts).toISOString()}>{formatLiveFlowTime(event.ts, i18n.language)}</time>
              <span className={styles.feedModel}>{event.model || t('usage_stats.live_flow_unresolved_model')}</span>
              <span>{event.event === 'started' ? t('usage_stats.live_flow_started') : `${event.method} · ${event.status}`}</span>
              <span>{event.event === 'started' ? t('usage_stats.live_flow_pending') : `${event.latency_ms}ms`}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
