# CPA Usage Keeper — Architecture

This document describes how CPA Usage Keeper is built: the runtime topology,
module layering, data flow, persistence design, frontend, and deployment
models. For *what* the system does, see [SPECIFICATION.md](SPECIFICATION.md).

---

## 1. System Context

```
                 ┌─────────────────────┐
                 │   CLIProxyAPI (CPA) │
                 │  (LLM proxy, in-    │
                 │   memory usage)     │
                 └─────────┬───────────┘
                           │ usage events (Redis pub/sub,
                           │  Redis list queue, or HTTP pull)
                           │ control messages (metadata changes)
                           ▼
┌────────┐        ┌──────────────────────────┐        ┌────────────────┐
│ Browser│◄──────►│      CPA Usage Keeper    │◄──────►│  Official       │
│ (SPA)  │  HTTP  │  Go server + SQLite      │ HTTPS  │  Ranking Center │
└────────┘        │  (this project)          │        │  keepers.cc.cd  │
                  └──────────┬───────────────┘        └────────────────┘
                             │ file backups / logs / archive
                             ▼
                        WORK_DIR (./data)
```

Usage Keeper is a single Go process embedding the web UI. It owns all of its
state in one SQLite database directory; the only hard external dependency is
CPA itself (Redis or HTTP), plus optional egress to GitHub (update checks)
and the ranking center (opt-in).

---

## 2. Module Layout (backend)

```
cmd/server/main.go ── flags (--env, --host, -v/--version), bootstrap logging
        │
        ▼
internal/app ────── application assembly (app.go) + background runners
        │            (maintenance, metadata sync, backups)
        ▼
internal/api ────── Gin router, handlers, auth middleware, DTO mapping
        │
        ├── internal/service ───── business services (usage, request logs,
        │                          identities, CPA API keys, auth files,
        │                          pricing, metadata sync, token processor)
        ├── internal/gnn ────────── Provider Model GNN feature domain (self-
        │                          contained: sanitized provider↔model graph,
        │                          GNN node/edge features + embeddings; HTTP
        │                          surface in internal/gnn/httpapi)
        ├── internal/poller ────── ingestion: subscribe/pull sources, inbox
        │                          writer, ingest/process runners, the
        │                          serial usage aggregation runner
        ├── internal/repository ── GORM data access (+ activitystore,
        │                          latencystore, overviewstore, migration),
        │                          writer/reader pool wiring (dbresolver)
        ├── internal/quota ─────── provider quota clients, refresh engine &
        │                          offline subscription resolvers (claude /
        │                          codex / antigravity plan classification)
        ├── internal/ranking ───── local and community ranking (local by default)
        ├── internal/modelranking ─ Model Ranking feature domain (self-contained:
        │                          six local-calendar periods, alias-grouped
        │                          aggregation over usage_events, seven
        │                          quantitative metrics; HTTP surface in
        │                          internal/modelranking/httpapi)
        ├── internal/modeltest ─── bounded regular/streaming CPA model probes;
        │                          HTTP registration in internal/modeltest/httpapi
        ├── internal/telegram ──── model-test failure Telegram notifications
        │                          (self-contained: contract, rendering,
        │                          Bot API client, limiter; HTTP surface in
        │                          internal/telegram/httpapi)
        ├── internal/adminsettings settings service for encrypted Admin Settings
        │                          key storage, bounded model-test timeouts, and
        │                          encrypted Telegram notification settings
        ├── internal/benchmark ─── production capacity suite (test-only tooling)
        ├── internal/pricing ───── price catalog, resolver, snapshot
        ├── internal/cpa ───────── CPA management API client (HTTP + Redis queue client)
        ├── internal/auth ──────── session management (memory/persistent; captures login IP, user-agent, and async activity touch — login_ip/last_seen_ip/user_agent/last_seen_at)
        ├── internal/enrichgeo ─── opt-in privacy-preserving IP geo/enrichment (stdlib reverse-DNS + TTL cache; private/reserved IPs classified locally)
        ├── internal/backup ────── scheduled SQLite backups
        ├── internal/updatecheck ─ GitHub release check
        ├── internal/entities ──── GORM models (single source of schema truth)
        ├── internal/config ────── env/flag loading (godotenv), TZ data
        ├── internal/logging ───── logrus setup, Gin recovery, GORM logger
        └── internal/helper, internal/timeutil ── shared helpers
```

