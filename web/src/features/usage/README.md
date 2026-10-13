# Live Flow

`LiveFlowPanel` is the standalone administrator view at `/live-flow` in the
Keeper dashboard. It renders the flow canvas before events arrive, then updates
nodes from the root CLIProxyAPI Live Flow WebSocket stream.

## Data path

```text
Browser
  └─ authenticated Keeper WebSocket: /api/v1/live-flow/ws
       └─ Keeper server-side CPA connection: /live-flow/ws
            └─ root CPA Live Flow hub
```

Keeper relays only the root event contract:

```ts
interface LiveFlowEvent {
  id: string
  ts: number
  event: 'started' | 'completed'
  method: string
  path: string
  model?: string
  status: number
  latency_ms: number
}
```

The stream deliberately excludes prompts, request bodies, headers, and
credentials. `CPA_MANAGEMENT_KEY` and `CPA_LIVE_FLOW_API_KEY` remain server-side.
`model` is the requested/routed model ID as CPA received it (for example,
`kr/claude-sonnet-4.5`); Keeper preserves it verbatim instead of substituting
an unrelated registry default.

## Root prerequisites

- Root CLIProxyAPI must have `cpa-fork-liveflow-enabled: true` **at startup**.
  The root route and hub are not created by configuration hot reload, so restart
  root CPA after enabling or changing this setting.
- If root CPA also enables `ws-auth: true`, configure Keeper's optional
  `CPA_LIVE_FLOW_API_KEY` with a root CPA client API key accepted by that
  WebSocket authentication layer.

## Access

Live Flow is protected by Keeper's administrator session and is intentionally
not available to API-key viewers or CPAMC embeds, because the root stream is
not scoped to one client API key.

## UI behavior

- The card/canvas is always mounted on `/live-flow`.
- Root CPA emits `started` immediately before execution and `completed` after the
  response. The canvas therefore shows long-running requests as in flight,
  rather than waiting for completion.
- Every lifecycle event produces one short directed edge pulse: amber when a
  request starts, green on success, and red on failure. These effects expire
  after the event; they do not loop continuously.
- It shows a specific waiting state before the first event.
- It reconnects after a root stream interruption and shows an unavailable state
  when the relay cannot connect. Relay failures send a safe control frame that
  distinguishes root unreachability, missing root Live Flow, and websocket-auth rejection.
- It keeps at most 80 recent metadata events in browser memory.
- The diagram starts at a fitted maximum zoom of `0.7`, approximately two
  React Flow zoom-out steps below the default viewport.
- The diagram uses the existing `@xyflow/react` topology components with nodes
  derived from observed model names.
