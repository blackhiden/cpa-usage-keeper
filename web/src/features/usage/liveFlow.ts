export type LiveFlowEventPhase = 'started' | 'completed'

export interface LiveFlowEvent {
  id: string
  ts: number
  event: LiveFlowEventPhase
  method: string
  path: string
  model?: string
  status: number
  latency_ms: number
}

export interface LiveFlowModel {
  id: string
  name: string
  requestCount: number
  activeRequests: number
  failures: number
  latestStatus: number
  latestLatencyMs: number
  lastSeenAt: number
}

export interface LiveFlowPulse {
  id: string
  modelID: string
  phase: LiveFlowEventPhase
  status: number
}

export type LiveFlowActiveRequests = Record<string, string>

const maxLiveFlowEvents = 80

export interface LiveFlowRelayError {
  kind: 'keeper_live_flow_error'
  code: string
  message: string
  retryable: boolean
}

export function parseLiveFlowRelayError(payload: unknown): LiveFlowRelayError | null {
  if (!payload || typeof payload !== 'object') return null
  const value = payload as Partial<LiveFlowRelayError>
  if (
    value.kind !== 'keeper_live_flow_error'
    || typeof value.code !== 'string'
    || typeof value.message !== 'string'
    || typeof value.retryable !== 'boolean'
  ) {
    return null
  }
  return value as LiveFlowRelayError
}

export function parseLiveFlowEvent(payload: unknown): LiveFlowEvent | null {
  if (!payload || typeof payload !== 'object') return null
  const value = payload as Partial<LiveFlowEvent>
  const phase = value.event ?? 'completed'
  if (
    typeof value.id !== 'string'
    || typeof value.ts !== 'number'
    || (phase !== 'started' && phase !== 'completed')
    || typeof value.method !== 'string'
    || typeof value.path !== 'string'
    || typeof value.status !== 'number'
    || typeof value.latency_ms !== 'number'
  ) {
    return null
  }
  return {
    id: value.id,
    ts: value.ts,
    event: phase,
    method: value.method,
    path: value.path,
    model: typeof value.model === 'string' ? value.model : undefined,
    status: value.status,
    latency_ms: value.latency_ms,
  }
}

export function appendLiveFlowEvent(events: readonly LiveFlowEvent[], event: LiveFlowEvent): LiveFlowEvent[] {
  return [...events, event].slice(-maxLiveFlowEvents)
}

export function liveFlowModelID(event: Pick<LiveFlowEvent, 'model'>): string {
  return event.model?.trim() || 'unresolved'
}

export function applyLiveFlowEvent(activeRequests: LiveFlowActiveRequests, event: LiveFlowEvent): LiveFlowActiveRequests {
  if (event.event === 'started') {
    return { ...activeRequests, [event.id]: liveFlowModelID(event) }
  }
  if (!(event.id in activeRequests)) return activeRequests
  const next = { ...activeRequests }
  delete next[event.id]
  return next
}

export function toLiveFlowPulse(event: LiveFlowEvent): LiveFlowPulse {
  return {
    id: `${event.id}:${event.event}:${event.ts}`,
    modelID: liveFlowModelID(event),
    phase: event.event,
    status: event.status,
  }
}

export function deriveLiveFlowModels(events: readonly LiveFlowEvent[], activeRequests: LiveFlowActiveRequests = {}): LiveFlowModel[] {
  const models = new Map<string, LiveFlowModel>()
  for (const event of events) {
    if (event.event !== 'completed') continue
    const id = liveFlowModelID(event)
    const current = models.get(id)
    models.set(id, {
      id,
      name: id === 'unresolved' ? 'Unresolved model' : id,
      requestCount: (current?.requestCount ?? 0) + 1,
      activeRequests: 0,
      failures: (current?.failures ?? 0) + (event.status >= 400 ? 1 : 0),
      latestStatus: event.status,
      latestLatencyMs: event.latency_ms,
      lastSeenAt: event.ts,
    })
  }
  for (const modelID of Object.values(activeRequests)) {
    const current = models.get(modelID)
    models.set(modelID, {
      id: modelID,
      name: modelID === 'unresolved' ? 'Unresolved model' : modelID,
      requestCount: current?.requestCount ?? 0,
      activeRequests: (current?.activeRequests ?? 0) + 1,
      failures: current?.failures ?? 0,
      latestStatus: current?.latestStatus ?? 0,
      latestLatencyMs: current?.latestLatencyMs ?? 0,
      lastSeenAt: current?.lastSeenAt ?? 0,
    })
  }
  return [...models.values()].sort((left, right) => right.lastSeenAt - left.lastSeenAt || right.activeRequests - left.activeRequests)
}

export function formatLiveFlowTime(timestamp: number, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(timestamp))
}