**Layering rules**

- Handlers in `internal/api` are thin: parse/validate → call service → map to
  DTO. SQL never appears above `internal/repository`.
- Services compose repositories (composition over inheritance); repositories
  own all GORM queries for their entity family.
- `internal/entities` is the schema source of truth; migrations live under
  `internal/repository/migration`.

---

## 3. Startup Wiring Order

`main()` → `app.NewWithOptions` → `config.Load` → `NewWithConfig`. The
constructor builds components in this strict order (see
`internal/app/app.go`):

1. `logging.Configure` (logrus, file rotation, GORM logger).
2. `repository.OpenDatabasePools` — **writer + reader** SQLite pools via
   GORM dbresolver (single writer, concurrent readers).
3. `ranking.NewLocalRankingService` / `NewLocalRankingRunner` for the local
   API-key leaderboard; its five-minute refresh remains independent of the
   optional community-ranking client.
4. `UsageRecentEventCache` — failure-tolerant; falls back to DB reads.
5. `LoadPricingSnapshot` → `pricing.NewCatalog`.
6. `cpa.NewClient` — CPA management API client.
7. `quota.NewServiceWithOptions`.
8. `poller.NewUsageAggregationRunner` (serial, 3 checkpoints).
9. `service.NewSyncServiceWithOptions` (recent-events hook + notifier).
10. `MetadataSyncRunner` (Auth Files / API Keys / providers / pricing sync).
11. Ingestion sources: `RedisPullSource` / `HTTPPullSource` /
    `RedisSubscribeSource` → `ControlAwareRedisInboxWriter` →
    `RedisIngestRunner` (backoff 1s→30s) → `RedisProcessRunner`, exposed via
    the `poller.NewRedisPoller` facade.
12. Optional `DatabaseBackupRunner`.
13. Domain services: Usage, RequestLog, UsageIdentity, CPAAPIKey,
    AuthFilesManagement, Pricing, `gnn.NewService` (sanitized proxy of the
    CPA `/v0/management/config` — the DTO in
    `internal/cpa/dto/providerconfig` decodes only whitelisted fields;
    `api-key`/`base-url`/`headers` are never parsed; internally, this
    service constructs a Graph Neural Network structure for providers
    and models with node/edge features).
14. `SessionManager` (persistent GORM store when auth is enabled, else
    in-memory) → auth handler → `api.NewRouter(webui.Static, ...)`.

`Run()` starts background tasks — **RedisIngest, RedisProcess,
UsageAggregation, LocalRanking, Maintenance, MetadataSync, QuotaAutoRefresh,
BackupMaintenance** — then serves HTTP (`ListenAndServe` or
`ListenAndServeTLS`). `Close()` shuts down in strict reverse order: stop
tasks → quota stop → cache close → reader pool → writer pool → logs.

---

## 4. Data Flow — Event Ingestion & Aggregation

```
CPA ──► IngestSource (subscribe | redis pull | http pull)
           │  raw payload (at-least-once)
           ▼
     redis_usage_inboxes  ── control msgs ──► MetadataSyncRunner
           │  (source column: redis_sub/redis_pull/http_pull)
           ▼
     RedisProcessRunner ──► usage_events ──► UsageAggregationRunner (serial)
                                                │  per pipeline, in tx:
                                                │  read batch → roll up →
                                                │  advance checkpoint
                                                ▼
                       overview hourly/daily · activity · latency stats
                                                │
                                                ▼
                                       API handlers ──► React SPA
```

`LocalRankingRunner` independently refreshes local API-key day and month
snapshots every five minutes. `LocalRankingService` also creates a missing
current-day or current-month snapshot during the corresponding leaderboard
read, covering the startup and post-migration interval before the first tick.

Key invariants:

- **Durable inbox first.** Ingestion never blocks on analytics writes; a
  crash mid-pipeline loses nothing already acknowledged.
- **Single-writer discipline.** All writes go through the writer pool;
  analytics reads use the reader pool.
- **Exactly-once rollup.** `AdvanceUsageAggregationCheckpoint` does an
  optimistic-concurrency update (`WHERE name=? AND cursor=?`) **inside the
  caller's transaction** and requires `RowsAffected == 1`. Combined with the
  serial runner, every event is aggregated exactly once, even across
  restarts.
- **Warm reads.** A recent-events cache serves hot queries; on failure the
  system degrades to DB reads (availability over novelty).

