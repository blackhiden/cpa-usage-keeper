export interface LiveFlowEvent {
  id: string
  ts: number
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
  failures: number
  latestStatus: number
  latestLatencyMs: number
  lastSeenAt: number
}

const maxLiveFlowEvents = 80

export function parseLiveFlowEvent(payload: unknown): LiveFlowEvent | null {
  if (!payload || typeof payload !== 'object') return null
  const value = payload as Partial<LiveFlowEvent>
  if (
    typeof value.id !== 'string'
    || typeof value.ts !== 'number'
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

export function deriveLiveFlowModels(events: readonly LiveFlowEvent[]): LiveFlowModel[] {
  const models = new Map<string, LiveFlowModel>()
  for (const event of events) {
    const id = event.model?.trim() || 'unresolved'
    const current = models.get(id)
    const next: LiveFlowModel = {
      id,
      name: id === 'unresolved' ? 'Unresolved model' : id,
      requestCount: (current?.requestCount ?? 0) + 1,
      failures: (current?.failures ?? 0) + (event.status >= 400 ? 1 : 0),
      latestStatus: event.status,
      latestLatencyMs: event.latency_ms,
      lastSeenAt: event.ts,
    }
    models.set(id, next)
  }
  return [...models.values()].sort((left, right) => right.lastSeenAt - left.lastSeenAt)
}

export function formatLiveFlowTime(timestamp: number, locale: string): string {
  return new Intl.DateTimeFormat(locale, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(new Date(timestamp))
}