---

## 5. Retention, Archive & Backups

- Daily maintenance at **04:30 local** moves `usage_events` older than **90
  local calendar days** into `usage_events_archive` (cold, permanent,
  reserved as the basis for future schema rebuilds; never queried by APIs).
- Inbox rows: kept for the current day on success, 7 days on failure.
- `backup.DatabaseBackupRunner` snapshots the SQLite file (default: enabled,
  every 24h, 7-day retention) using a dedicated store on the writer pool.
- Timezone (`TZ`, default `Asia/Shanghai`) anchors all calendar-day
  boundaries; `internal/config/tzdata.go` embeds tz data so containers work
  without system zoneinfo.

---

## 6. HTTP & Auth Architecture

- Gin router (`internal/api/router.go`) with `OptionalProviders` (quota,
  status, provider-model GNN, …) so the binary still boots when optional
  subsystems are off.
- Public edge routes (auth status/login/logout, embed transports, healthz)
  register **before** the session middleware; everything else sits behind it.
- Session transport: HttpOnly cookie, or a per-tab header token as the CPAMC
  iframe fallback. `frame-ancestors` CSP derives only from `CPA_PUBLIC_URL`.
- Persistent sessions (`auth_sessions` table) when password auth is on;
  otherwise an in-memory manager.
- Provider-group routes accept both administrators and active API-key viewers.
  The handler derives a viewer scope from the session's CPA API-key ID; the
  service resolves it to the synchronized `cpa_api_keys.api_key`, and repository
  joins restrict visible provider identities to matching `usage_events.api_group_key`
  and `auth_type='apikey'`. This keeps all SQL in the repository and prevents a
  client from supplying an authorization scope.
- Static SPA served from embedded FS (`web/static.go`: `go:embed all:dist`),
  including base-path rewriting when `APP_BASE_PATH` is set.
- Structured recovery via `internal/logging/gin_recovery.go`.

---

## 7. Frontend Architecture (`web/`)

- **React 19 + TypeScript (strict) + Vite**; no UI framework — a bespoke
  component kit in `src/components/ui` (Button, Card, Input, Select, Modal,
  PortalTooltip, LoadingSpinner, EmptyState, LanguageSwitcher,
  MainActionButton, icons).
- Styling: SCSS (dart-sass) global layers + **CSS Modules** per component;
  theming via `data-theme` (light "white" / dark / auto).
- Charts: **Chart.js 4** via `react-chartjs-2` (`src/lib/chartjs.ts`).
- **AI Provider usage:** `/ai-provider` renders `ProviderSummarySection`, which
  calls `listProviders` for one server-aggregated row per provider and
  server-side sort selection. Its Detail link uses a URL-encoded single
  provider segment. `App.tsx` routes `/ai-provider/:providerId` to
  `ProviderDetailPage`, which calls `listProviderKeys` with server-side key
  pagination and sorting. At request time, the Go static handler rewrites
  Vite's relative `./assets/` references to the configured base-root assets
  path, so direct nested provider URLs load the SPA entry assets correctly.
  Provider routes are accepted only for administrators; API-key viewer path
  normalization redirects them to their scoped dashboard.

  > **Permanence requirement:** the AI Provider table redesign and detail
  > navigation is a **core, permanent feature**. The `/ai-provider` view must
  > always render one server-aggregated row per provider (not one row per key),
  > with success-rate sorting and a Detail link to `/ai-provider/:providerId`.
  > The detail route must render that provider's paginated key list with
  > server-side sorting. Do **not** remove, rename, hide, or replace the
  > provider grouping table (`ProviderSummarySection`), the detail page
  > (`ProviderDetailPage`), or the provider API endpoints
  > (`GET /api/v1/usage/providers`, `GET /api/v1/usage/providers/:providerId/keys`)
  > without an explicit, documented architectural decision approved by the
  > maintainers and recorded in `docs/ARCHITECTURE.md`,
  > `docs/SPECIFICATION.md` (UC-7a), and `session.md`. See
  > [MERGE-PRESERVATION.md](MERGE-PRESERVATION.md) §6 for the complete policy.
- Graphs: the Usage overview tab hosts an interactive **Graph Neural Network
  (GNN)** diagram of the provider ↔ model relationship
  (`ProviderModelGNNPanel`), fed by `GET /api/provider-model-gnn`.
- **Live Flow** is a standalone Usage menu page at `/live-flow`. The
  browser opens Keeper’s authenticated `/api/v1/live-flow/ws` endpoint; Keeper
  then opens CPA’s `/live-flow/ws` stream with a server-held optional client
  key and relays only the existing metadata event contract (`id`, `ts`,
  `method`, `path`, `model`, `status`, `latency_ms`). The client never receives
  `CPA_MANAGEMENT_KEY` or `CPA_LIVE_FLOW_API_KEY`. API key viewers see a global view: they observe metadata for all requests
  across all API keys, with a privacy notice explaining the scope. No prompts,. The `model` field is the request's
  routed model ID and is preserved verbatim; Keeper does not replace it with a
  local registry default. Keeper visualizes `started` as an animated amber
  pending trail, then retains a green success or red failure trail for 25
  seconds after the matching completion. Root filters the stream before relay:
  `/chat/completions` is included, while other routes and WebSocket turns need
  prompt-bearing input; token refresh, authentication, management, and
  model-less background operations are excluded.
- **Live Flow model history** is a self-contained feature domain. The backend is
  `internal/liveflow/` (period model, history service) plus
  `internal/liveflow/httpapi/` (route registration), aggregated by
  `internal/repository/liveflow_models.go`; the frontend is
  `web/src/features/live-flow/` (types, API client, merge and trend helpers,
  period select). `LiveFlowPanel` has exactly one integration point: it renders
  the merged historical and live model set. Periods are half-open local-calendar
  windows resolved with `timeutil.NormalizeStorageTime`; weeks are ISO-style
  Monday→Sunday. History is aggregated on demand from `usage_events`
  (generation traffic only, reader pool, bounded `LIMIT`); there is no new
  table, no background runner, and `usage_events_archive` is never queried.
  Registration is `GET /api/v1/live-flow/models?period=<value>`, mounted next to
  the relay with `authHandler.roleMiddleware(auth.RoleAdmin, auth.RoleAPIKeyViewer)`.
- **Live Flow model blocks are named by model alias.** Blocks group by
  `COALESCE(NULLIF(TRIM(model_alias), ''), model)`, so several prefixed real
  model names that share one alias collapse into a single block instead of
  appearing once per prefix. When no alias exists the real model name is used.
  Each history row reports its grouped real names in `source_models`, and the
  client uses that map to fold live stream events (which carry only the routed
  model name) onto the matching alias block.
- **Live Flow canvas layout.** Blocks are centered on a ring whose radius grows
  with the model count (`liveFlowRingRadius` in `FlowTopology.tsx`) so
  neighbouring blocks keep a visible gap instead of overlapping. A block with
  in-flight requests or a pending trail is raised above idle blocks
  (`liveFlowNodeZIndex`, 10 vs 1, plus the `live-flow-node-active` class) so a
  busy block and its label are never covered by an overlapping neighbour.
- **Model-test failure Telegram notifications** are a self-contained feature
  domain. The backend is `internal/telegram/` (request/message contract, bounded
  and redacted reason rendering, a Bot API client with a fixed 10-second timeout
  and token-free errors, and a per-reporter-per-model cooldown plus hourly cap)
  plus `internal/telegram/httpapi/` for `POST /api/v1/model-test/notify`, mounted
  on the shared `catalogProtected` group so administrators and API-key viewers
  can both report. Configuration is read through a `SettingsSource`, implemented
  by the `telegramSettingsSource` adapter in `internal/app/app.go` over
  `adminsettings.TelegramConfig` (AES-GCM encrypted token, chat id, optional
  topic id), so the domain never imports GORM or the settings package. The
  reporter is always derived from the session — `helper.CPAAPIKeyDisplayName`
  for viewers, `telegram.AdminReporterLabel` for administrators — and the request
  body has no reporter field. Every failed probe is recorded in an in-memory,
  bounded `FailureStore` keyed by session scope and model (10-minute TTL), and
  the automatic notification runs on its own goroutine so the model-test
  response path is never blocked; the explicit **Notify admin** action re-sends
  the same recorded failure. Only `model` and `mode` travel from the browser,
  so all message content is server-authored. The enable toggle and per-model
  mute list from `adminsettings.TelegramConfig` are enforced before delivery,
  and `POST /api/v1/telegram/test-message` (admin-only) sends a fixed
  configuration check. `telegram.Message` and `telegram.Notifier` are
  channel-neutral, so another channel can be added without touching the
  model-test feature.

  > **Permanence requirement:** the Provider Model GNN diagram is a **core,
  > permanent feature** of the Usage overview tab. It must always remain
  > mounted on the overview tab (`UsagePage.tsx`), backed by the
  > `GET /api/provider-model-gnn` route
  > (`internal/gnn/httpapi/routes.go` →
  > `internal/gnn/gnn.go`). Do **not** remove, rename,
  > hide, or replace it (or either of its two renderers) without an explicit,
  > documented architectural decision approved by the maintainers and
  > recorded in `docs/ARCHITECTURE.md`, `docs/SPECIFICATION.md` (UC-12), and
  > `session.md`.

  > **Domain isolation:** the entire feature lives in a dedicated,
  > self-contained domain so upstream merges cannot remove or replace it:
  > backend `internal/gnn` (+ `internal/gnn/httpapi` for route registration,
  > following the `internal/ranking/httpapi` pattern) and frontend
  > `web/src/features/model-gnn/` (panel, both renderers, layout/join helper,
  > feature-local `api.ts`/`types.ts` after the `features/ranking`
  > convention). The only allowed integration points are: `gnn.NewService`
  > wiring in `internal/app/app.go`, the `OptionalProviders.ProviderModelGraph`
  > field + `gnnhttpapi.RegisterRoutes` call in `internal/api/router.go`, the
  > `@/features/model-gnn` import in `web/src/pages/UsagePage.tsx`, and the
  > `usage_stats.provider_model_graph.*` i18n keys in `src/i18n`.

  Two renderers are switchable at runtime from a header toggle:
  - **Grid (xyflow)** — `@xyflow/react` (React Flow) compact two-column grid
    layout; labels prefer the model alias, Gemini entries merge into a
    single node, oauth aliases are sorted.
  - **Network (Reagraph)** — **reagraph** (WebGL/three.js) force-directed
    2d layout (`ProviderModelReagraphCanvas` in
    `ProviderModelReagraphPanel.tsx`), lazy-loaded via a dynamic `import()`
    so the three.js bundle (~380 kB gzip) only downloads when the user
    selects it; node size encodes degree (provider model count / model
    sharing count), and edges render curved without arrows. Its WebGL canvas
    is theme-adaptive: a `Theme` built via `buildGraphTheme(isDark)` spreads
    the matching reagraph `lightTheme`/`darkTheme` preset and pins
    `canvas.background` plus node/edge label, sub-label, and edge/arrow fills
    to the exact `--bg-primary`/`--text-*`/`--border-color` hexes from
    `themes.scss`, so the background and all elements track the active
    light/dark theme instead of staying white.

  Both views reuse the same fetched `ProviderModelGraphResponse` and the
  `buildProviderModelGraph` layout/join helper, so the GNN surface stays
  identical across renderers: providers are tinted by hue from the
  `kind_hash` feature, models flagged with `is_shared=1` get a badge, and
  per-node feature vectors and embeddings appear in tooltips. The summary
  chip shows provider/model/dim counters sourced from the GNN response meta
  block. Node and edge features support future advanced analytics
  (embeddings, structural learning, prediction).
- **Model Ranking on `/ranking` (Phase 2.25)** is a self-contained feature
  domain. The backend is `internal/modelranking/` (six half-open local-calendar
  periods resolved with `timeutil.NormalizeStorageTime`, alias-grouped
  aggregation, seven
  quantitative metrics, deterministic ordering, and a 30-second in-memory read
  cache) plus `internal/modelranking/httpapi/` registering admin-only
  `GET /api/v1/model-ranking/leaderboards`. The SQL lives in
  `internal/repository/model_ranking.go` and groups by
  `COALESCE(NULLIF(TRIM(model_alias), ''), model)` on the reader pool, with a
  1,000-candidate safety bound and five-minute peak buckets. The frontend
  `web/src/features/model-ranking/` renders exactly one `ModelRankingSection`
  card with its own period and metric selectors immediately after
  `RankingPage` in `UsagePage.tsx`. There is no schema change, no background
  runner, and no archive access; the API-key ranking is untouched.
- Credentials surface **provider subscription badges**: `QuotaRow` has no
  per-row `planType` (removed upstream); instead the quota check response
  carries a top-level `subscription` (`UsageSubscriptionInfo`), which
  `credentialSubscription.ts` maps to a display plan rendered by
  `CredentialSubscriptionBadge` on auth-file credential cards. Codex
  auth-file identities may also carry `subscription` on identity responses.
- The request-events custom date range is capped at **90 days**
  (`utils/usage/customRange.ts`), and the events default page size is 50.
- The **Session Settings** card (`SessionSettingsCard.tsx`) was redesigned to
  display client metadata for each admin session. Each session item now shows
  a User-Agent block in a dedicated bordered box and a `<dl>` detail grid
  with **Login IP**, **Recent IP** (shown only when it differs from the
  login IP), **Last active**, **Login**, and **Expires** fields — laid out
  with named-area CSS Grid that degrades gracefully from desktop to mobile.
  Sessions are sorted with the current session pinned first, then by most
  recent activity (`last_seen_at` descending). Additive migration
  `20260813_add_auth_session_client_metadata` backfills `last_seen_at` for
  pre-existing sessions.
- i18n: **i18next**-backed system (`src/i18n`), locales **en + zh + zh-TW**
  (Traditional Chinese) selected via `LanguageSwitcher`.
- Structure: `assets`, `components/{ui,usage,test}`, `embed` (CPAMC iframe
  mode), `features/ranking`, `hooks`, `i18n`, `lib`, `pages`, `stores`,
  `styles`, `types`, `utils/usage`.
- Build: Vite alias `@ → src`, `base: './'`; dev proxy `/api → 127.0.0.1:8080`
  (`VITE_API_PROXY_TARGET`). Output lands in `web/dist` and is embedded into
  the Go binary.

---

## 8. Deployment Topologies

1. **Docker (recommended).** Multi-stage `Dockerfile`; data persisted via a
   volume at `WORK_DIR`. `docker-compose.example.yml` runs Keeper alone;
   README shows a combined CPA + Keeper stack.
2. **Prebuilt binaries.** GitHub `binary-release.yml` (tag `v*`): linux
   amd64/arm64, macOS amd64/arm64, Windows amd64/arm64 (MSYS2 UCRT64 /
   CLANGARM64), tar.gz/zip bundling `.env.example`, READMEs, LICENSE, and the
   systemd unit.
3. **Homebrew** (`Willxup/cpa-usage-keeper` tap), **GHCR** image.
4. **systemd** unit template (`deploy/linux/cpa-usage-keeper.service`,
   `__CPA_USAGE_KEEPER_DIR__` placeholder).
5. **From source:** `go run ./cmd/server/main.go` + `npm run dev` for the UI.

Container/start script: `docker-entrypoint.sh`; version injected via
`-ldflags -X cpa-usage-keeper/internal/version.Version=${VERSION}`.

---

## 9. Build & Verification Pipeline

- `make verify` (the baseline, also in CI `ci.yml`):
  `go test ./cmd/... ./internal/...` plus `npm --prefix ./web run
  test | lint | typecheck | build`.
- The production capacity benchmark suite (`internal/benchmark/`) is **not**
  part of the baseline. It is a Linux-only, on-demand harness (requires
  amd64 + cgroup v2/systemd); run it via
  `internal/benchmark/scripts/run-capacity.sh`. Formal measurements and the
  suite contract live in `internal/benchmark/REPORT.md` and its README.
- Release: `binary-release.yml` on `v*` tags (CGO builds per platform).

---

## 10. Extensibility Points

| Extension | Hook |
|---|---|
| New quota provider | Add a client under `internal/quota`, register in `registry.go`. |
| New ingest source | Implement the `IngestSource` interface in `internal/poller`. |
| New analytics | Add a store under `internal/repository` + extend the aggregation runner with a new checkpoint name. |
| New API surface | Thin handler in `internal/api` + service in `internal/service`. |
| Storage backends | Repository layer isolates GORM/dialect specifics (single seam for a future multi-DB port). |
| UI features | Pages under `web/src/pages`, shared widgets in `components/ui`, feature folders like `features/ranking`. |

---

## 11. Related Documents

- [SPECIFICATION.md](SPECIFICATION.md) — functional contract.
- [CONFIGURATION.md](CONFIGURATION.md) — env/flag reference.
- [ROADMAP.md](ROADMAP.md) — future phases.
- [CONTRIBUTING.md](CONTRIBUTING.md) — dev workflow.
