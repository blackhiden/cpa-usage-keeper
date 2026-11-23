
# CPA Usage Keeper — Roadmap (Compacted)

**Legend:** ⬜ not started · 🟡 in progress · ✅ done

**Policy:** Provider Model GNN Panel is a permanent, protected feature. Any changes require explicit, documented architectural decisions (see ARCHITECTURE.md §7, SPECIFICATION.md UC-12, session.md).

**Policy (Phase 2.22):** Live Flow Model History (`internal/liveflow/` + `web/src/features/live-flow/`) is a protected feature. Removal or replacement requires an explicit documented decision recorded in ARCHITECTURE.md, SPECIFICATION.md, and session.md; see MERGE-PRESERVATION.md.

**Policy (Phase 2.24, planned):** Model Test Failure Telegram Notification (`internal/telegram/` + `web/src/features/model-test/`) becomes a protected feature once implemented. Removal or replacement requires an explicit documented decision recorded in ARCHITECTURE.md, SPECIFICATION.md, and session.md; see MERGE-PRESERVATION.md.

**Policy (Phase 2.25):** Model Ranking (`internal/modelranking/` + `internal/modelranking/httpapi/`, `internal/repository/model_ranking.go` and `model_ranking_rollup.go`, `web/src/features/model-ranking/`, and mounts in `UsagePage.tsx` and `KeyRankingPage.tsx`) is a protected feature. Removal or replacement requires an explicit documented decision recorded in ARCHITECTURE.md, SPECIFICATION.md, and session.md; see MERGE-PRESERVATION.md PF-10. The existing API-key ranking (`web/src/features/ranking/`, `internal/ranking/`) must keep working unchanged.

---

## Phase 0 — Documentation & Developer Tooling
- 0.1 Project docs: ✅ Complete, kept in sync.
- 0.2 OpenAPI spec: ⬜ To be generated and served at `/api/openapi.json`.
- 0.3 Postman/HTTP collections: ⬜ To be created for main flows.
- 0.4 E2E smoke suite: ⬜ Playwright (or similar) tests for main flows in CI.
- 0.5 ADRs: ⬜ Start `docs/adr/` for major decisions.
- 0.6 Trusted-proxy tests: ⬜ Extend IP-capture tests and document poisoning-prevention.

---

## Phase 1 — Reliability & Hardening
- 1.1 Transactional write retry: ⬜ Retry on SQLITE_BUSY with backoff.
- 1.2 Backup restore tooling: ⬜ Scripted restore flow, optional UI affordance.
- 1.3 Schema rebuild from archive: ⬜ Offline rebuild from `usage_events_archive`.
- 1.4 Ingestion dead-letter surfacing: ⬜ Admin UI for failed inbox rows.
- 1.5 Health & readiness depth: ⬜ Extend `/api/healthz` with component checks.
- 1.6 Chaos/recovery test matrix: ⬜ (details truncated)

---

## Provider Grouping & Detail (Permanent Feature)
- FEAT-PROV-1.1: Backend endpoint to list providers with aggregate stats (in progress).
- FEAT-PROV-1.2: Backend endpoint to list keys by provider (complete).
- FEAT-PROV-1.3: Route registration & middleware (complete).
    no inbox loss (documented pass criteria per scenario).
- **1.7 Session pruning policy.** ⬜
  - Leverage the reliable `last_seen_at` field (including backfilled rows) to
    introduce an optional auto-revoke/cleanup policy for sessions idle past a
    configurable TTL, superseding the fixed `AUTH_SESSION_TTL` for the
    in-memory session sweep (follow-up to upstream PR #423).
- **1.8 Touch coalescing/backpressure for async activity writer.** ⬜
  - Formalize the async session-activity writer with a bounded queue,
    drop-oldest policy, and periodic flush so a burst of touches under load
    cannot grow an unbounded queue in memory (follow-up to upstream PR #423).

---

## Phase 2 — Analytics & UI/UX Enhancements

**Goal:** deeper insight and a more polished dashboard experience.

### Active Feature Development

- **2.21 Provider Keys & Models Dialogs with View Modes.** ✅ (integrated into `main`)
  - **Status:** Implemented and merged from the five worktree branches
    (`feature/phase-2-21-backend-models-api`, `-dialog-infrastructure`,
    `-keys-dialog`, `-models-dialog`, `-provider-actions`). Re-verified on
    `main` 2026-11-19: endpoint, role scoping, both dialogs, view toggle, and
    en/zh/zh-TW strings are present and tested. One documentation item
    (`MERGE-PRESERVATION.md`) remains open below.
  - **Objective:** Enhance `/ai-provider` table with separate "Keys" and "Models" action buttons. Each opens a dialog showing provider-specific data with card and list view toggles.
  - **User Story:** As an admin, I want to quickly inspect all keys or all models for a provider without navigating away from the providers table, and toggle between card and list presentations based on my preference.
  - **Scope:**
    - **Backend:**
      - New endpoint: `GET /api/v1/usage/providers/:providerId/models` (paginated, with server-side sorting)
      - Repository: `ListProviderModels(ctx, db, request)` query (aggregate distinct models from `usage_events` where `provider = :providerId`, with role-based scoping for API-key viewers)
      - Service: `ProviderModelsService.ListModels(providerId, apiGroupKey, sort, page, pageSize)`
      - Tests: Repository query, service logic, API authorization (admin vs viewer)
    - **Frontend:**
      - Rename "Detail" button to "Keys" (update i18n key `provider_detail_action` → `provider_keys_action`)
      - Add "Models" button next to "Keys"
      - Create reusable `ProviderDataDialog` component with card/list view toggle
      - Create `ProviderKeysDialog` (wraps existing key-fetching logic from `ProviderDetailPage`)
      - Create `ProviderModelsDialog` (fetches and displays provider models)
      - Card view: grid layout with model/key cards showing name, usage stats, success rate
      - List view: table layout similar to existing credential tables
      - ViewMode state (card/list) persisted in component state
      - Update i18n: add `provider_keys_action`, `provider_models_action`, `provider_keys_dialog_title`, `provider_models_dialog_title`, `view_mode_card`, `view_mode_list` (en, zh, zh-TW)
      - Tests: Dialog open/close, view toggle, data rendering, role-based display
    - **Documentation:**
      - Add feature to `docs/MERGE-PRESERVATION.md` PF-08 (permanent feature)
      - Update `docs/ARCHITECTURE.md` (new endpoint and dialog pattern)
      - Update `docs/SPECIFICATION.md` (UC-7b: Provider keys/models quick inspection)
      - Update `session.md` with implementation notes
  - **Dependencies:** None (builds on existing provider table infrastructure)
  - **Acceptance Criteria:**
    - [x] Backend endpoint returns paginated models for a provider
    - [x] Admin sees all models; API-key viewer sees only models from their events
    - [x] "Keys" button (renamed from "Detail") opens a dialog with key list
    - [x] "Models" button opens a dialog with model list
    - [x] Both dialogs support card and list view toggle
    - [x] Card view renders grid layout with stats
    - [x] List view renders table layout
    - [x] View mode selection persists during dialog session
    - [x] i18n complete for all three locales
    - [x] Tests cover dialog interaction, view toggle, and role-based filtering
    - [ ] Feature documented in `MERGE-PRESERVATION.md` — **open:** the Keys and
      Models dialogs are not yet listed as a protected feature
      (`MERGE-PRESERVATION.md` PF-06 covers `ProviderSummarySection` and
      `ProviderDetailPage` only). Recording new protection policy needs an
      explicit maintainer decision; tracked in `session.md` (2026-11-19).
  - **Tasks:** See [PHASE_2_21_PLAN.md](PHASE_2_21_PLAN.md) for detailed task breakdown (12 tasks total, 2-3 developer-days).

- **2.22 Live Flow Model History & Period Filter.** ⬜ (planned)
  - **Objective:** Keep Live Flow model nodes on the canvas after their 25-second
    trail expires and after a page refresh, for a user-selected period (Today,
    Yesterday, Current Week, Last Week, Current Month, Last Month), plus a
    period dropdown and per-model usage/trend figures.
  - **User Story:** As an admin, I want the `/live-flow` canvas to retain every
    model used during the selected period so I can see how many models I have
    used and how their usage is trending, and prepare credits accordingly.
  - **Scope:** new self-contained domain (backend `internal/liveflow/`,
    frontend `web/src/features/live-flow/`), read-only aggregation over
    `usage_events`, one new endpoint, one integration point in
    `web/src/features/usage/LiveFlowPanel.tsx`, i18n in en/zh/zh-TW,
    `docs/MERGE-PRESERVATION.md` protection entry.
  - **Dependencies:** usage-event ingestion plus the additive model-ranking
    rollup migration/checkpoint and serial aggregation pipeline. The repository
    exposes bounded resume via `BackfillModelRankingRollups(ctx, db, now,
    maxPages)`.
  - **Acceptance Criteria:**
    - [ ] A model node stays visible for the whole selected period and after a
      browser refresh.
    - [ ] The period dropdown offers exactly Today, Yesterday, Current Week,
      Last Week, Current Month, Last Month.
    - [ ] The panel shows how many distinct models were used in the period.
    - [ ] Per-model request/failure figures match the aggregated period data.
    - [ ] Live in-flight state still renders immediately and is never removed
      by the historical view.
    - [ ] Admin and API-key viewer access rules match the existing Live Flow
      relay.
    - [ ] Tests cover period boundaries, aggregation, authorization, refresh
      persistence, and i18n parity.
    - [ ] Feature documented in `docs/MERGE-PRESERVATION.md` and
      `docs/ARCHITECTURE.md`.
  - **Execution plan:** see [Phase 2.22 — Live Flow Model History](#phase-222--live-flow-model-history--period-filter-execution-plan).

- **2.23 Model Test & API-key Viewer Models Access.** ✅ (integrated into `main`)
  - **Status:** Implemented and merged; validation recorded in `session.md`.
  - **Confirmed requirements:** API-key viewers may navigate to `/models` and
    see the Models catalog only, not Model Pricing Settings; each model row
    offers a test action and a mode selector for streaming or single-response
    requests; show a small Success or Failed result, with failure reason; viewer
    tests use that viewer's CPA API key; administrator tests use a key configured
    in a new `/settings` → **Admin Settings** section, and missing admin key
    displays “Please provide Admin API Key In Setting”; Admin Settings includes
    API Key and Request Timeout fields.
  - **Implementation direction:** Isolate the feature in a dedicated backend
    domain (`internal/modeltest/` with its HTTP registration under
    `internal/modeltest/httpapi/`) and frontend feature domain
    (`web/src/features/model-test/`); limit integration points to router/app
    wiring, the existing model catalog, the viewer route/page allowlist, and
    settings-page composition. Keep the existing pricing UI and pricing API
    admin-only. Reuse the current CPA endpoint/base URL and bounded HTTP request
    pattern rather than invoking an external host supplied by the browser.
  - **Dependencies:** use the verified active-viewer-key middleware context for
    OQ-1; resolve OQ-2 (secure persistence/return semantics for the admin key),
    OQ-3 (test prompt and expected response criteria), and OQ-4 (timeout bounds)
    before finalizing request and settings contracts. Tasks can proceed on UI
    shell, route authorization tests, feature domain skeleton, and documentation
    in parallel after these boundaries are agreed; execution tasks below are
    ordered where contracts are prerequisites.
  - **Acceptance criteria:** viewer route and navigation expose only the Models
    catalog; pricing settings remain absent and inaccessible; the viewer test
    executes using only the authenticated viewer's own credential; the admin
    test uses the configured admin test key and gives the requested message if
    missing; both test modes are selectable and display success/failure; failure
    details are bounded and safe to display; settings include the key and
    request-timeout controls; all user-facing strings are translated in en,
    zh, and zh-TW; authorization, credential non-disclosure, timeout, and
    request-mode behaviors have automated coverage.
  - **Execution plan:** see [Phase 2.23 — Model Test & API-key Viewer Models
    Access](#phase-223--model-test--api-key-viewer-models-access-execution-plan).

- **2.24 Model Test Failure Telegram Notification.** ✅ (implemented)
  - **Status:** Implemented and extended (Phase 2.24) — FEAT-TG-1 through
    FEAT-TG-9 delivered, plus the accepted follow-ups REC-010 (automatic send),
    REC-011 (Send test message), REC-012 (server-side failure record), REC-013
    (enable toggle and per-model mute), and REC-014 (channel-neutral seam).
  - **Confirmed requirements:** when a model test fails, the user can notify the
    administrator through a Telegram bot. The message reports the reporter as
    the **API Key Name** (never the API key value), the tested **model**, and
    the **failure reason**. `/settings` → **Admin Settings** gains Telegram
    configuration: **Bot Token**, **Chat ID**, and **Topic ID**
    (`message_thread_id`). The feature must live in a dedicated domain and be
    documented in `docs/MERGE-PRESERVATION.md`.
  - **Implementation direction:** new backend domain `internal/telegram/` with
    HTTP registration in `internal/telegram/httpapi/` (following the
    `internal/gnn/` and `internal/liveflow/` precedent); extend
    `internal/adminsettings/` for encrypted Telegram settings; extend
    `web/src/features/model-test/` and
    `web/src/components/usage/AdminSettingsCard.tsx` on the frontend. Reuse the
    existing `app_settings` store, the existing AES-GCM protection key derived
    from `CPA_MANAGEMENT_KEY`, and the bounded outbound-HTTP pattern used by
    `internal/ranking/client.go`.
  - **Dependencies:** Phase 2.23 (implemented) supplies the model-test result
    contract (`internal/modeltest/modeltest.go` `Result`). OQ-5 (explicit vs
    automatic notification), OQ-6 (administrator reporter label), OQ-7 (rate
    limiting), and OQ-8 (failure payload depth) must be settled before the
    request contract is final.
  - **Acceptance criteria:** a failed test exposes a notify action; the
    delivered Telegram message contains the API Key Name, the model, and a
    bounded reason, and never the credential; the bot token is never returned,
    logged, or echoed; unconfigured Telegram yields a stable, non-secret error
    code; the endpoint is rate-limited per reporter; all user-facing strings are
    translated in en, zh, and zh-TW; the feature is documented in
    `docs/MERGE-PRESERVATION.md`.
  - **Execution plan:** see [Phase 2.24 — Model Test Failure Telegram
    Notification](#phase-224--model-test-failure-telegram-notification-execution-plan).

- **2.25 Model Ranking on `/ranking`.** ✅ (implemented 2026-11-19)
  - **Objective:** Add a Model leaderboard next to the existing API Key
    leaderboard on `/ranking`, ranked over a user-selected period
    (Today, Yesterday, Current Week, Last Week, Current Month, Last Month;
    default Today), grouping the same model served through different providers
    under its alias so it accumulates one combined value.
  - **User Story:** As an admin, I want to see which models are used most (and
    how reliably) during a period, with the same model from different providers
    counted together, so I can compare model demand without reading raw events.
  - **Scope:** new self-contained domain (backend `internal/modelranking/` +
    `internal/modelranking/httpapi/`, repository query
    `internal/repository/model_ranking.go`, frontend
    `web/src/features/model-ranking/`), one read-only endpoint, one mount point
    in `web/src/pages/UsagePage.tsx`, i18n in en/zh/zh-TW,
    `docs/MERGE-PRESERVATION.md` protection entry. The expansion adds an
    additive rollup schema, a checkpointed serial pipeline, and a viewer-scoped
    route on `/key-ranking`.
  - **Dependencies:** usage-event ingestion, model-ranking rollups/checkpoint,
    and the existing single-writer aggregation scheduler. OQ-MR-1 through
    OQ-MR-5 are settled by the expansion decisions below.
  - **Acceptance Criteria:**
    - [x] `/ranking` shows two distinct leaderboards: API Key ranking and
      Model ranking, each with its own period selector.
    - [x] The model selector offers exactly Today, Yesterday, Current Week,
      Last Week, Current Month, Last Month, defaulting to Today.
    - [x] Models that share an alias across providers appear once with the
      combined totals of every provider.
    - [x] Ranked values match a direct aggregation of `usage_events` for the
      same window.
    - [x] Loading, empty, stale, and error states are handled without breaking
      the API Key leaderboard.
    - [x] The global endpoint is admin-only and the viewer endpoint is strictly
      session-scoped; neither returns prompts, bodies, headers, or
      credentials, and validates its two query parameters.
    - [x] i18n keys exist in en, zh, and zh-TW.
    - [x] The feature is recorded in `docs/MERGE-PRESERVATION.md` and
      `docs/ARCHITECTURE.md`.
  - **Execution plan:** see [Phase 2.25 — Model Ranking for the Ranking
    Page](#phase-225--model-ranking-for-the-ranking-page-execution-plan).

### Phase 2.22 — Live Flow Model History & Period Filter (Execution Plan)

**Goal:** Make `/live-flow` model blocks persistent for a selected period and
add the six-period dropdown, without changing the realtime relay contract.

**Status:** ⬜ Planned — not implemented.

**Evidence used for this plan**

- `web/src/features/usage/LiveFlowPanel.tsx` keeps `events` (capped at 80 by
  `appendLiveFlowEvent`), `activeRequests`, and `trails` in React state only.
  `deriveLiveFlowModels()` builds nodes from completed events in that buffer,
  and completed trails are deleted after
  `LIVE_FLOW_COMPLETED_TRAIL_DURATION_MS` (25 s). A refresh remounts the panel
  with empty state, which is the reported "canvas resets" behavior.
- `internal/api/live_flow.go` is a pure relay: `registerLiveFlowRoutes` proxies
  CPA `/live-flow/ws` to `/api/v1/live-flow/ws` and persists nothing.
  Registration uses `authHandler.roleMiddleware(auth.RoleAdmin,
  auth.RoleAPIKeyViewer)` in `internal/api/router.go` (admin + key viewer).
- Durable per-model history already exists in `usage_events` (`model`,
  `provider`, `failed`, `timestamp`, `generate`, token columns) with indexes
  `idx_usage_events_timestamp_id` and `idx_usage_events_model`. Hot retention
  is 90 local calendar days; `usage_events_archive` is explicitly documented as
  never queried by APIs (`docs/SPECIFICATION.md` §4, §6).
- Existing aggregation precedent for "models of a provider":
  `internal/repository/provider_models.go` (`ListProviderModels` and
  `providerModelEventAggregateExpressions`) already aggregates
  `COUNT(event.id)`, `SUM(CASE WHEN event.failed ...)`, tokens, and
  `MAX(event.timestamp)` grouped by `event.model`.
- Period vocabulary already exists in ranking: `today`, `yesterday`,
  `current_month`, `previous_month` in
  `web/src/features/ranking/components/RankingToolbar.tsx` and the
  `ranking.period_*` / `ranking.period_trigger_*` i18n keys. Week periods are
  new.
- Domain-isolation precedent: `internal/gnn` + `internal/gnn/httpapi` and
  `web/src/features/model-gnn/`, registered from `internal/api/router.go` and
  protected in `docs/MERGE-PRESERVATION.md` PF-01.
- Docs drift to fix while implementing: `docs/MERGE-PRESERVATION.md` PF-04 still
  lists `web/src/pages/LiveFlowPage.tsx` and
  `web/src/components/usage/LiveFlowCanvas.tsx`, which do not exist. The real
  files are `web/src/features/usage/LiveFlowPanel.tsx`, `liveFlow.ts`,
  `FlowTopology.tsx`, and `web/src/pages/KeyLiveFlowPage.tsx`.

**Confirmed requirements (from the feature request)**

1. Model blocks must remain visible for a selectable period instead of
   disappearing ~25 s after the event or on refresh.
2. Periods: Today, Yesterday, Current Week, Last Week, Current Month, Last
   Month.
3. A dropdown filter selects the period.
4. The view communicates how many models were used and per-model trend, so the
   user can plan credits.

**Technical implementation decisions (baseline scope)**

- **Read model, not event replay.** Historical nodes come from aggregated
  `usage_events` rows grouped by model for the selected period. The live
  WebSocket continues to drive in-flight/pending state. No new table, no new
  background runner, no change to the CPA relay contract.
- **Aggregation is on demand.** The endpoint aggregates per request with a
  bounded model count; it must not scan `usage_events_archive`.
- **Filter must mirror the Live Flow stream contract.** The CPA relay only
  emits generation traffic, so the historical query must restrict to
  generation events (`generate` is NULL or true, matching
  `usageEventGenerateEnabled()` in `internal/repository/usage.go`). Without
  this the canvas would show models that never appear as live blocks.
- **Period boundaries are project-local calendar boundaries.** Use
  `timeutil.NormalizeStorageTime` (`internal/timeutil/storage.go`) and
  local-midnight arithmetic like `internal/activity/grain.go`, so Today /
  Yesterday / weeks / months match the dashboard's existing calendar-day
  semantics. Weeks are ISO-style Monday→Sunday; this must be stated in the API
  contract and covered by tests.
- **Domain isolation.** All new code lives in `internal/liveflow/` (pure
  domain + `internal/liveflow/httpapi/` route registration, following
  `internal/gnn/httpapi`) and `web/src/features/live-flow/` (panel, types,
  feature-local API client). `LiveFlowPanel` gets exactly one new integration
  point. This satisfies the request to keep the feature maintainable across
  upstream merges.
- **Access control.** Reuse the same authorization as the relay: admin and
  API-key viewer. The historical payload is metadata-only (model id, counts,
  status, latency, last-seen) and contains no prompts, bodies, headers, or
  credentials — consistent with `docs/SPECIFICATION.md` UC-2a / UC-2c.
**Task breakdown**

#### FEAT-LF-1.1 — Domain skeleton and period model

- **Layer:** backend domain.
- **Work:** create `internal/liveflow/` with `Period` (Today, Yesterday,
  CurrentWeek, PreviousWeek, CurrentMonth, PreviousMonth), a parser that
  rejects unknown values, and `ResolvePeriodWindow(period, now)` returning
  half-open `[Start, End)` local-calendar bounds. Mirror the naming used by
  `ranking.LeaderboardPeriod` for the overlapping four periods.
- **Acceptance:** table-driven tests prove Monday-start weeks, month
  boundaries, `AddDate` month arithmetic, and that `End` is exclusive.
- **Dependencies:** none.

#### FEAT-LF-1.2 — Repository aggregation query

- **Layer:** `internal/repository/liveflow_models.go`.
- **Work:** `ListLiveFlowModels(ctx, db, window)` returning, per model:
  `model`, `request_count`, `success_count`, `failure_count`,
  `total_tokens`, `last_seen_at`, `last_latency_ms`, `last_status`. Filter
  `timestamp >= ? AND timestamp < ?`, generation-only, non-empty `model`,
  ordered by `request_count DESC, model ASC` with a hard `LIMIT`.
  Read from the reader pool (`dbresolver.Read`), matching
  `provider_models.go`.
- **Acceptance:** repository tests cover window edges (inclusive start,
  exclusive end), generation filtering, empty-model exclusion, ordering, and
  the row limit.
- **Dependencies:** FEAT-LF-1.1.

#### FEAT-LF-1.3 — Domain service and DTOs

- **Layer:** `internal/liveflow/` service.
- **Work:** compose the repository into a `Service` exposing
  `Models(ctx, period)` → `{ period, period_key, start, end, timezone,
  total_models, models[] }`. `period_key` is a stable string (for example
  `2026-W46` or `2026-11`) so the UI can label the range.
- **Acceptance:** service tests assert the response shape, that `total_models`
  equals `len(models)`, and that an empty period returns an empty list rather
  than an error.
- **Dependencies:** FEAT-LF-1.1, FEAT-LF-1.2.

#### FEAT-LF-1.4 — HTTP surface and route registration

- **Layer:** `internal/liveflow/httpapi/`.
- **Work:** `GET /api/v1/live-flow/models?period=<value>` returning 400 for an
  unknown period, 503 when the provider is not configured, and a JSON body
  matching FEAT-LF-1.3. Register from `internal/api/router.go` on a group with
  `authHandler.roleMiddleware(auth.RoleAdmin, auth.RoleAPIKeyViewer)`, next to
  `registerLiveFlowRoutes`, and add the provider to `OptionalProviders`.
- **Acceptance:** API tests cover admin access, API-key viewer access,
  unauthenticated 401, invalid period 400, and default-period behavior.
- **Dependencies:** FEAT-LF-1.3.

#### FEAT-LF-1.5 — Frontend feature domain and API client

- **Layer:** `web/src/features/live-flow/`.
- **Work:** feature-local `types.ts` and `api.ts` (`fetchLiveFlowModels`),
  period constants and label keys, plus a pure merge helper
  `mergeLiveFlowModels(historical, liveModels)` that unions historical nodes
  with live in-flight models and lets live status override historical status.
- **Acceptance:** unit tests cover union, de-duplication by model id,
  live-over-historical precedence, stable ordering, and the unresolved-model
  fallback.
- **Dependencies:** FEAT-LF-1.4.



#### FEAT-LF-1.6 — Period dropdown and persistent canvas integration

- **Layer:** `web/src/features/usage/LiveFlowPanel.tsx` (+ feature domain
  component).
- **Work:** add the period `Select` (reuse `web/src/components/ui/Select.tsx`,
  as `RankingToolbar` does), fetch the selected period, persist the selection
  in `localStorage`, and render the merged model set into `FlowTopology`.
  Keep the 25-second trail expiry for *trails* while model nodes persist for
  the period. Handle loading, empty, and error states without unmounting the
  canvas.
- **Acceptance:** tests prove a node survives trail expiry, survives a
  remount/refresh with a populated response, that the dropdown renders the six
  periods, that the selected period is restored, and that an API failure keeps
  the live canvas usable.
- **Dependencies:** FEAT-LF-1.5.

#### FEAT-LF-1.7 — Usage counters and trend display

- **Layer:** frontend feature domain.
- **Work:** show the period's distinct-model count, total requests, and
  per-model trend indicator. Trend compares the selected period's request
  count with the immediately preceding period of the same kind (Today vs
  Yesterday, Current Week vs Last Week, Current Month vs Last Month) — the only
  comparison the six fixed periods make unambiguous.
- **Acceptance:** tests cover the trend computation, the "no baseline" state
  for the oldest period, and locale formatting.
- **Dependencies:** FEAT-LF-1.6.

#### FEAT-LF-1.8 — i18n (en / zh / zh-TW)

- **Work:** add `usage_stats.live_flow_period_*`, `live_flow_models_count`,
  `live_flow_trend_*`, and period-range labels to all three locales in
  `web/src/i18n/index.ts`.
- **Acceptance:** existing i18n parity check passes; no raw keys render.
- **Dependencies:** FEAT-LF-1.6, FEAT-LF-1.7.

#### FEAT-LF-1.9 — Documentation and merge protection

- **Work:** add a Live Flow history section to `docs/MERGE-PRESERVATION.md`
  (domain, endpoints, files, verification commands); update
  `docs/ARCHITECTURE.md` (module list + Live Flow subsection),
  `docs/SPECIFICATION.md` (UC-2d historical model view), `docs/CODE_GRAPH.md`
  (feature location map), `CHANGELOG.md`, `session.md`, and correct the stale
  §4 file paths.
- **Acceptance:** each document names the real paths; the merge-preservation
  verification block passes.
- **Dependencies:** FEAT-LF-1.4, FEAT-LF-1.8.

#### FEAT-LF-1.10 — Verification baseline

- **Work:** run `go test ./cmd/... ./internal/...`, then
  `npm --prefix ./web run test | lint | typecheck | build` (equivalently
  `make verify`).
- **Acceptance:** all green; manual check that `/live-flow` and
  `/key-live-flow` both keep nodes after refresh.
- **Dependencies:** all previous tasks.

**Testing strategy**

- Go table-driven tests for period windows, repository window edges, and API
  authorization.
- Vitest component tests for persistence across trail expiry and remount, plus
  pure-function tests for the merge and trend helpers.
- Regression: existing `LiveFlowPanel` tests (trail window, relay diagnostics)
  and `KeyLiveFlowPage` tests must keep passing.

**Risks and mitigations**

- *Performance:* per-request aggregation over a month of events. Mitigate with
  the model index, generation filter, bounded `LIMIT`, reader-pool routing, and
  a short-lived in-process cache if measurements show a problem (see REC-002).
- *Semantics mismatch with the live stream:* historical "used models" may
  include models the root no longer routes. Accepted; it is the requested
  "what did I use" view.
- *Retention:* periods beyond 90 days are unavailable by design; the API must
  document that month views older than hot retention are empty.
- *Authorization drift:* if the relay ever becomes admin-only, the history
  endpoint must follow. Keep both registered in the same place.
- *Duplicate nodes:* live and historical sets must merge by model id, never
  render twice.

**Open questions**

- **OQ-1:** Should API-key viewers see global historical models (matching the
  global live view) or only their own key's models? Baseline plan assumes
  global, matching UC-2c.
- **OQ-2:** Week definition — Monday→Sunday is assumed; confirm if
  Sunday→Saturday is preferred.
- **OQ-3:** Is a trend comparison required in the first release, or is a
  period total sufficient? Baseline plan includes the simple
  preceding-period comparison.

---

### Phase 2.23 — Model Test & API-key Viewer Models Access (Execution Plan)

**Goal:** Let authenticated API-key viewers browse `/models` without seeing
pricing controls, test catalog models with their own key, and let administrators
test with a separately configured key and timeout.

**Status:** ⬜ Planned — no feature code has been implemented.

**Evidence and existing boundaries**

- `/models` maps to the `models` tab in `web/src/lib/usageNavigation.ts` and is
  rendered by `web/src/pages/UsagePage.tsx`. `ModelCatalog` and
  `useModelCatalogData` live in `web/src/components/usage/models/`; the current
  tab also loads `useModelPricingData` and renders `PricingSettings`, so a viewer
  must not be allowed to reuse the full page without a restricted composition.
- `web/src/App.tsx` routes API-key viewers through an explicit route allowlist
  (`web/src/features/key-viewer/navigation.ts`) to dedicated pages and does not
  currently route them through `UsagePage`. The new `/models` viewer route must
  be explicitly added, with its model-only page composition and navigation.
- Backend `GET /api/v1/models` is registered on `adminProtected` in
  `internal/api/router.go`; `registerModelsRoutes` in `internal/api/models.go`
  calls `service.ModelsFetcher.FetchModels`. `internal/cpa/client.go`
  `FetchModels` uses a configured CPA management API key to retrieve the catalog
  then issues the catalog request using the first configured management API
  key. This is not a user-key model test and must not be mistaken for one.
- Auth middleware in `internal/api/auth.go` binds resolved role/session context
  and supports `activeAPIKeyViewerMiddleware`; viewer-owned key access must be
  resolved server-side, not accepted as an arbitrary credential from request
  data. Admin-only endpoints already use `adminMiddleware`.
- `app_settings` is represented by `internal/entities/app_setting.go` and
  accessed through `internal/repository/app_settings.go`. `config.Config.RequestTimeout`
  is the existing application-wide CPA timeout; user-editable Admin Settings
  timeout is a new scoped setting, not an environment-variable change.
- No chat-completion or model-test endpoint was found in `internal/cpa/` or
  `internal/api/`. The feature needs an explicit test request path and response
  handling, while retaining the configured CPA base URL. CPA's management key
  must never be confused with the user/admin test API key.

**Confirmed scope and behavior**

1. An API-key viewer can open `/models` and see only the Models catalog and
   model-test controls; they cannot see or call Model Pricing Settings.
2. Each model has a test control and a selector for streaming or regular
   (single-response) mode. A completed test displays Success or Failed; failure
   includes a compact reason box.
3. API-key viewer tests use that session's own CPA API key. Admin tests use the
   admin-configured test key; if it is missing, show the exact requested message:
   “Please provide Admin API Key In Setting”.
4. `/settings` gains an Admin Settings section with an API Key field and Request
   Timeout field.
5. Do not change the existing pricing experience or grant viewer access to its
   APIs. Do not return credentials in settings responses, errors, logs, or test
   responses.

**Task breakdown**

#### FEAT-MTEST-1 — Resolve API and credential contracts

- **Layer:** feature contract / security boundary.
- **Work:** use the session-bound viewer credential decision recorded under
  OQ-1; settle OQ-2 through OQ-4 before implementing test requests: decide
  whether the admin test key is persisted and how it is protected; define
  the request prompt/input, token/output bound, streaming completion criterion,
  and what counts as a successful response. Define a stable API error/result
  shape without returning an unbounded upstream body.
- **Acceptance:** a reviewed contract documents request/response fields,
  credential source, status mapping, streaming termination, response limits,
  timeout range/default, and secret redaction. No raw key is sent back to the
  browser after save or test.
- **Dependencies:** decisions OQ-2, OQ-3, OQ-4; OQ-1 is resolved as the
  session-bound active viewer key.
- **Tests:** contract cases include valid single-response/streaming success,
  upstream HTTP failure, malformed response, cancelled request, timeout, and
  oversized error/output handling.
- **Risk:** existing `CPAAPIKeyProvider` settings endpoints intentionally return
  raw values only to admin browser code; do not reuse them for key-viewer
  testing or place test keys in a generic response DTO.

#### FEAT-MTEST-2 — Create isolated backend test domain and CPA client operation

- **Layer:** backend domain and CPA integration.
- **Files:** new `internal/modeltest/` and
  `internal/modeltest/httpapi/`; extend `internal/cpa/client.go` only as needed;
  wire through `internal/app/app.go` and `internal/api/router.go`.
- **Work:** define the model-test request/response types, validate model ID and
  mode, issue requests only to the configured CPA base URL, attach the selected
  server-resolved key as a bearer credential, and apply the Admin Settings
  timeout. Support streaming without buffering an unlimited response; report
  completion only after a valid terminal result. Keep the handler thin and the
  business/integration behavior inside the feature domain.
- **Acceptance:** requests cannot select a host, path, header, or credential;
  supported modes use the contract from FEAT-MTEST-1; response size and duration
  are bounded; upstream error bodies are summarized safely; secrets are not
  logged or serialized.
- **Dependencies:** FEAT-MTEST-1.
- **Tests:** CPA client tests verify endpoint, authorization header, request
  payload, streaming and regular responses, timeout, cancellation, status errors,
  and bounded body reads; service/domain tests verify validation and status
  mapping.

#### FEAT-MTEST-3 — Add authorized model-test and catalog HTTP routes

- **Layer:** backend HTTP API and auth integration.
- **Files:** `internal/modeltest/httpapi/`, `internal/api/router.go`,
  `internal/api/auth.go` integration only if existing middleware context cannot
  supply a verified active viewer credential; add API tests under
  `internal/api/test/`.
- **Work:** expose a test operation for Admin and API-key viewer sessions using
  the existing authenticated `/api/v1` routing conventions. Make the existing
  model-catalog read available to authenticated API-key viewers without also
  exposing pricing routes; keep its metadata-only response and non-cacheable
  behavior. Admin resolves the
  configured admin test key and returns a stable configuration-required result
  when unset. Viewer requests resolve the credential belonging to the validated
  session; ignore/reject any caller-provided key. Preserve 401/403 behavior and
  do not add access to pricing endpoints.
- **Acceptance:** authenticated viewers can load the model catalog while
  pricing routes remain admin-only; admin and viewer can test under their own
  credential source;
  a viewer cannot test as another key by altering body/header/path; anonymous
  access is rejected when auth is enabled; missing admin key produces an
  explicit stable error code that the UI maps to the specified message.
- **Dependencies:** FEAT-MTEST-1, FEAT-MTEST-2.
- **Tests:** route authorization matrix for catalog access (admin/viewer),
  pricing denial (viewer), admin, own viewer, forged key,
  unauthenticated, wrong role, inactive/revoked viewer key, missing admin key,
  timeout, and upstream errors. Assert response bodies never contain raw keys
  or unbounded upstream payloads.

#### FEAT-MTEST-4 — Implement Admin Settings persistence and API

- **Layer:** settings service/repository/API.
- **Files:** `internal/entities/app_setting.go`,
  `internal/repository/app_settings.go`, service package, `internal/api/router.go`
  and settings handler; only add migration if the accepted storage decision
  needs schema changes (the generic `app_settings` table already exists).
- **Work:** persist the admin test API key and timeout as explicitly named
  settings; validate timeout bounds and required formats; provide admin-only
  read/write operations with secret-safe read semantics (presence/masked state,
  never the stored value). Keep this distinct from environment-level
  `RequestTimeout` and existing per-user CPA API-key settings.
- **Acceptance:** persisted timeout/key survive restart; settings are admin-only;
  saving a blank key follows an explicitly documented clear/retain contract;
  reads never disclose the stored key; validation failures do not alter saved
  settings; database backups retain the setting under the repository's existing
  protected-backup model.
- **Dependencies:** FEAT-MTEST-1 (especially OQ-2).
- **Tests:** repository persistence/round-trip, service validation and secret
  projection, API role checks, blank-key semantics, restart-compatible storage,
  and migration tests only if schema changes are approved.

#### FEAT-MTEST-5 — Add Admin Settings UI

- **Layer:** frontend `/settings` composition.
- **Files:** settings section composed by `web/src/pages/UsagePage.tsx` and
  relevant existing settings components; API client/types in
  `web/src/lib/api.ts` and `web/src/lib/types.ts`; component tests.
- **Work:** add a clearly named Admin Settings section with API Key input and
  Request Timeout control; mask saved-key state, provide save/loading/validation
  feedback, and ensure viewer shells never render or fetch this section.
- **Acceptance:** an administrator can save and update both values; reload shows
  whether a key exists but does not reveal it; invalid timeout is explained;
  saving does not leak the key in DOM attributes, notices, URLs, console, or
  network responses; existing Settings behavior remains unchanged.
- **Dependencies:** FEAT-MTEST-4.
- **Tests:** form rendering, secret-masked reload, save/update/clear according
  to OQ-2, timeout validation, API error states, and role composition.

#### FEAT-MTEST-6 — Add viewer `/models` route and catalog-only page

- **Layer:** frontend navigation and page composition.
- **Files:** `web/src/App.tsx`,
  `web/src/features/key-viewer/navigation.ts`,
  `web/src/features/key-viewer/KeyViewerShell.tsx`, viewer page under
  `web/src/pages/`, plus `web/src/lib/usageNavigation.ts` only if route
  resolution requires it.
- **Work:** whitelist `/models` for viewer role and add a viewer-safe page that
  renders the existing `ModelCatalog` without mounting `PricingSettings`,
  `useModelPricingData`, or pricing fetches. Reuse the shared catalog/API client
  where role authorization permits; keep admin `/models` composition intact.
- **Acceptance:** direct navigation and in-app navigation to `/models` work for
  viewers; unknown paths still normalize to viewer home; viewer page has no
  pricing section or pricing API calls; admin still sees the existing pricing
  controls; refresh and base-path deployments route correctly.
- **Dependencies:** no dependency on test execution; may proceed alongside
  backend implementation after the authorization boundary is agreed.
- **Tests:** route normalization and role-target tests, viewer shell navigation,
  catalog-only rendering, no pricing fetch/render regression, and admin page
  regression.

#### FEAT-MTEST-7 — Add per-model test controls and result states

- **Layer:** frontend model-test feature and catalog integration.
- **Files:** `web/src/features/model-test/` (API, types, test controls/result
  component), `web/src/components/usage/models/ModelCatalog.tsx` and styles,
  frontend API client only if shared API conventions require it.
- **Work:** add streaming/regular mode selector and a per-model Test action.
  Prevent duplicate concurrent tests for the same model, cancel requests when
  appropriate, and display compact Success/Failed status plus bounded failure
  reason. Map missing-admin-key response to the exact requested message. Do not
  show raw credentials or full unbounded upstream bodies.
- **Acceptance:** mode selection controls the backend mode; outcome is associated
  with the tested model; loading and cancellation are understandable; failures
  expose a useful bounded reason; long/hostile reason text cannot break layout;
  both viewer and admin use their respective backend-selected credentials.
- **Dependencies:** FEAT-MTEST-2, FEAT-MTEST-3, FEAT-MTEST-6.
- **Tests:** API client contract, mode selection, model association, loading,
  success, upstream failure, missing admin key, timeout, cancellation, duplicate
  action behavior, and bounded error display.

#### FEAT-MTEST-8 — Add translations and accessible responsive styling

- **Layer:** frontend localization/accessibility.
- **Files:** `web/src/i18n/index.ts`, model catalog/feature CSS Modules and
  relevant locale tests.
- **Work:** translate test modes, action labels, success/failure, failure reason,
  request timeout, Admin Settings, key presence/input, and exact missing-key
  message in en, zh, and zh-TW. Use accessible names/status semantics and ensure
  controls and result box work at narrow widths and in light/dark/auto themes.
- **Acceptance:** parity tests find all keys in all three locales; screen-reader
  status is announced; keyboard users can operate selector/action; no raw i18n
  key is rendered; theme and responsive tests or review criteria pass.
- **Dependencies:** FEAT-MTEST-5, FEAT-MTEST-7.

#### FEAT-MTEST-9 — Document and protect the feature domain

- **Layer:** project documentation and merge protection.
- **Files:** `docs/MERGE-PRESERVATION.md`, `docs/ARCHITECTURE.md`,
  `docs/SPECIFICATION.md`, `docs/CODE_GRAPH.md`, `docs/ROADMAP.md`,
  `README.md` / `README.zh.md` where appropriate, `CHANGELOG.md`, and
  `session.md`.
- **Work:** add the dedicated model-test domain and its approved integration
  points to merge-preservation policy; document API role/key boundaries,
  settings persistence, streaming behavior, timeout/error limits, user-visible
  viewer Models-only access, and any approved data/migration decisions. Keep
  documentation synchronized with the delivered implementation.
- **Acceptance:** preservation checks identify domain paths, route registration,
  viewer route, test controls, Admin Settings, translations, and test commands;
  all documentation describes implemented behavior rather than planned
  assumptions; README accurately reflects any user-visible capability.
- **Dependencies:** FEAT-MTEST-1 through FEAT-MTEST-8, implementation decisions.

#### FEAT-MTEST-10 — Integrate and verify the complete feature

- **Layer:** end-to-end regression and release readiness.
- **Work:** run the focused Go and frontend suites, then the project verification
  baseline `make verify`; inspect route permission boundaries, i18n parity,
  migration compatibility if applicable, redaction, and merge-preservation
  checklist. Update task status only from observed implementation evidence.
- **Acceptance:** Go tests, Vitest, ESLint, TypeScript typecheck, and Vite build
  pass via `make verify`; manual or automated browser checks confirm viewer
  Models-only navigation and both modes, admin configured-key flow, failure
  reason display, and no pricing exposure; no changes to unrelated protected
  features.
- **Dependencies:** FEAT-MTEST-1 through FEAT-MTEST-9.
- **Risks:** external CPA behavior may differ on streamed error formats or
  mid-stream termination; contract tests with representative CPA responses
  should supplement mocks before release.

**Open questions requiring resolution**

- **Resolved implementation decision — Viewer credential source:** the viewer
  session contains a CPA key identity, and `activeAPIKeyViewerMiddleware`
  resolves the active key into request context;
  `activeAPIKeyViewerContext` confirms that its ID matches the authenticated
  session. Use only this context for viewer tests (enable the middleware on the
  test route); never accept a caller-provided credential. This satisfies the
  requested "user's API key" while preserving session ownership. Any alternate
  viewer-entered-key flow would be a scope change requiring separate approval
  and threat review.
- **OQ-2 — Admin key persistence and clearing:** should the key be stored in the
  existing `app_settings` store or supplied ephemerally per test? If persisted,
  approve the at-rest protection, backup implications, replacement and clear
  behavior, and whether it is ever allowed to be read back (recommended: never
  return it; show only configured/unconfigured status). Persistence in the
  current SQLite database places it in the protected backup set.
- **OQ-3 — Test request content and success:** what prompt/input should be sent
  to the model, and does any valid completed response count as success? Define
  maximum output/token budget and how partial/empty streaming responses are
  classified. No prompt or token limit was specified.
- **OQ-4 — Timeout range/default:** the request asks for a timeout control but
  supplies no unit, default, minimum, or maximum. Set these in the API contract
  and document them before implementation; the existing global
  `config.Config.RequestTimeout` is not automatically the Admin Settings value.

---

### Phase 2.24 — Model Test Failure Telegram Notification (Execution Plan)

**Goal:** Let a user whose model test failed notify the administrator through a
Telegram bot, with a message that identifies the reporter by API Key Name, the
tested model, and the failure reason — without exposing any credential — and
configure the bot from `/settings` → **Admin Settings**.

**Status:** ✅ Implemented — FEAT-TG-1 through FEAT-TG-9 delivered (explicit
user-triggered action, encrypted settings, session-derived reporter, server-side
rate limiting). REC-010/012/013/014 remain deferred and require separate
approval.

**Evidence and existing boundaries**

- `internal/modeltest/modeltest.go` defines the current failure contract:
  `Result{Success, Mode, StatusCode, ErrorCode, Reason}`. `safeReason()` caps
  the reason at 240 characters, and `internal/cpa/client.go` `TestModel`
  discards upstream error bodies (`io.CopyN(io.Discard, …)`), so no raw
  upstream payload is currently retained anywhere. Any "failure request
  response" in the notification is therefore the bounded `Reason` plus
  `StatusCode` and `ErrorCode` unless the CPA client is changed (see OQ-8).
- `internal/modeltest/httpapi/routes.go` registers `POST /model-test` through
  `RegisterAuthenticatedRoutes`; `internal/api/router.go` mounts it on the
  `catalogProtected` group (`RoleAdmin` + `RoleAPIKeyViewer` plus
  `activeAPIKeyViewerIfPresentMiddleware`). The resolver returns
  `(credential, isViewer)` from `activeAPIKeyViewerContext`.
- `activeAPIKeyViewerContext` (`internal/api/auth.go`) returns the bound
  `auth.Session` and `entities.CPAAPIKey` only when the session's
  `CPAAPIKeyID` matches the resolved key. `helper.CPAAPIKeyDisplayName`
  (`internal/helper/redact.go`) already implements the display priority
  `KeyName → KeyAlias → masked key`, which is the correct source for the
  reporter label and never yields the raw key.
- `internal/adminsettings/settings.go` already persists settings in
  `app_settings` through `repository.GetAppSettings` / `UpsertAppSetting`,
  encrypts secrets with AES-GCM using a key derived from
  `CPA_MANAGEMENT_KEY`, and exposes only `api_key_configured` — never the
  stored value. `internal/adminsettings/httpapi/routes.go` serves admin-only
  `GET/PUT /admin-settings` (registered on `adminProtected` in
  `internal/api/router.go`).
- `web/src/components/usage/AdminSettingsCard.tsx` is the existing Admin
  Settings surface, composed in `web/src/pages/UsagePage.tsx` on the
  `settings` tab. `web/src/features/model-test/ModelTestControls.tsx` renders
  the per-model test control and is mounted by
  `web/src/components/usage/models/ModelCatalog.tsx` when `enableTesting` is
  set.
- `web/src/lib/api.ts` already exposes `fetchAdminSettings`,
  `updateAdminSettings`, and `testModel`; `web/src/lib/types.ts` holds
  `AdminSettingsResponse`, `UpdateAdminSettingsRequest`, `ModelTestRequest`,
  and `ModelTestResult`.
- No Telegram, webhook, or outbound-notification code exists in the repository
  today. `docs/ROADMAP.md` §3.1 (webhook notifications) is a broader, unstarted
  Phase 3 item; this phase is a narrower, earlier delivery and does not replace
  it.
- `internal/ranking/client.go` is the established outbound-HTTP pattern:
  dedicated `http.Client` with a fixed timeout, bounded response read, and no
  secret logging. `internal/auth/login_limiter.go` is the established
  in-process limiter pattern (mutex + per-source window map).

**Confirmed scope and behavior**

1. A failed model test offers the user an explicit action to notify the
   administrator. The action is available to API-key viewers (the primary
   reporter) and to administrators.
2. The Telegram message contains exactly: **Reporter** (API Key Name, never the
   key value), **Model**, and **Reason** (bounded failure reason). Message
   wording may be formal or casual; it must be plain text or correctly escaped
   markup.
3. `/settings` → **Admin Settings** gains Telegram configuration: **Bot Token**,
   **Chat ID**, and **Topic ID** (`message_thread_id`, optional).
4. The bot token is stored encrypted at rest, is never returned by any
   endpoint, and is never written to logs or error messages.
5. The reporter identity is resolved server-side from the authenticated
   session; the browser cannot claim another reporter.
6. When Telegram is not configured, the notify action fails with a stable,
   non-secret error code and a translated message; nothing is sent.
7. The feature lives in a dedicated domain and is recorded in
   `docs/MERGE-PRESERVATION.md`.

**Open questions (resolved 2026-11-18)**

- **OQ-5 — Explicit action vs automatic send:** both. Every failed probe
  notifies automatically (REC-010) and the explicit **Notify admin** action
  remains available.
- **OQ-6 — Administrator reporter label:** the reporter is the API Key Name for
  viewer sessions and `Administrator` for administrators.
- **OQ-7 — Abuse and volume control:** server-enforced per-reporter-per-model
  cooldown plus a per-reporter hourly cap.
- **OQ-8 — Failure payload depth:** the bounded `Reason` + `StatusCode` +
  `ErrorCode` only; raw upstream bodies are not captured.
- **OQ-9 — Topic ID semantics:** `message_thread_id` is optional and omitted
  when empty; invalid values are rejected at save time.
- **OQ-10 — Configuration verification:** Admin Settings provides a
  **Send test message** action (REC-011).

**Task breakdown**

#### FEAT-TG-1 — Resolve the notification contract and trust boundary

- **Layer:** feature contract / security boundary.
- **Work:** settle OQ-5 through OQ-10. Define the notify request and response
  DTOs, the exact message template, the reporter resolution rule per role, the
  accepted failure fields, and the stable error codes
  (`telegram_not_configured`, `telegram_send_failed`, `notify_rate_limited`,
  `invalid_notify_request`). Decide explicitly that the client-supplied
  `reason` is untrusted and is re-bounded and redacted server-side.
- **Acceptance:** a written contract lists every request/response field, every
  error code, the message template, and the trust boundary; no field can carry
  a credential.
- **Dependencies:** Phase 2.23 result contract.

#### FEAT-TG-2 — Create the isolated backend notification domain

- **Layer:** backend domain (`internal/telegram/`).
- **Files:** `internal/telegram/telegram.go` (service, DTOs, validation,
  message rendering), `internal/telegram/client.go` (Telegram Bot API sender),
  `internal/telegram/limiter.go` (per-reporter cooldown), and co-located
  `*_test.go`.
- **Work:** implement a `Notifier` interface
  (`SendMessage(ctx, Message) error`) with a Bot API implementation that posts
  to `https://api.telegram.org/bot<token>/sendMessage` using a dedicated
  `http.Client` with a fixed timeout (baseline 10 s), a bounded response read,
  and no token in any error string. Support `chat_id` and optional
  `message_thread_id`. Render a plain-text message with Reporter, Model, and
  Reason. Keep the domain free of Gin and GORM types; accept a
  `SettingsSource` interface so the service never reads the database directly.
- **Acceptance:** unit tests cover message rendering, field validation,
  timeout/transport failure mapping, non-2xx Bot API responses, and the
  guarantee that the token never appears in returned errors or logs.
- **Dependencies:** FEAT-TG-1.

#### FEAT-TG-3 — Extend Admin Settings persistence and API for Telegram

- **Layer:** settings service/repository/API (`internal/adminsettings/`).
- **Files:** `internal/adminsettings/settings.go`,
  `internal/adminsettings/httpapi/routes.go`,
  `internal/adminsettings/settings_test.go`.
- **Work:** add `telegram.bot_token` (encrypted, same AES-GCM protection key),
  `telegram.chat_id` (plain, validated), and `telegram.message_thread_id`
  (optional integer). Extend `Settings` with a nested `telegram` object that
  reports only `bot_token_configured`, `chat_id`, and `message_thread_id`.
  Extend `Update` with a nested pointer struct so omitted fields keep their
  value and an explicit empty string clears the token. Validate the token shape
  (single `:`, bounded length, no whitespace/control characters), the chat ID
  (`-?[0-9]{1,20}` or `@username`), and the topic ID (positive integer).
  Expose a `TelegramSettings(ctx)` accessor for the notification domain.
- **Acceptance:** `GET /admin-settings` never returns the token; `PUT` accepts
  partial updates; invalid values return `400` with `invalid_admin_settings`;
  clearing the token makes the feature inert; existing API-key and timeout
  behavior is unchanged; tests cover round-trip, clearing, and validation.
- **Dependencies:** FEAT-TG-1.

#### FEAT-TG-4 — Add the notify HTTP route with role-aware reporter resolution

- **Layer:** HTTP surface (`internal/telegram/httpapi/`) and router wiring.
- **Files:** `internal/telegram/httpapi/routes.go`, `internal/api/router.go`,
  `internal/app/app.go`.
- **Work:** register `POST /api/v1/model-test/notify` on the existing
  `catalogProtected` group so both roles can call it. Resolve the reporter
  server-side: for viewers use `activeAPIKeyViewerContext` +
  `helper.CPAAPIKeyDisplayName`; for administrators use the agreed label from
  OQ-6. Reject any caller-supplied reporter field. Validate and re-bound the
  model, mode, status code, error code, and reason; strip control characters
  and redact key-like substrings. Apply the per-reporter limiter before
  sending. Map domain errors to stable codes and never echo the token, chat ID,
  or Bot API response body.
- **Acceptance:** authorization matrix tests (admin, viewer, unauthenticated,
  forged reporter, cross-key attempt) pass; a viewer cannot attribute a
  notification to another key; rate limiting returns `429` with
  `notify_rate_limited`; unconfigured Telegram returns a stable non-secret
  error; response bodies never contain the token.
- **Dependencies:** FEAT-TG-2, FEAT-TG-3.

#### FEAT-TG-5 — Add the Admin Settings Telegram UI

- **Layer:** frontend settings surface.
- **Files:** `web/src/components/usage/AdminSettingsCard.tsx`,
  `AdminSettingsCard.module.scss`, `web/src/lib/api.ts`,
  `web/src/lib/types.ts`, and the co-located test.
- **Work:** add a Telegram subsection to the existing card with a Bot Token
  password input (configured placeholder, clear action), a Chat ID text input,
  and an optional Topic ID number input. Extend
  `AdminSettingsResponse`/`UpdateAdminSettingsRequest` with the nested
  `telegram` object and send only changed fields. Show inline validation for
  the chat ID and topic ID, and keep the existing save/notice/error pattern.
- **Acceptance:** the card loads and saves Telegram settings; the token is
  never rendered back; clearing works; invalid input is blocked client-side and
  still rejected server-side; the card renders correctly in light, dark, and
  auto themes and at narrow widths; component tests cover load, save, clear,
  and validation.
- **Dependencies:** FEAT-TG-3.

#### FEAT-TG-6 — Add the failure "Notify admin" action to model-test controls

- **Layer:** frontend model-test feature.
- **Files:** `web/src/features/model-test/ModelTestControls.tsx`,
  `ModelTestControls.module.scss`, `web/src/lib/api.ts`,
  `web/src/lib/types.ts`, and the co-located test.
- **Work:** after a failed result, render a secondary "Notify admin" action
  next to the failure state. Send only the model, mode, status code, error
  code, and bounded reason. Show sending, sent, and failed states; prevent
  duplicate submissions while in flight; keep the action hidden for successful
  tests. Map `telegram_not_configured` and `notify_rate_limited` to distinct
  translated messages rather than a generic failure.
- **Acceptance:** the action appears only after a failure; a successful send
  shows a confirmation and cannot be double-submitted; a rate-limited or
  unconfigured response shows the specific message; no credential is included
  in the request; component tests cover all states.
- **Dependencies:** FEAT-TG-4.

#### FEAT-TG-7 — Add translations (en / zh / zh-TW)

- **Layer:** i18n.
- **Files:** `web/src/i18n/index.ts` (all three locale blocks).
- **Work:** add `admin_settings.telegram_*` keys (section title, description,
  bot token, chat ID, topic ID, hints, validation, configured/clear states) and
  `usage_stats.model_test_notify_*` keys (action, sending, sent, failed,
  not configured, rate limited). Keep key names and structure identical across
  the three locales.
- **Acceptance:** the existing i18n parity test passes; no hard-coded
  user-facing string remains in the new components.
- **Dependencies:** FEAT-TG-5, FEAT-TG-6.

#### FEAT-TG-8 — Document and protect the feature domain

- **Layer:** documentation and merge preservation.
- **Files:** `docs/MERGE-PRESERVATION.md`, `docs/ARCHITECTURE.md`,
  `docs/SPECIFICATION.md`, `docs/CODE_GRAPH.md`, `docs/CONFIGURATION.md`
  (only if a new env var or flag is introduced — the baseline adds none),
  `README.md` / `README.zh.md`, `CHANGELOG.md`, `session.md`.
- **Work:** add a protected-feature entry to `docs/MERGE-PRESERVATION.md` with
  the domain paths (`internal/telegram/`, `internal/telegram/httpapi/`,
  `web/src/features/model-test/`,
  `web/src/components/usage/AdminSettingsCard.tsx`), the routes
  (`POST /api/v1/model-test/notify`, extended `GET/PUT /api/v1/admin-settings`),
  the security invariants, and copy-paste verification commands. Add the
  domain to the `docs/ARCHITECTURE.md` module map and extensibility table, the
  use case and API surface to `docs/SPECIFICATION.md`, the domain row to
  `docs/CODE_GRAPH.md` §8.3 (Feature Location Map), and the capability to
  `README.md` / `README.zh.md`.
- **Acceptance:** every listed document describes the implemented behavior;
  the preservation entry contains runnable verification commands; no duplicate
  documentation is created.
- **Dependencies:** FEAT-TG-1 through FEAT-TG-7.

#### FEAT-TG-9 — Verification baseline

- **Layer:** end-to-end regression and release readiness.
- **Work:** run focused suites first
  (`go test ./internal/telegram/... ./internal/adminsettings/... ./internal/api/...`,
  `npm --prefix ./web run test`), then the project baseline `make verify`.
  Perform a manual check with a real bot in a test chat: configure, save,
  trigger a failing test, notify, and confirm the message content and that the
  token never appears in the UI, logs, or API responses.
- **Acceptance:** `make verify` passes; the manual check confirms message
  content (Reporter = API Key Name, Model, Reason) and redaction; no unrelated
  protected feature regressed.
- **Dependencies:** FEAT-TG-1 through FEAT-TG-8.

**Risks and mitigations**

| Risk | Mitigation |
|---|---|
| Bot token disclosure through logs, errors, or API responses | Encrypt at rest; return only `bot_token_configured`; never include the token in error strings; add a test asserting the token is absent from responses and logs. |
| Viewer-triggered message flooding | Server-enforced per-reporter-per-model cooldown plus a per-reporter hourly cap; `429` with a stable code; UI disables the action while in flight. |
| Reporter spoofing | Reporter is derived only from `activeAPIKeyViewerContext`; any caller-supplied reporter field is ignored or rejected. |
| Sensitive content in the failure reason | Send only the bounded `Reason`/`StatusCode`/`ErrorCode`; strip control characters; redact key-like substrings; do not capture raw upstream bodies without explicit approval (OQ-8). |
| Telegram outage or slow API | Fixed client timeout, bounded response read, failure-tolerant handling that never blocks the model-test response path, and a translated failure state. |
| Merge regression of the new domain | Dedicated domain plus a `docs/MERGE-PRESERVATION.md` entry with verification commands. |
| Configuration mistakes (wrong chat ID, topic ID) | Save-time validation plus the optional "Send test message" action (REC-011). |

**Verification commands**

```bash
go build ./...
go test ./internal/telegram/... ./internal/adminsettings/... ./internal/api/...
npm --prefix ./web run typecheck
npm --prefix ./web run test
make verify
```

**Acceptance criteria (feature level)**

- [x] A failed model test exposes a notify action for viewers and
      administrators.
- [x] The delivered Telegram message contains the API Key Name, the model, and
      the bounded reason, and no credential.
- [x] The bot token is never returned, logged, or echoed; only
      `bot_token_configured` is exposed.
- [x] Admin Settings persists and clears Bot Token, Chat ID, and optional Topic
      ID with server-side validation.
- [x] Unconfigured Telegram and rate limiting produce distinct, translated,
      non-secret errors.
- [x] All new strings exist in en, zh, and zh-TW.
- [x] `docs/MERGE-PRESERVATION.md` lists the domain, routes, invariants, and
      verification commands.
- [x] `make verify` passes.

---

### Phase 2.25 — Model Ranking for the Ranking Page (Execution Plan)

**Goal:** Add a second leaderboard to `/ranking` that ranks **models** (not API
keys) over a selected period, folding the same model served through different
providers under its alias so a model used through several providers accumulates
one combined value.

**Status:** ✅ Expanded implementation merged into `main` at `93746bac`
(2026-11-19; original Phase 2.25 was `676c2bda`). The feature is protected in
`docs/MERGE-PRESERVATION.md` PF-10.

**Evidence used for this plan**

- `web/src/features/ranking/RankingPage.tsx` renders exactly one leaderboard
  card: `<article className={leaderboardCard}>` containing one period select
  (`RankingToolbar`), one metric select (`RankingMetricSelect`), and
  `RankingLeaderboardResults`. `web/src/pages/UsagePage.tsx` (around line 2171)
  mounts it for `activeTab === 'ranking'` with
  `rankingScope: RankingScope = 'local'`, `rankingPeriod` defaulting to
  `'today'`, and `rankingMetric` defaulting to `'overall'`.
  `web/src/pages/KeyRankingPage.tsx` reuses the same components for the API-key
  viewer route.
- `web/src/features/ranking/hooks/useLocalRankingData.ts` fetches
  `/ranking/local/leaderboards`, caches per `period:metric`, aborts stale
  requests, polls every `LOCAL_RANKING_POLL_INTERVAL_MS` (60 s) while the
  document is visible, and keeps the last value visible when a silent refresh
  fails.
- `internal/ranking/httpapi/local_routes.go` registers
  `GET /ranking/local/leaderboards` and `GET /key-ranking/local/leaderboards`
  with exactly two query parameters (`period`, `metric`), `Cache-Control:
  no-store`, `400 invalid_leaderboard_selection`, and
  `503 local_ranking_unavailable`. `internal/api/router.go` mounts
  `RegisterLocalRoutes` on `adminProtected` and
  `RegisterKeyViewerLocalRoutes` on `keyViewerProtected`; community ranking
  routes remain commented out.
- `internal/ranking/client.go` defines the existing vocabulary: periods
  `today`, `yesterday`, `current_month`, `previous_month` only, metrics
  `overall`, `total_tokens`, `request_count`, `cache_read_rate`,
  `ttft_average`, `latency_average`, `peak_tpm`, `peak_rpm`, and the
  `Leaderboard` / `LeaderboardEntry` JSON shape.
- `internal/ranking/local_service.go` maintains `local_ranking_period_stats`
  (entity `internal/entities/local_ranking_period_stat.go`) per **API key**
  with kinds `day` and `month` only, refreshed every five minutes by
  `LocalRankingRunner` (`internal/ranking/local_runner.go`,
  `LocalRankingAggregationInterval = 5 * time.Minute`). The table has no model
  dimension and cannot represent week periods, so it cannot back this feature.
- `internal/ranking/local_service.go` resolves its windows in a hardcoded
  `time.FixedZone("Asia/Shanghai", 8*60*60)` (`localRankingPeriodLocation`),
  while `internal/config/config.go` assigns `time.Local` from `TZ` and
  `internal/liveflow/period.go` resolves periods with
  `timeutil.NormalizeStorageTime`. The two calendar definitions disagree for a
  deployment whose `TZ` is not Asia/Shanghai; recorded as OQ-MR-4 instead of
  silently changing the existing key ranking.
- Alias grouping already exists in production code:
  `internal/repository/liveflow_models.go` groups with
  `COALESCE(NULLIF(TRIM(event.model_alias), ''), event.model)`, bounded by
  `liveFlowModelLimit = 60`, reads through `dbresolver.Read`, and filters to
  generation traffic (`event.generate IS NULL OR event.generate = true`).
  `internal/repository/provider_models.go` aggregates per provider through
  `providerModelEventAggregateExpressions()`.
- `internal/entities/usage_event.go` carries `Model`, `ModelAlias *string`,
  `Provider`, `Failed`, `Timestamp`, `InputTokens`, `CacheReadTokens`,
  `TotalTokens`, `TTFTMS`, `LatencyMS`, and `Generate`, with indexes
  `idx_usage_events_timestamp_id` (timestamp, id) and `idx_usage_events_model`
  (model). There is no index on `model_alias`.
- Five-minute peak buckets have an established SQL shape in
  `LocalRankingService.aggregateLocalRankingWindow`
  (`CAST(strftime('%s', timestamp) AS INTEGER) / 300 AS bucket_key`, then
  `MAX(request_count)` / `MAX(total_tokens)`), and
  `rankingTimeRangePredicate` handles the stored-text versus local-offset
  comparison needed for correct window edges.
- Domain-isolation precedent: `internal/gnn` + `internal/gnn/httpapi`,
  `internal/liveflow` + `internal/liveflow/httpapi`, and
  `web/src/features/live-flow/`, each registered from
  `internal/api/router.go` and recorded in `docs/MERGE-PRESERVATION.md`.
- Documentation state: `docs/SPECIFICATION.md` UC-9 covers local API-key
  ranking only; `docs/ARCHITECTURE.md` §7 describes `features/ranking`;
  `docs/MERGE-PRESERVATION.md` protects "Local Ranking Default" with no model
  ranking entry; this roadmap has no model-ranking task before this phase.

**Confirmed requirements (from the feature request)**

1. `/ranking` must show a **Model Ranking** in addition to the existing API Key
   ranking, in its own card or section on the same page.
2. The model ranking is period-based, with exactly six selectable periods:
   Today, Yesterday, Current Week, Last Week, Current Month, Last Month.
   **Today is the default.**
3. Models are identified by **alias**: the same model served through different
   providers, under different model names but with the same alias, must rank as
   one entry so the combined usage produces a higher value.
4. The feature must live in its own domain and be documented in
   `docs/MERGE-PRESERVATION.md` so an upstream merge cannot remove it.

**Technical implementation decisions (baseline scope)**

- **New self-contained domain.** Backend `internal/modelranking/` with route
  registration in `internal/modelranking/httpapi/`; frontend
  `web/src/features/model-ranking/` (types, API client, data hook, section
  component, styles). There are four integration points:
  `internal/app/app.go` (construct the service and populate `api.OptionalProviders`)
  plus the `OptionalProviders.ModelRanking` field and `RegisterRoutes` call in
  `internal/api/router.go` (on `adminProtected`), the repository query file
  described below, and `web/src/pages/UsagePage.tsx` (one mount immediately
  after `<RankingPage … />` inside the existing `activeTab === 'ranking'`
  block). `web/src/features/ranking/` and `internal/ranking/` are not modified,
  so the API-key leaderboard keeps its behavior and its merge surface.
- **Rollup-backed read model.** The normal read path uses the additive
  `model_ranking_period_stats` rollup maintained by the serial usage
  aggregation runner and `model_ranking_aggregation_checkpoints`. Raw
  `usage_events` remains the source for bounded backfill/rebuild; online ranking
  never queries `usage_events_archive`.
- **SQL stays in `internal/repository`.** Add
  `internal/repository/model_ranking.go` plus a DTO in
  `internal/repository/dto/model_ranking.go`, matching
  `internal/repository/liveflow_models.go`. The domain service owns only
  period resolution, value projection, ordering, and response shaping. The
  repository file is an explicit, documented integration point (it is listed
  in the protection entry) because the layering rule puts all SQL there.
- **Grouping key.** Use exactly
  `COALESCE(NULLIF(TRIM(event.model_alias), ''), event.model)`: the alias wins
  when present, otherwise the real model name. Several real model names that
  share one alias collapse into a single row, which is the requested
  cross-provider accumulation. Rows also report the distinct
  `event.provider` values that contributed (`providers[]`, bounded), so the UI
  can show provenance without splitting the ranking.
- **Metric vocabulary.** The domain defines its own metric type with the same
  wire strings as `ranking.LeaderboardMetric`, and a unit test asserts string
  parity. This keeps the domain import-free of `internal/ranking` while
  preventing silent drift. The expansion exposes the seven quantitative
  metrics plus a model-population `overall` score (`total_tokens` remains the
  default).
- **Period resolution.** `internal/modelranking/period.go` resolves the six
  periods as half-open `[Start, End)` local-calendar windows using
  `timeutil.NormalizeStorageTime`, ISO Monday→Sunday weeks, `AddDate` month
  arithmetic, and stable keys (`2006-01-02`, `2026-W46`, `2006-01`). This
  mirrors `internal/liveflow/period.go` and uses the project `TZ` rather than
  the hardcoded Asia/Shanghai zone of the API-key ranking (OQ-MR-4).
- **API contract.**
  `GET /api/v1/model-ranking/leaderboards?period=<value>&metric=<value>`
  accepts exactly those two parameters and returns
  `{period, period_key, metric, start, end, timezone, generated_at, entries[]}`
  where each entry is
  `{rank, model, providers[], request_count, success_count, failure_count,
  input_tokens, cache_read_tokens, total_tokens, value, rate_numerator?,
  rate_denominator?, metrics?}`. Unknown or extra parameters →
  `400 invalid_model_ranking_selection`; unconfigured provider →
  `503 model_ranking_unavailable`; every response carries
  `Cache-Control: no-store`.
- **Metric semantics parity with the existing leaderboard.**
  `value`, `rate_numerator`, and `rate_denominator` follow
  `internal/ranking/local_leaderboard.go`: tokens and requests sort
  descending; cache read rate is `cache_read_tokens / input_tokens` scaled by
  1e6; TTFT and latency are scaled mean milliseconds that sort ascending
  (lower is better) and require at least one successful sample with
  `ttft_ms > 0` and `latency_ms > 0`; peak TPM/RPM use the largest five-minute
  token/request bucket. Models with no eligible sample for the selected metric
  are omitted.
- **Bounds.** At most `modelRankingTopLimit = 100` displayed entries, matching
  `localRankingTopLimit`; the repository caps candidate rows at
  `modelRankingCandidateLimit = 1000` before ranking. Ties break by model name
  ascending so the order is deterministic.
- **Read path and caching.** Aggregate through `dbresolver.Read`. A process
  cache keyed by `period kind + period key + metric` with a 30-second TTL
  (the same `leaderboardReadCacheTTL` used by `internal/ranking/cache.go`)
  prevents bursty polls and rapid period toggling from rescanning the window.
  The cache is in-memory only; a restart simply recomputes, and nothing new is
  persisted.
- **Authorization.** Admin only in the baseline (`adminProtected`). Model
  ranking is global across all API keys, so exposing it to API-key viewers
  would disclose other users' usage patterns; a viewer-scoped variant is
  OQ-MR-3.
- **Frontend shape.** `ModelRankingSection` renders its own card with its own
  period select (six options, default Today, persisted under a dedicated
  `localStorage` key) and metric select (seven options, default
  `total_tokens`). The two leaderboards have independent selectors: changing
  the model period must not reload or reset the API-key leaderboard. Loading,
  empty, error, and stale states mirror `RankingPage`, and the 60-second poll
  stops while the document is hidden.
- **i18n namespace.** New keys live in a dedicated `model_ranking` namespace in
  `web/src/i18n/index.ts` for en, zh, and zh-TW rather than extending the
  existing `ranking` block, which keeps merge conflicts confined to the new
  namespace.

**Task breakdown**

#### FEAT-MR-1.1 — Domain skeleton and period model

- **Layer:** `internal/modelranking/` (`period.go`, `period_test.go`).
- **Work:** define `Period` with the six values (`today`, `yesterday`,
  `current_week`, `previous_week`, `current_month`, `previous_month`),
  `AllPeriods()` in display order, `ParsePeriod` rejecting unknown values, and
  `ResolveWindow(period, now)` returning half-open local-calendar bounds with a
  stable `Key`. Weeks start Monday; months use `AddDate(0, 1, 0)` so short
  months and DST transitions stay correct. Use
  `timeutil.NormalizeStorageTime(now)` for the local clock.
- **Tests:** table-driven coverage for each period, a Sunday→Monday week edge,
  a month edge (31st → 1st), a `TZ` other than Asia/Shanghai, and proof that
  `End` is exclusive.
- **Acceptance:** `ResolveWindow` never returns an empty or inverted range; an
  unknown period returns an error; period keys are stable strings.
- **Dependencies:** none.

#### FEAT-MR-1.2 — Repository aggregation query

- **Layer:** `internal/repository/model_ranking.go`,
  `internal/repository/dto/model_ranking.go`, plus tests under
  `internal/repository/test/`.
- **Work:** implement
  `ListModelRankingRows(ctx, db, window)` returning per display-name totals
  (`request_count`, `success_count`, `failure_count`, `input_tokens`,
  `cache_read_tokens`, `total_tokens`, `ttft_sum_ms`, `ttft_sample_count`,
  `latency_sum_ms`, `latency_sample_count`, `peak_5m_request_count`,
  `peak_5m_total_tokens`) and the contributing providers.
  1. One bucketed statement built like
     `LocalRankingService.aggregateLocalRankingWindow`: a `ranged_events` CTE
     with the window predicate, a `five_minute_buckets` CTE grouped by
     display name and `CAST(strftime('%s', timestamp) AS INTEGER) / 300`, and a
     final `GROUP BY` on display name computing every sum plus
     `MAX(request_count)` / `MAX(total_tokens)`. No metric-specific
     pre-ordering or request-count limit is applied, because limiting
     candidates by popularity would bias the TTFT, latency, and cache
     rankings; the result is ordered by display name for determinism and
     capped at `modelRankingCandidateLimit = 1000` rows as a safety bound.
  2. A providers query grouped by display name and `event.provider`, folded in
     Go into `providers[]` per row (trimmed, de-duplicated, capped at a small
     number such as 10, sorted).

  Both queries read through `dbresolver.Read` and use the coarse-text plus
  epoch predicate shape of `rankingTimeRangePredicate` (its helper is
  unexported in `internal/ranking`; document the intentional duplicate and
  reference the precedent in a comment). Skip rows with an empty display name.
- **Tests:** window edges (inclusive start, exclusive end), alias collapsing
  across two `provider` values, real-name fallback when `model_alias` is
  NULL/blank, empty window returning an empty slice, the candidate safety cap,
  and peak bucket selection.
- **Acceptance:** a fixture with two providers and one shared alias yields one
  row with summed counters and both providers; the candidate row set returned by
  the repository is identical for every metric, so a metric switch cannot change
  which models are considered; no query touches `usage_events_archive`.
- **Dependencies:** FEAT-MR-1.1.
- **Dependencies:** FEAT-MR-1.1.

#### FEAT-MR-1.3 — Domain service, metric projection, and ordering

- **Layer:** `internal/modelranking/` (`service.go`, `metric.go`,
  `service_test.go`).
- **Work:** define the metric type with the seven wire values, validate it, and
  project each row into a `LeaderboardEntry`-equivalent value plus
  `rate_numerator` / `rate_denominator`. Sort descending for tokens, requests,
  and peak metrics; ascending for TTFT and latency; filter ineligible rows;
  truncate to `modelRankingTopLimit`; assign `rank` starting at 1; break ties
  by model name ascending. Build the response with `period`, `period_key`,
  `metric`, `start`, `end`, `timezone` (`time.Local.String()`),
  `generated_at` (UTC), and `entries`. Add the 30-second in-memory cache with a
  mutex, keyed by kind/key/metric.
- **Tests:** metric parity against the formulas in
  `internal/ranking/local_leaderboard.go` (fixed rows with known expected
  values), ascending versus descending order, ineligible rows omitted,
  deterministic ties, top-100 truncation, empty window, and cache hit versus
  miss behavior.
- **Acceptance:** identical fixtures produce identical `value` output to the
  API-key leaderboard formulas; `generated_at` is UTC; an empty period returns
  an empty entry list, not an error.
- **Dependencies:** FEAT-MR-1.2.

#### FEAT-MR-1.4 — HTTP surface, app wiring, and route registration

- **Layer:** `internal/modelranking/httpapi/routes.go`, `internal/app/app.go`,
  `internal/api/router.go`, plus `internal/api/test/`.
- **Work:** register
  `GET /model-ranking/leaderboards` on `adminProtected` through a
  `RegisterRoutes(router gin.IRoutes, provider *modelranking.Service)` call,
  following `modeltest/httpapi` and `gnn/httpapi`. Reject anything other than
  exactly `period` and `metric` (`400 invalid_model_ranking_selection`), return
  `503 model_ranking_unavailable` when the provider is nil, and set
  `Cache-Control: no-store`. Add the `ModelRanking` field to
  `OptionalProviders` in `internal/api/router.go` and construct the service in
  `internal/app/app.go` next to `ranking.NewLocalRankingService`
  (the service needs no runner and must not block startup).
- **Tests:** `internal/api/test/model_ranking_routes_test.go` covering admin
  200, viewer 403, missing/extra parameter 400, unknown period/metric 400, nil
  provider 503, and the response headers.
- **Acceptance:** `internal/api/router.go` comment block documents the single
  registration; no existing route path or group changes.
- **Dependencies:** FEAT-MR-1.3.

#### FEAT-MR-1.5 — Frontend domain: types, API client, data hook

- **Layer:** `web/src/features/model-ranking/` (`types.ts`, `api.ts`,
  `hooks/useModelRankingData.ts`, `period.ts`, plus tests under
  `web/src/features/model-ranking/test/`).
- **Work:** feature-local `types.ts` with the six periods, seven metrics, and
  the response contract from FEAT-MR-1.4 (no imports from
  `@/features/ranking`). `api.ts` reuses `apiFetch`/`apiPath` from
  `@/lib/api`, sets `credentials: 'include'` and `cache: 'no-store'`, and maps
  failures to a stable `ModelRankingApiError` carrying `status` and `code`,
  following `features/ranking/api.ts`. The hook mirrors
  `useLocalRankingData`: abort stale requests, keep a per-`period:metric`
  cache, show the cached board while refreshing, mark it `stale` when a
  background refresh fails, poll every 60 s only while the document is
  visible, and expose `refresh`.
- **Tests:** request URL/parameters, abort on selection change, cached board
  reuse, stale flag on failed silent refresh, error propagation, and
  `localStorage` persistence of the selected period (invalid stored values
  fall back to `today`).
- **Acceptance:** no network call is issued when the section is not mounted;
  an API failure leaves the last successful board rendered with a stale badge.
- **Dependencies:** FEAT-MR-1.4 (contract only; can be developed in parallel
  against the documented JSON shape).

#### FEAT-MR-1.6 — Model ranking section and page integration

- **Layer:** `web/src/features/model-ranking/ModelRankingSection.tsx`,
  `ModelRankingSection.module.scss`, and one mount in
  `web/src/pages/UsagePage.tsx`.
- **Work:** render a second `card` with a title, its own period `Select`
  (reuse `web/src/components/ui/Select.tsx` as `RankingToolbar` does), its own
  metric `Select`, and a table (rank, model, providers, metric value) plus a
  three-entry podium only if it fits the card density; otherwise table-only.
  Show `generated_at`, a stale badge, a loading state, an empty state
  ("no model usage in this period"), and an error state with a retry action.
  Mount it inside `activeTab === 'ranking'` immediately after
  `<RankingPage ... />`, without passing props from `RankingPage`. Styles use
  the global `card` class and the existing title classes, with the feature's
  own CSS module for the rest.
- **Tests:** `web/src/features/model-ranking/test/ModelRankingSection.test.tsx`
  covering six period options, default `today`, alias-collapsed row rendering
  with two provider badges, metric switching, empty and error states, and that
  changing the model period does not call the API-key leaderboard endpoint.
  Add a `UsagePage` composition test asserting both cards render on the
  ranking tab.
- **Acceptance:** both leaderboards render on `/ranking`; the model card looks
  correct in light, dark, and auto themes and stacks without horizontal
  overflow at mobile widths.
- **Dependencies:** FEAT-MR-1.5.

#### FEAT-MR-1.7 — i18n for en, zh, and zh-TW

- **Layer:** `web/src/i18n/index.ts`, `web/src/i18n/test/`.
- **Work:** add a `model_ranking` namespace: card title, period labels and
  trigger labels for the six periods, metric labels and short labels for the
  seven metrics, table headers, provider label, updated-at, stale, loading,
  empty, error, and retry strings.
- **Tests:** extend the existing translation-parity pattern
  (`web/src/i18n/test/rankingTranslations.test.ts`) with a
  `modelRankingTranslations.test.ts` that asserts the same key set in all
  three locales and no empty values.
- **Acceptance:** no raw translation key can render in the new card; all three
  locales carry identical key sets.
- **Dependencies:** FEAT-MR-1.6.

#### FEAT-MR-1.8 — Documentation and merge-preservation record

- **Layer:** `docs/MERGE-PRESERVATION.md`, `docs/ARCHITECTURE.md`,
  `docs/SPECIFICATION.md`, `CHANGELOG.md`, `session.md`, and (if the capability
  is user-visible enough to warrant it) `README.md` / `README.zh.md`.
- **Work:** add a numbered protection entry to `MERGE-PRESERVATION.md`
  covering the domain paths, the repository query file, the route, the mount
  point in `UsagePage.tsx`, the i18n namespace, and post-merge verification
  commands. Add `internal/modelranking` (+ `httpapi/`) to the
  `ARCHITECTURE.md` §2 module map and a §7 note describing the domain, its
  single mount point, and its read-only aggregation. Extend
  `SPECIFICATION.md` UC-9 (or add a sibling use case) with the six periods,
  alias grouping, the seven metrics, and the admin-only boundary. Record the
  delivered behavior in `CHANGELOG.md` and the implementation notes in
  `session.md`.
- **Acceptance:** the checklist grep commands in the new protection entry pass
  against the implemented files; no documentation claims behavior the code
  does not have; the API-key ranking documentation is unchanged.
- **Dependencies:** FEAT-MR-1.1 through FEAT-MR-1.7.

#### FEAT-MR-1.9 — Integration, regression, and release readiness

- **Layer:** repository-wide verification.
- **Work:** run the focused suites first
  (`go test ./internal/modelranking/... ./internal/repository/... ./internal/api/...`)
  and the frontend checks
  (`npm --prefix ./web run test -- model-ranking`,
  `npm --prefix ./web run typecheck`, `npm --prefix ./web run lint`), then the
  full baseline `make verify`. Exercise the route by hand or in a test with a
  seeded database: confirm alias collapsing across two providers, period
  switching, the empty state, and that the API-key leaderboard still loads,
  sorts, edits profiles, and polls normally.
- **Acceptance:** `make verify` passes; `/ranking` renders both cards; no
  change to existing ranking endpoints, their payloads, or their tests beyond
  the new composition test; `git diff` shows only the planned files.
- **Dependencies:** FEAT-MR-1.1 through FEAT-MR-1.8.
- **Risks:** the frontend suite is large; a shared fixture or i18n parity test
  can fail for an unrelated reason, so failures must be diagnosed rather than
  masked.

**Testing and acceptance strategy**

- **Period logic (unit).** Six windows, Monday-start weeks, month arithmetic,
  half-open bounds, project `TZ`, stable keys.
- **Aggregation (repository).** Alias collapsing across providers, real-name
  fallback, window edges, five-minute peak extraction, the candidate safety cap,
  and read-pool routing. Use the existing SQLite fixture setup under
  `internal/repository/test/`.
- **Metric semantics (unit).** Expected scaled values for cache rate, TTFT,
  latency, TPM, and RPM computed from fixed rows, plus ascending versus
  descending ordering and ineligible-row omission.
- **API contract (handler).** Admin 200, viewer 403, 400 for missing, extra, or
  unknown parameters, 503 when unconfigured, `cache-control: no-store`.
- **Frontend (vitest).** Period and metric selection, persistence, polling
  gating by visibility, stale and error states, alias-collapsed rendering, and
  the two-card composition on `/ranking`.
- **i18n.** Key-set parity across en, zh, zh-TW.
- **Regression.** Existing `internal/api/test/ranking_routes_test.go`,
  `internal/api/test/key_viewer_ranking_routes_test.go`,
  `internal/ranking/local_service_test.go`, and the ranking frontend tests must
  pass unchanged.
- **Manual or browser check.** Two cards on `/ranking`, correct rendering in
  light, dark, and auto themes, no horizontal overflow at mobile width, and a
  period switch that only refreshes the model card.

**Risks and mitigations**

| Risk | Mitigation |
|---|---|
| Month/week aggregation over `usage_events` is slow on large databases (no `model_alias` index) | Candidate safety cap, 30 s process cache, time-range predicate on the indexed `timestamp` column; REC-015 if measurements justify a rollup. |
| Alias collapsing merges visually different names into one row | Show contributing `providers[]`; OQ-MR-6 covers an explicit merged-alias disclosure. |
| Metric drift between the two leaderboards | String-parity test for metric values plus formula tests against fixed rows. |
| Two independent period selectors confuse users | Independent selectors are an explicit decision; REC-019 offers a link-periods affordance if feedback asks for it. |
| Timezone mismatch with the API-key ranking (Asia/Shanghai versus project `TZ`) | Model ranking uses the project `TZ` and states it in the payload; OQ-MR-4 tracks aligning the existing ranking separately. |
| A model-ranking request failure breaks the ranking tab | The model card owns its error state; a failed model request must never unmount or reset `RankingPage`. |
| An upstream merge deletes the mount point or the module map entry | Protection entry with grep-based verification commands; the domain is self-contained and additive. |

**Open questions requiring resolution**

- **OQ-MR-1 — Metric set and default metric.** The request does not name
  metrics. The baseline exposes the seven quantitative metrics with
  `total_tokens` as the default, matching the existing vocabulary minus
  `overall`. Confirm the set and the default, or name a single metric if the
  card must stay minimal.
- **OQ-MR-2 — `overall` for models.** The existing overall score
  (`internal/ranking/local_overall.go`, `localOverallScoreExplanation`) is
  normalized only among API keys and its text says so. Decide whether models
  get an equivalent score (and against which population), or whether the card
  offers quantitative metrics only. This decides whether `overall` is added at
  all.
- **OQ-MR-3 — API-key viewer exposure.** Should `/key-ranking` show model
  ranking too? If yes it must be scoped to the viewer's own `api_group_key`, so
  a viewer cannot infer other users' model usage. Not part of the baseline.
- **OQ-MR-4 — Calendar timezone.** `internal/ranking/local_service.go` uses a
  hardcoded Asia/Shanghai zone while the project configures `time.Local` from
  `TZ`. Decide whether model ranking follows `TZ` (recommended, matching Live
  Flow and the dashboard day boundaries) and whether the existing key ranking
  should later be aligned in a separately reviewed change.
- **OQ-MR-5 — Traffic scope.** Should the model ranking include every usage
  event (baseline, matching the API-key ranking) or only generation traffic
  (`generate IS NULL OR generate = true`, as Live Flow does)? Embedding and
  other non-generation models are visible in the baseline.
- **OQ-MR-6 — Alias normalization depth.** Exact match after `TRIM` is the
  baseline and matches Live Flow. Case-insensitive or punctuation-normalized
  matching would merge more variants but risks false merges. If widening is
  wanted, the card should disclose which real names were merged.

**Resolutions (2026-11-19 expansion):** OQ-MR-1 remains the quantitative
metrics with `total_tokens` default; OQ-MR-2 now supports a model-level
`overall` score with model-population explanation; OQ-MR-3 now exposes a
strictly viewer-scoped card on `/key-ranking`; OQ-MR-4 uses project `TZ` for
both rankings; OQ-MR-5 remains every usage event; OQ-MR-6 remains exact `TRIM`
alias matching.

**Recommendations:** REC-015 through REC-022 are implemented in this expansion:
checkpointed rollups, shared overall scoring, provider filtering/contributions,
viewer-scoped ranking, optional linked periods, predecessor trends, unified TZ,
and cost/unavailable-cost columns.

---

### Feature Recommendations for User Review

> Proposals only. They are **not** part of the confirmed scope and must not be
> implemented until explicitly accepted.

#### REC-001: Reuse the existing provider-model aggregation instead of a new query

**Category:** Architecture
**Recommendation:** Serve historical model nodes by extending the existing
`GET /api/v1/usage/providers/:providerId/models` aggregation rather than
adding a Live Flow-specific endpoint.
**Reasoning:** `internal/repository/provider_models.go` already aggregates
per-model counts from `usage_events`; a second query path duplicates SQL and
tests.
**Expected Benefits:** Less code; one aggregation to maintain.
**Trade-offs:** Couples a fork feature to an upstream endpoint that merges can
change, weakening the merge isolation the request asks for; the existing
endpoint is provider-scoped, not period-scoped.
**Implementation Impact:** `internal/repository/provider_models.go`,
`internal/api/usage_identities.go`.
**Priority:** Low — recommended *against* for merge-safety reasons.
**Status:** Pending User Review

#### REC-002: Pre-aggregate model history in the existing rollup pipeline

**Category:** Performance
**Recommendation:** If per-request aggregation proves slow, extend
`UsageAggregationRunner` with a `liveflow_models` checkpoint and a small
per-period rollup table instead of querying raw events.
**Reasoning:** The project already uses checkpointed, exactly-once rollups
(`docs/ARCHITECTURE.md` §4) for overview/activity/latency.
**Expected Benefits:** Constant-time reads; aggregation cost independent of
hot retention.
**Trade-offs:** New entity, migration, runner pipeline, and backfill; larger
implementation and more merge surface.
**Implementation Impact:** `internal/entities`,
`internal/repository/migration`, `internal/poller/usage_aggregation_runner.go`.
**Priority:** Medium — only if measurements justify it.
**Status:** Pending User Review

#### REC-003: Persist the last canvas state so the first paint is never empty

**Category:** UX
**Recommendation:** Cache the last successful period response in
`localStorage` and render it while the fresh request is in flight.
**Reasoning:** The complaint includes "canvas is empty and back to start
canvas"; a stale-but-instant first paint removes that flash.
**Expected Benefits:** No empty flash on refresh; better perceived performance.
**Trade-offs:** Stale data can be briefly shown; needs an "as of" caption and
cache invalidation.
**Implementation Impact:** `web/src/features/live-flow/`.
**Priority:** Medium
**Status:** Pending User Review


#### REC-004: Add token/cost columns to the model cards

**Category:** Business Logic
**Recommendation:** Include total tokens and estimated cost per model for the
period, reusing the existing pricing snapshot.
**Reasoning:** The stated goal is preparing credits; token and cost figures
answer that more directly than request counts.
**Expected Benefits:** Directly actionable credit planning.
**Trade-offs:** Pulls pricing resolution into the aggregation path, increasing
query cost and test surface.
**Implementation Impact:** `internal/repository/liveflow_models.go`,
`internal/pricing`.
**Priority:** Medium
**Status:** Pending User Review

#### REC-005: Label the historical overlay explicitly

**Category:** UX
**Recommendation:** Visually distinguish "used in period" nodes from currently
in-flight nodes (for example a muted outline, a legend, and an "as of"
timestamp).
**Reasoning:** Mixing live and historical state on one canvas is ambiguous
otherwise, and the canvas is the only place both appear.
**Expected Benefits:** Removes misreading of stale nodes as active traffic.
**Trade-offs:** Additional styling and i18n strings.
**Implementation Impact:** `FlowTopology` node data, `LiveFlowPanel`.
**Priority:** High (clarity), low effort.
**Status:** Pending User Review

#### REC-006: Refresh the historical layer on a slow interval

**Category:** UX
**Recommendation:** Poll the period endpoint every 30–60 s while the tab is
visible instead of fetching once per period change.
**Reasoning:** Keeps the persistent view current without reconnecting the
socket.
**Expected Benefits:** Data stays fresh during long sessions.
**Trade-offs:** Extra read load; must respect `document.visibilityState` and
avoid overlapping requests.
**Implementation Impact:** `web/src/features/live-flow/`.
**Priority:** Low
**Status:** Pending User Review

#### REC-007: Encrypt or avoid persistent admin test-key storage

**Category:** Security / Data Protection
**Recommendation:** Prefer a runtime-only administrator key if persistence is
not essential. If it must survive restarts, establish an explicit encryption
key-management design before storing it in SQLite; never rely on UI masking as
protection for the value in `app_settings` or backups.
**Reasoning:** `app_settings` is ordinary SQLite data, and database backups
contain application data. The admin test key grants direct access to the CPA
model API, so plaintext persistence expands the impact of database or backup
disclosure.
**Expected Benefits:** Reduces credential exposure and makes backup/security
expectations explicit.
**Trade-offs:** Runtime-only storage loses the key on restart; encryption
requires a separately managed key, rotation/recovery procedure, and deployment
documentation.
**Implementation Impact:** FEAT-MTEST-4 settings persistence, configuration/key
management decisions, backup documentation, security tests.
**Priority:** High
**Status:** Pending User Review

#### REC-008: Use a fixed, non-mutating low-output probe request

**Category:** Reliability / Cost Control
**Recommendation:** Have the server send a fixed, minimal test prompt and a
small explicit output-token limit, rather than accepting arbitrary prompt text
or output settings from the browser.
**Reasoning:** The feature only needs to determine whether a model can answer;
arbitrary input can increase cost, expose sensitive content, or turn the action
into a general-purpose proxy. The request body/limit still requires OQ-3
agreement and CPA compatibility verification.
**Expected Benefits:** Predictable cost, reduced data exposure, simpler
validation and safer streamed-response bounds.
**Trade-offs:** A fixed probe may not exercise user-specific capabilities or
provider-specific structured input; a very small output budget may reject some
models.
**Implementation Impact:** FEAT-MTEST-1 request contract, FEAT-MTEST-2 CPA
client, service and integration tests.
**Priority:** High
**Status:** Pending User Review

#### REC-009: Make timeout bounds server-enforced and independent from global CPA timeout

**Category:** Security / Operations
**Recommendation:** Define documented minimum/maximum timeout values, validate
them server-side, and use a per-test context deadline. Do not let the browser
raise the cap or reinterpret the existing global `RequestTimeout` implicitly.
**Reasoning:** User-editable timeouts affect resource occupancy for streamed
requests. A fixed server-side range prevents unbounded connection/resource use
and keeps a test’s configured behavior distinct from ingestion and metadata
operations.
**Expected Benefits:** Predictable resource limits and fewer accidental
long-lived requests.
**Trade-offs:** A strict maximum may interrupt unusually slow providers; the
range and default need an operationally realistic choice.
**Implementation Impact:** FEAT-MTEST-1 contract, FEAT-MTEST-2 request context,
FEAT-MTEST-4 validation, settings UI and timeout tests.
**Priority:** High
**Status:** Pending User Review

#### REC-010: Send the Telegram notification automatically on every failed test

**Category:** Product / UX
**Recommendation:** Instead of (or in addition to) the explicit "Notify admin"
action, send the Telegram message automatically whenever a model test fails.
**Reasoning:** The stated problem is that the administrator does not learn about
failures. An explicit action depends on the user remembering to click it, so
some failures will still go unreported.
**Expected Benefits:** No missed reports; no extra user step; the administrator
sees failures as they happen.
**Trade-offs:** Message volume grows with test usage; a viewer can generate
traffic without an explicit action, which raises abuse and privacy concerns and
makes the rate limiter mandatory rather than defensive; repeated failures of the
same model produce duplicate messages unless deduplication is added.
**Implementation Impact:** FEAT-TG-1 contract, FEAT-TG-4 route/service call site
inside `internal/modeltest` or the route handler, FEAT-TG-6 UI (the action
becomes informational), limiter and dedupe tests.
**Priority:** Medium — requires an explicit decision because it changes the
feature's volume and abuse profile.
**Status:** Accepted and implemented (2026-11-18) — every failed probe records
the failure and notifies automatically off the response path; the explicit
**Notify admin** action remains available.

#### REC-011: Add a "Send test message" action to Admin Settings

**Category:** Operations / UX
**Recommendation:** Add an admin-only action that sends a fixed sample message
through the configured bot, chat, and topic so the administrator can verify the
configuration immediately.
**Reasoning:** Chat IDs and forum topic IDs are easy to mistype, and the first
real notification is a poor place to discover a misconfiguration. The
notification path is otherwise only exercised by a failing model test.
**Expected Benefits:** Immediate, low-risk configuration feedback; fewer silent
delivery failures; a natural place to surface Bot API error descriptions
(without the token).
**Trade-offs:** One more endpoint and UI state to build and test; it must be
rate-limited and admin-only so it cannot be used to spam the chat.
**Implementation Impact:** `internal/telegram/httpapi/routes.go`,
`internal/adminsettings/httpapi/routes.go` (or a dedicated admin route),
`AdminSettingsCard.tsx`, i18n keys, tests.
**Priority:** Medium
**Status:** Accepted and implemented (2026-11-18) — admin-only
`POST /api/v1/telegram/test-message` plus a **Send test message** button in
Admin Settings.

#### REC-012: Record the failure server-side instead of trusting the client payload

**Category:** Security / Architecture
**Recommendation:** Keep a short-lived, in-memory record of the last failed test
per session and model, and have the notify endpoint read the reason from that
record so the browser sends only `{model, mode}`.
**Reasoning:** The current plan accepts a client-supplied `reason` and re-bounds
and redacts it server-side. That is workable, but the value is still
caller-controlled, so a viewer can put arbitrary text into the administrator's
chat within the length limit.
**Expected Benefits:** The message content is fully server-authored; no
client-controlled text reaches Telegram; simpler redaction guarantees.
**Trade-offs:** Adds in-memory state with a TTL, must be keyed per session and
model, and is lost on restart or in a multi-instance deployment (the project is
single-instance today, so this is acceptable but should be documented).
**Implementation Impact:** `internal/modeltest` (record failures),
`internal/telegram` (read the record), FEAT-TG-4 handler, tests.
**Priority:** Medium — recommended if the administrator chat is considered
sensitive.
**Status:** Accepted and implemented (2026-11-18) — in-memory, per-session and
per-model failure record with a 10-minute TTL; the browser sends only
`{model, mode}`.

#### REC-013: Add an enable toggle and per-model mute

**Category:** Operations
**Recommendation:** Add a `telegram.enabled` switch and a per-model mute list so
the administrator can silence noisy models without clearing the bot token.
**Reasoning:** The baseline treats "configured" as "enabled". A single noisy
model (for example a deprecated model that always fails) would otherwise
generate a steady stream of messages, and the only remedy would be clearing the
configuration.
**Expected Benefits:** Operational control without re-entering credentials;
fewer duplicate reports for known-broken models.
**Trade-offs:** More settings surface, more validation, and a mute list that
needs its own UI and persistence; it also adds a second reason for the feature
to be inert, which must be reflected in the error codes.
**Implementation Impact:** `internal/adminsettings` (new keys),
`AdminSettingsCard.tsx`, `internal/telegram` (mute check), i18n, tests.
**Priority:** Low
**Status:** Accepted and implemented (2026-11-18) —
`admin.telegram.enabled` and `admin.telegram.muted_models`, enforced server-side
with the `telegram_disabled` and `telegram_model_muted` codes.

#### REC-014: Introduce a channel abstraction instead of a Telegram-only service

**Category:** Architecture
**Recommendation:** Define a small `Notifier` interface with a Telegram
implementation now, and keep the message model channel-neutral so a Discord or
generic webhook sender can be added later without touching the model-test
feature.
**Reasoning:** `docs/ROADMAP.md` §3.1 already plans configurable outbound
webhooks with Discord/Telegram templates. A channel-neutral message model lets
this phase stay small while leaving a clean seam for that work.
**Expected Benefits:** No duplicated message-building logic later; the
model-test feature depends on an interface rather than a Telegram client.
**Trade-offs:** Slightly more indirection than a Telegram-only service; the
abstraction must stay minimal to avoid speculative design.
**Implementation Impact:** `internal/telegram/telegram.go` (interface and
message type), FEAT-TG-2, FEAT-TG-4.
**Priority:** Low — the baseline already isolates the sender behind a
`Notifier` interface; this recommendation only asks that the message model stay
channel-neutral.
**Status:** Accepted and implemented (2026-11-18) — `telegram.Message` and
`telegram.Notifier` remain channel-neutral; Telegram is the only implementation.

**Model Ranking (Phase 2.25) proposals**
The recommendations below relate to the planned Model Ranking card. They are
**not** part of the confirmed scope above and require explicit user acceptance
before any of them becomes a task.

#### REC-015: Pre-aggregate model ranking if on-demand aggregation proves slow

**Category:** Performance
**Recommendation:** Only if measurements show the on-demand query is too slow
for month or week periods, add a per-model rollup table (period kind/key/model)
maintained by a checkpointed aggregation pipeline, and serve the leaderboard
from it.
**Reasoning:** `usage_events` has no `model_alias` index, so a month window is
grouped by an expression over every event in the range. The project already
uses checkpointed rollups (`UsageAggregationRunner`, `docs/ARCHITECTURE.md`
§4) for overview, activity, and latency, and maintains a five-minute per-API-key
snapshot in `local_ranking_period_stats`.
**Expected Benefits:** Constant-time reads; request cost independent of window
size and hot-retention volume.
**Trade-offs:** New entity, migration, aggregation pipeline and checkpoint,
backfill, plus a second copy of the truth that can drift; it also enlarges the
fork's schema surface that upstream merges must not disturb.
**Implementation Impact:** `internal/entities`,
`internal/repository/migration`, `internal/poller/usage_aggregation_runner.go`,
`internal/repository/model_ranking.go`.
**Priority:** Low until measured; the 30 s cache and candidate safety cap are
expected to cover normal installations.
**Status:** Pending User Review

#### REC-016: Define a model-level `overall` score deliberately, or omit it

**Category:** Business Logic
**Recommendation:** Either (a) keep the card quantitative-only, or (b) extract
the numeric core of `internal/ranking/local_overall.go` into a shared
population-scoring helper (token/request utility, Wilson reliability, TTFT and
latency utilities, cache and peak utility, weighted geometric mean) and run it
over model rows with model-specific explanation text.
**Reasoning:** The current overall score is normalized only among API keys and
its explanation says so. Reusing the label for models without adapting the
formula and the text would misdescribe the number. Copying the ~200-line
algorithm into the new domain would create two implementations that drift.
**Expected Benefits:** Users can rank models with the same mental model as API
keys; one scoring implementation instead of two.
**Trade-offs:** Refactoring a fork feature used in production
(`internal/ranking/local_overall.go`) adds regression risk to the API-key
leaderboard; a model population (dozens of rows) is also statistically smaller
than a key population, so the same shrinkage assumptions may not hold.
**Implementation Impact:** `internal/ranking/local_overall.go` (extraction),
`internal/modelranking/`, explanation strings, tests on both leaderboards.
**Priority:** Medium; option (a) is the baseline and needs no work.
**Status:** Pending User Review

#### REC-017: Add a provider filter and per-provider contribution to the model rows

**Category:** UX / Business Logic
**Recommendation:** Beyond listing `providers[]`, let the administrator filter
the model leaderboard to one provider and show each provider's share of the
collapsed row (requests or tokens).
**Reasoning:** The whole point of alias collapsing is that one ranked model can
be served by several providers. When a number looks surprising, the immediate
next question is "which provider produced it", and the raw provider list does
not answer how much each contributed.
**Expected Benefits:** Explains merged rows; supports cost and routing
decisions; turns the card into a diagnostic view rather than a bare top list.
**Trade-offs:** The aggregation must then group by display name and provider,
which grows the result set and the SQL; it also needs a second table state or a
drill-down affordance in the card.
**Implementation Impact:** `internal/repository/model_ranking.go`,
`internal/modelranking/`, `ModelRankingSection.tsx`, i18n, tests.
**Priority:** Medium
**Status:** Pending User Review

#### REC-018: Offer a viewer-scoped model ranking on `/key-ranking`

**Category:** Functional / Security
**Recommendation:** After the admin card ships, evaluate adding model ranking to
the API-key viewer page, strictly filtered to the session key's
`api_group_key`.
**Reasoning:** The requesting user described this as a user-facing request
("one of my user asked"), yet the global model ranking aggregates traffic from
every key. A viewer card is the safe way to serve that audience.
**Expected Benefits:** Viewers can see their own model mix without seeing other
users' usage; one interaction pattern across both roles.
**Trade-offs:** Requires a second authorization path
(`RegisterKeyViewerRoutes`-style), scoping tests, and a clear label that the
viewer sees only their own key; the global ranking must never be reachable from
the viewer group.
**Implementation Impact:** `internal/modelranking/httpapi/routes.go`,
`internal/api/router.go`, `web/src/pages/KeyRankingPage.tsx`,
`docs/MERGE-PRESERVATION.md`.
**Priority:** Medium, and only after OQ-MR-3 is answered.
**Status:** Pending User Review

#### REC-019: Add an optional "link periods" toggle for the two leaderboards

**Category:** UX
**Recommendation:** Keep the two period selectors independent by default, but
offer a small toggle that mirrors the API-key period into the model card.
**Reasoning:** Independent selectors avoid coupling, but a user comparing
"which key" with "which model" for the same window will otherwise set the same
period twice on every visit.
**Expected Benefits:** Faster side-by-side comparison; no loss of the
independent behavior for users who want it.
**Trade-offs:** One more persisted preference and interaction state; the two
cards can look inconsistent if the toggle is on and one request fails.
**Implementation Impact:** `ModelRankingSection.tsx`, the mount in
`UsagePage.tsx` (to pass the current period when linked), i18n, tests.
**Priority:** Low
**Status:** Pending User Review

#### REC-020: Show a previous-period trend per model

**Category:** UX
**Recommendation:** Add a delta column comparing the selected period with its
natural predecessor (Today vs Yesterday, Current Week vs Last Week, Current
Month vs Last Month), reusing the pairing rule already stated for Live Flow.
**Reasoning:** A ranking answers "who is on top"; a delta answers "what
changed", which is what usually prompts a routing or credit decision. The six
fixed periods make the comparison unambiguous.
**Expected Benefits:** Actionable movement instead of a static snapshot.
**Trade-offs:** Doubles the aggregation work per request (two windows) or needs
a second cached query; the oldest periods have no baseline and must render an
explicitly empty delta.
**Implementation Impact:** `internal/modelranking/` (second window),
`internal/repository/model_ranking.go`, `ModelRankingSection.tsx`, i18n, tests.
**Priority:** Medium
**Status:** Pending User Review

#### REC-021: Use one calendar definition across ranking and the dashboard

**Category:** Architecture / Correctness
**Recommendation:** Treat the hardcoded `localRankingPeriodLocation`
(`Asia/Shanghai`) in `internal/ranking/local_service.go` as a defect, and
migrate the API-key ranking to `timeutil.NormalizeStorageTime` / `TZ` in a
separately reviewed change; meanwhile model ranking uses the project `TZ`.
**Reasoning:** `docs/CONFIGURATION.md` and `README.md` describe `TZ` as the
timezone for "Today, daily totals, page timestamps", and Live Flow already
follows it. A deployment with `TZ=America/New_York` currently gets two
different "Today" windows on the same page, which is a correctness bug rather
than a preference.
**Expected Benefits:** Consistent day/week/month boundaries everywhere; no
misleading "Today" totals; one place to reason about period keys.
**Trade-offs:** Changing the existing key ranking alters which events fall in
Today/Yesterday/This month for non-Asia/Shanghai deployments. The snapshots are
cleared on the zone change: today/yesterday/current month are rebuilt on the
next read and the previous month is re-settled by a bounded background pass (at
most 31 complete days per run), so the previous-month window can briefly return
partial values while the pass catches up.
**Implementation Impact:** `internal/ranking/local_service.go`,
`internal/ranking/local_leaderboard.go`, `local_recovery_test.go`, migration or
re-settlement of `local_ranking_period_stats`, documentation.
**Priority:** High for correctness, but explicitly out of Model Ranking scope.
**Status:** Implemented (2026-11-19) — API-key ranking follows the project `TZ`,
the saved period zone is fingerprinted by name and UTC offset, live windows are
rebuilt on read, and the settlement pass is bounded and self-healing.

#### REC-022: Add an estimated cost column per model

**Category:** Functional
**Recommendation:** Include the period's estimated cost per model
(`usage_events.cost_usd` summed per display name, with the existing
"cost unavailable" handling) as an optional column.
**Reasoning:** Model choice is usually driven by cost as much as volume, and
the pricing pipeline already populates cost on ingestion and recalculation.
**Expected Benefits:** Directly answers "which model costs me most" without a
second report.
**Trade-offs:** Cost availability differs by provider and pricing coverage, so
the column needs an explicit unavailable state; aggregating it adds a pricing
dependency to a read path that is otherwise pure counters.
**Implementation Impact:** `internal/repository/model_ranking.go` (cost sum and
unavailable count), `internal/modelranking/`, `ModelRankingSection.tsx`, i18n,
tests.
**Priority:** Low
**Status:** Pending User Review

---

### Restoration Tasks (Upstream Merge Recovery)

- **2.20 Restore API Key Name Display (Key Name Alias).** 🟡 (in progress)
  - **Problem:** The upstream merge removed the `KeyName` field from
    `entities.CPAAPIKey` and broke the display priority chain. Currently, API
    keys display as: User Alias → Masked Key. The intended priority is:
    **CPA Key Name → User Alias → Masked Key**.
  - **Root cause:** Although the database schema (`key_name` column),
    migration (`20261013_add_cpa_api_key_name`), CPA client endpoint
    (`FetchAuthAPIKeys` for `/v0/management/auth-api-keys`), and sync
    function (`SyncCPAAPIKeysWithNames`) all exist, the entity field was
    removed and the sync function doesn't write to the column.
  - **Scope:**
    - Backend: Add `KeyName` field to `entities.CPAAPIKey` struct
    - Backend: Fix `SyncCPAAPIKeysWithNames()` to actually sync `key_name`
    - Backend: Update `CPAAPIKeyDisplayName()` helper to use priority:
      `KeyName` → `KeyAlias` → `MaskedKey`
    - Backend: Ensure all API key queries include `key_name` column
    - Frontend: Verify all components use the backend-provided display name
    - Tests: Add/restore tests for the three-tier display priority
    - Tests: Add test for fallback when `key_name` is empty
    - Documentation: Update `docs/MERGE-PRESERVATION.md` with explicit
      protection of this display logic and verification steps
  - **Affected modules:**
    - `internal/entities/cpa_api_key.go` (add `KeyName` field)
    - `internal/repository/cpa_api_key.go` (sync logic for `key_name`)
    - `internal/helper/redact.go` (display priority function)
    - `internal/helper/test/redact_test.go` (test coverage)
    - `internal/ranking/local_service.go` (already reads it in local struct)
    - `web/src/features/ranking/components/RankingLeaderboardResults.tsx` (uses backend name)
  - **Verification steps:**
    - Ranking leaderboard shows key names when available
    - API key settings card shows key names when available
    - Key viewer dashboard shows key name when available
    - Fallback chain works: name missing → use alias → use masked key
    - `make verify` passes (backend + frontend tests)
  - **Protected by:** `docs/MERGE-PRESERVATION.md` PF-07 (API Key Name Display)

- **2.1 Theme system polish.** ⬜
  - Extend the light/dark/auto `data-theme` system with additional theme
    variants (e.g., high-contrast, OLED dark) and per-user persistence.
- **2.2 Advanced analytics visualizations.** ⬜
  - Model/provider share over time, cost-per-identity leaderboards,
    anomaly markers on latency charts, exportable chart images.
- **2.3 Custom dashboard layouts.** ⬜
  - Pin/reorder/hide overview cards; per-admin layout persisted in
    `AppSetting`.
- **2.4 Richer exports.** ⬜
  - CSV/JSON export for overview, analysis, and identity views (not just raw
    events); scheduled export option to WORK_DIR.
- **2.5 Additional i18n locales.** ⬜
  - Community-contributed locales beyond en/zh/zh-TW (plural rules, RTL
    readiness check in the i18n layer).
- **2.6 Accessibility pass.** ⬜
  - Keyboard navigation, focus management in modals, ARIA labels across the
    custom UI kit; automated axe checks in the E2E suite (ties to 0.4).
- **2.7 Subscription-aware analytics.** ⬜
  - Filter and group events, identity, and quota views by provider
    subscription plan — the response-level `subscription` contract from
    upstream PR #404 is the backend hook; plan-distribution panels on the
    quota/identity pages.
- **2.8 Subscription change history.** ⬜
  - Persist a timeline of detected plan changes per auth file (Codex
    `pro-5x`→`pro-20x`, Claude `free`→`pro`, …) and surface it in the
    credential detail view.

### Capacity & Performance Benchmarking (new, from upstream PR #417)

*Derived from the production capacity suite merged from `author/main`
(`internal/benchmark/`) — these are the recommended follow-ups.*

- **2.9 Multi-arch capacity results.** ⬜
  - Extend the `capacity-v1` manifest to `linux/arm64` and capture a second
    reference dataset/measurement column so the deployment guidance covers
    ARM hosts, not just `linux/amd64`.
- **2.10 Benchmark regression gate in CI.** ⬜
  - Promote the JSON result contracts (`internal/benchmark/schema/`) into a
    CI check: run a reduced-rate cell against a small fixture dataset and
    fail the build when sustained ingestion or Core Dashboard p99 regresses
    beyond a manifest-defined budget.
- **2.11 Dashboard-latency SLO tracking over time.** ⬜
  - Persist each formal campaign's Core p99 / Analysis Latency p99 per CPU
    profile to a versioned history so latency drift across releases is
    visible as a trend, not a single point-in-time report.
- **2.12 Memory-pressure & limit scenarios.** ⬜
  - Add cells that run the same workload under bounded cgroup memory
    (256/512/768/1024 MiB) to convert the current "observed peak +
    headroom" guidance into a verified minimum-memory recommendation.
- **2.13 Schema-evolution re-benchmarking.** ⬜
  - Re-run the canonical dataset across schema migrations (new rollup
    tables/columns) so a migration's cost is measured before release, and
    publish the before/after in `internal/benchmark/REPORT.md`.
- **2.14 Versioned benchmark manifest publishing.** ⬜
  - Publish `manifest/capacity-v*.json` alongside each release with the
    expanded-plan and binary SHA-256 digests so results stay attributable to
    an exact, reproducible build.

### Auth Session & Client Metadata Enhancements (new, from upstream PR #423)

*Derived from the session client-activity tracking merge — recommended
follow-ups for richer session insight and manageability.*

- **2.15 Login-IP geo/enrichment layer.** ✅ done 2026-08-26 → [PHASE_2_15_PLAN.md](PHASE_2_15_PLAN.md)
  - Optional reverse-DNS or privacy-preserving geo lookup keyed by
    `login_ip` / `last_seen_ip` for the Session Settings panel (opt-in,
    async, cache with TTL). Implemented as `internal/enrichgeo` (stdlib-only,
    opt-in via `IP_ENRICHMENT_ENABLED`, async PTR resolution, in-memory TTL
    cache, private/reserved IPs classified locally). Enriches the session
    list with `loginGeo` / `lastSeenGeo`; frontend Session Settings card
    renders enriched labels in en/zh/zh-TW.
- **2.16 Session activity history table.** ⬜
  - Rather than overwriting `last_seen_ip` / `last_seen_at`, record an
    append-only per-session activity trail (timestamps + IP + UA), exposing
    "first login", "last login", and unusual-IP transitions; aligns with
    the existing usage-events archive pattern.
- **2.17 Embed-session metadata caption.** ⬜
  - Propagate login IP / last-seen for CPAMC embed sessions so the session
    list is equally informative for the header-token session fallback path.
- **2.18 Frontend a11y & layout polish for session metadata.** ⬜
  - Test the `auto-fit` detail grid at narrow widths, ensure `aria` labeling
    for the `<dl>` rows, and consider a copy-to-clipboard action for the
    User-Agent value.
- **2.19 Enhanced API Key Viewer Capabilities.** ⬜
  - **Motivation:** API key users currently have limited visibility (only
    `/key-overview`, `/key-activity`, and `/key-analysis`). Expand access to
    include request events (scoped to their key), local ranking, and Live Flow.
  - **Current state:** API key viewers have Overview (scoped), Activity (scoped),
    Analysis (scoped), optionally Local Ranking (controlled by
    `APIKeyViewerLocalRankingEnabled`). They do NOT have Request Events or Live
    Flow (explicitly denied in SPECIFICATION.md).
  - **Target state:** API key viewers should access Local Ranking (visible by
    default), Overview (own stats), Live Flow (global view of all flows),
    Analysis (own stats), Request Events (own events only).
  - See [PHASE_2_19_PLAN.md](PHASE_2_19_PLAN.md) for detailed implementation plan.

---

## Phase 3 — Alerts, Metrics & Integrations

**Goal:** Usage Keeper stops being only a place you *look at* and starts
*telling you* when things happen.

- **3.1 Webhook notifications.** ⬜
  - Configurable outbound webhooks (generic JSON + Discord/Telegram
    templates) for: quota threshold crossed, error-rate spike, ingest source
    down, aggregate cost budget exceeded.
  - **Relationship to Phase 2.24:** the model-test failure Telegram
    notification is a narrower, earlier delivery and does not replace this
    item. Phase 2.24 covers one event (a failed model test) and one channel
    (Telegram); 3.1 remains the general rule-driven notification system. If
    REC-014 is accepted, Phase 2.24's channel-neutral message model becomes the
    starting point for 3.1 rather than a parallel implementation.
- **3.2 Prometheus metrics endpoint.** ⬜
  - `GET /metrics`: ingestion rate, inbox depth, checkpoint lag, roll-up
    durations, quota refresh outcomes, Go runtime metrics.
- **3.3 Alert rule engine.** ⬜
  - Threshold rules over overview/activity stats with hysteresis and
    cooldown, configured in the UI, evaluated in the maintenance runner.
- **3.4 OpenTelemetry tracing (opt-in).** ⬜
  - OTLP exporter configuration; span coverage for ingest→process→aggregate
    and HTTP handlers.
- **3.5 SSO / OAuth2 login.** ⬜
  - Optional external identity providers (OIDC) alongside password auth, with
    role mapping to admin.
- **3.6 Anomalous-access detection signals.** ⬜
  - Surface sessions where the client metadata (IP, User-Agent) has changed
    relative to the login baseline, for example `LoginIP != LastSeenIP && UA
    changed`. Keep it deterministic and non-blocking; use the existing
    session-activity fields without adding extra upstream calls (follow-up
    to upstream PR #423).

---

## Phase 4 — Scale & Platform Growth

**Goal:** prepare for bigger footprints and a broader ecosystem.

- **4.1 Multi-database backend exploration.** ⬜
  - Postgres/MySQL behind the repository seam (analysis first: which stores
    must change, dbresolver mapping, migration strategy); gated by a config
    switch.
- **4.2 Horizontal read scaling.** ⬜
  - Read-replica routing for analytics endpoints; cache layer for hot
    overview queries with explicit invalidation on checkpoint advance.
- **4.3 Plugin system for custom metrics.** ⬜
  - Sandbox mechanism (YAML-defined aggregations or WASM) for user-defined
    rollups surfaced as new dashboard cards.
- **4.4 Quota provider expansion.** ⬜
  - New providers as they appear in the CPA ecosystem; checklist-driven
    provider template (client, normalizer, registry, tests, docs).
  - Extend the subscription contract (upstream PR #404) to the remaining
    providers (gemini_cli, kimi, xai) as their APIs expose plan metadata —
    offline resolver, canonical plan table, and badge template per provider.
- **4.5 Multi-instance awareness.** ⬜
  - Optional federation: several Keeper instances report rollups to a parent
    dashboard (builds on the ranking client plumbing).
- **4.6 Public API token support.** ⬜
  - Scoped, revocable API tokens for programmatic access (distinct from CPA
    API keys), documented in the OpenAPI spec (ties to 0.2).

---

## Maintenance Track (continuous)

Runs alongside every phase; items are pulled as needed rather than ordered.

- Dependency upgrades (Go, Gin, GORM, React, Vite, Chart.js) and security
  patch cadence.
- `make verify` kept green on all supported platforms; CI matrix hygiene.
- i18n key-parity lint in CI: en/zh/zh-TW locale keys must stay in lockstep
  (follow-up to upstream #413).
- Badge/style regression tests including WebKit rendering (follow-up to
  upstream #414).
- Make the 90-day request-events custom-range cap configurable (follow-up to
  upstream #412).
- README and `docs/*` kept in sync with every feature change (policy in
  [CONTRIBUTING.md](CONTRIBUTING.md)).
- Community support: issue triage, reproduction fixtures, release notes.

# CPA Usage Keeper — Development Roadmap

> This document tracks planned features, current work, and completed milestones.
> Items are organized by phase with dependencies and acceptance criteria. Tasks
> marked ✅ are complete; ⬜ are pending; 🚧 are in progress; ⚠️ are blocked.
>
> **Status:** Active development
> **Last updated:** 2026-10-19
>
> **⚠️ Policy reminder — Provider Model GNN permanence.** The
> `ProviderModelGNNPanel` diagram (Usage overview tab,
> `GET /api/provider-model-gnn`) is protected by a project permanence
> requirement — removal or replacement requires an explicit documented
> architectural decision recorded in `docs/ARCHITECTURE.md`,
> `docs/SPECIFICATION.md` (UC-12), and `session.md`. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) for the complete policy.
>
> **⚠️ Policy reminder — AI Provider Table Redesign & Detail Navigation
> permanence.** The `/ai-provider` provider grouping table and the
> `/ai-provider/:providerId` detail navigation are protected by a project
> permanence requirement — removal or replacement requires an explicit
> documented architectural decision recorded in `docs/ARCHITECTURE.md`,
> `docs/SPECIFICATION.md` (UC-7a), and `session.md`. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) PF-06 for the complete policy.

---

## Task ID Naming Convention

- **Feature phases:** `FEAT-PROV-X.Y` (AI Provider table redesign)
- **Live Flow history:** `FEAT-LF-X.Y` (Phase 2.22)
- **Model Ranking:** `FEAT-MR-X.Y` (Phase 2.25)
- **General improvements:** `IMP-X.Y`
- **Bug fixes:** `BUG-X`
- **Documentation:** `DOC-X`

---

## Current Phase: AI Provider Table Redesign & Detail Navigation

**Phase Goal:** Simplify the `/ai-provider` table to show only provider names (without API keys), add a detail page/view for each provider showing all keys with success rate sorting, and ensure proper access control for Admin and API-key viewer roles.

**Status:** ✅ **Complete** — Protected as a permanent core feature (see `docs/MERGE-PRESERVATION.md` PF-06).

**Context:**  
The current `/ai-provider` table shows all providers and their keys in a single flat table, which becomes bloated when many providers have many keys. This phase restructures the view to:
1. Show one row per provider in the main table
2. Add a "Detail" action to navigate to a provider-specific view
3. Display all keys for that provider on the detail page
4. Support success rate sorting in both views
5. Enforce proper access control (Admin + API-key viewer only)

**Background:**  
- Success rate sorting is already implemented (see `session.md`, 2026-10-19)
- Role-based credential masking exists for API-key viewers
- Current credential sections use `CredentialDetailDrawer` for identity details
- Need to preserve admin functionality while adding API-key viewer scoping

---

## Phase 1: Backend — Provider Grouping & Key Listing API

### FEAT-PROV-1.1 — Design Provider Grouping Model ✅

**Status:** Complete  
**Objective:** Define the data model for provider grouping and key listing to support the new UI structure.

**Implementation Details:**
- Review existing `UsageIdentity` model and determine how to group by provider
- Provider dimension: Use `provider` field from `usage_events` / `usage_identities`
- Key listing: Filter `UsageIdentity` records by `auth_type='apikey'` and `provider` field
- Success rate calculation: Already exists in `internal/repository/usage_identities.go`
- Scoping for API-key viewers: Filter by session's `api_key_id`

**Files Affected:**
- Review: `internal/entities/usage_identity.go`
- Review: `internal/repository/usage_identities.go`
- Review: `internal/service/usage_identities.go`

**Acceptance Criteria:**
- [ ] Data model design documented
- [ ] Grouping strategy defined
- [ ] API-key viewer scoping rules defined
- [ ] Success rate sorting approach confirmed

**Dependencies:** None

---

### FEAT-PROV-1.2 — Backend Endpoint: List Providers ✅

**Status:** Complete  
**Objective:** Create or extend an endpoint to list unique providers (grouped view) for the main `/ai-provider` table.

**Implementation Details:**
- Endpoint: `GET /api/v1/usage/providers` (or extend existing identity endpoint)
- Return: List of providers with aggregate stats (total keys, success rate, last used, etc.)
- SQL: `SELECT provider, COUNT(*) as key_count, ... FROM usage_identities WHERE auth_type='apikey' GROUP BY provider`
- Include success rate calculation per provider (average or aggregate)
- Role-based filtering:
  - Admin: See all providers
  - API-key viewer: See only providers with keys owned by their API key
- Pagination support (optional, if many providers)

**Files Affected:**
- New or extend: `internal/api/usage_identities.go` (handler)
- New or extend: `internal/service/usage_identities.go` (service method)
- New or extend: `internal/repository/usage_identities.go` (query)

**Acceptance Criteria:**
- [ ] Endpoint returns unique provider list with aggregate stats
- [ ] Success rate included per provider
- [ ] Admin sees all providers
- [ ] API-key viewer sees only their providers
- [ ] Response includes necessary fields for UI rendering
- [ ] Unit tests for service and repository layers
- [ ] API integration tests for both roles

**Dependencies:** FEAT-PROV-1.1

**Tests:**
- Unit: Repository query returns correct provider groups
- Unit: Service applies role-based filtering
- Integration: Admin endpoint returns all providers
- Integration: API-key viewer endpoint returns scoped providers
- Integration: Success rate calculation is accurate

---

### FEAT-PROV-1.3 — Backend Endpoint: List Keys by Provider ✅

**Status:** Complete  
**Objective:** Create an endpoint to fetch all keys for a specific provider, supporting the provider detail page.

**Implementation Details:**
- Endpoint: `GET /api/v1/usage/providers/:provider/keys` (or similar)
- Query: `SELECT * FROM usage_identities WHERE auth_type='apikey' AND provider=:provider`
- Support sorting by success rate (reuse `applyUsageIdentityPageSort`)
- Pagination support (reuse existing `UsageIdentityPageQuery`)
- Role-based scoping:
  - Admin: All keys for the provider
  - API-key viewer: Only keys owned by their API key
- Credential masking: Apply existing masking for API-key viewers

**Files Affected:**
- New: `internal/api/usage_identities.go` (handler for provider keys)
- Extend: `internal/service/usage_identities.go` (service method)
- Extend: `internal/repository/usage_identities.go` (query with provider filter)

**Acceptance Criteria:**
- [ ] Endpoint returns keys filtered by provider
- [ ] Sorting by success rate works
- [ ] Pagination works
- [ ] Admin sees all keys for provider
- [ ] API-key viewer sees only their keys for provider
- [ ] Credential masking applied for API-key viewers
- [ ] Unit and integration tests for both roles

**Dependencies:** FEAT-PROV-1.1, FEAT-PROV-1.2

---

### FEAT-PROV-1.4 — Backend Route Registration & Middleware ✅

**Status:** Complete  
**Objective:** Register new provider endpoints with correct auth middleware.

**Implementation Details:**
- Register in `internal/api/router.go`:
  - Provider list: `adminProtected` or `versionProtected` (both admin + viewer)
  - Provider keys: `adminProtected` or `versionProtected` (both admin + viewer)
- Reuse existing middleware:
  - `authHandler.roleMiddleware(auth.RoleAdmin, auth.RoleAPIKeyViewer)`
  - `costReadMiddleware` if cost data is included
- Ensure rate limiting applies to API-key viewers

**Files Affected:**
- `internal/api/router.go`

**Acceptance Criteria:**
- [ ] Routes registered with correct path and handler
- [ ] Auth middleware enforces admin + API-key viewer only
- [ ] Rate limiting applied
- [ ] Routes accessible to both roles, denied to unauthenticated

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3

**Tests:**
- Integration: Unauthenticated request returns 401
- Integration: Admin can access routes
- Integration: API-key viewer can access routes

---

## Phase 2: Frontend — Provider Table Redesign

### FEAT-PROV-2.1 — API Client: Provider List & Keys ✅

**Status:** Complete  
**Objective:** Add frontend API client functions for the new provider endpoints.

**Implementation Details:**
- Add to `web/src/lib/api.ts`:
  - `fetchProviders(query?: ProviderListQuery): Promise<ProviderListResponse>`
  - `fetchProviderKeys(provider: string, query?: UsageIdentityPageQuery): Promise<UsageIdentityPageResponse>`
- Define TypeScript types in `web/src/lib/types.ts`:
  - `ProviderListResponse` (provider name, key count, success rate, etc.)
  - Reuse existing `UsageIdentityPageQuery`, `UsageIdentityPageResponse`
- Use existing `buildQueryString`, `credentials: 'include'`, error handling

**Files Affected:**
- `web/src/lib/api.ts`
- `web/src/lib/types.ts`

**Acceptance Criteria:**
- [ ] API client functions implemented
- [ ] TypeScript types defined
- [ ] Error handling works
- [ ] Unit tests for API client (mocked fetch)

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3

**Tests:**
- Unit: `fetchProviders` constructs correct URL and handles response
- Unit: `fetchProviderKeys` constructs correct URL with provider param
- Unit: Error responses handled correctly

---

### FEAT-PROV-2.2 — Redesign AI Provider Table (Main View) ✅

**Status:** Complete  
**Objective:** Update `/ai-provider` table to show only provider names, remove key details, add "Detail" action column.

**Implementation Details:**
- Modify `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`:
  - Change data source from `aiProviderRows` (keys) to new `providerRows` (grouped)
  - Table columns: Provider Name, Key Count, Success Rate, Last Used, Detail Action
  - Remove columns: Identity, Prefix, Base URL (moved to detail page)
  - "Detail" action: Link or button to navigate to provider detail page
- Update state management in `web/src/pages/UsagePage.tsx`:
  - Fetch provider list instead of flat key list
  - Store in `credentialsData.providerRows`
- Preserve sorting by success rate in provider list
- Preserve pagination if many providers

**Files Affected:**
- `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`
- `web/src/pages/UsagePage.tsx`
- `web/src/components/usage/credentials/CredentialSectionShell.tsx` (may need adjustments)

**Acceptance Criteria:**
- [ ] Main table shows provider-level rows only
- [ ] "Detail" column/action present for each provider
- [ ] Success rate sorting works
- [ ] Pagination works (if implemented)
- [ ] No key-level details visible in main table
- [ ] UI rendering matches design

**Dependencies:** FEAT-PROV-2.1

**Tests:**
- Unit: Provider table renders correct columns
- Unit: "Detail" action triggers navigation
- Integration: Table displays provider list from API
- Integration: Sorting by success rate works

---

### FEAT-PROV-2.3 — Provider Detail Page/Component ✅

**Status:** Complete  
**Objective:** Create a new page to show all keys for a selected provider.

**Implementation Details:**
- New full page: `web/src/pages/ProviderDetailPage.tsx`
- Route: `/ai-provider/:providerId` or `/usage/provider/:providerId`
- Content:
  - Breadcrumb or back button: Back to AI Providers
  - Provider name header
  - Table: All keys for this provider (identity, alias, prefix, base_url, stats, health)
  - Sorting: By success rate, requests, tokens, last used
  - Pagination: Reuse existing key pagination
- Fetch keys via `fetchProviderKeys(providerId, query)`
- Apply role-based scoping (API-key viewers see only their keys)
- Credential masking for API-key viewers

**Files Affected:**
- New: `web/src/pages/ProviderDetailPage.tsx`
- Update: `web/src/App.tsx` (add route)
- Reuse: `web/src/components/usage/credentials/` (table components)

**Acceptance Criteria:**
- [ ] Detail page accessible via route
- [ ] Displays all keys for selected provider
- [ ] Sorting by success rate works
- [ ] Pagination works
- [ ] Admin sees all keys
- [ ] API-key viewer sees only their keys
- [ ] Credential masking works for viewers
- [ ] Back navigation works
- [ ] Empty state if no keys

**Dependencies:** FEAT-PROV-2.1, FEAT-PROV-2.2

---

### FEAT-PROV-2.4 — Navigation & Routing Updates ✅

**Status:** Complete  
**Objective:** Wire up navigation from main table to detail page, update routing.

**Implementation Details:**
- Update `web/src/App.tsx`:
  - Add route: `<Route path="/ai-provider/:providerId" element={<ProviderDetailPage />} />`
- Update `AiProviderCredentialsSection`:
  - "Detail" action: `<Link to={`/ai-provider/${encodeURIComponent(provider.name)}`}>Detail</Link>`
- Update breadcrumbs or header in `ProviderDetailPage`:
  - Show current provider name, back link to `/ai-provider`

**Files Affected:**
- `web/src/App.tsx`
- `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`
- `web/src/pages/ProviderDetailPage.tsx`

**Acceptance Criteria:**
- [ ] Clicking "Detail" navigates to provider detail page
- [ ] Browser back button works
- [ ] Direct URL access to detail page works
- [ ] Breadcrumbs/back navigation works
- [ ] URL encoding handles special characters

**Dependencies:** FEAT-PROV-2.3

---

## Phase 3: Internationalization, Testing & Documentation

### FEAT-PROV-3.1 — i18n: Provider Table & Detail Page ✅

**Status:** Complete  
**Objective:** Add translations for all new UI strings in English, Simplified Chinese, and Traditional Chinese.

**Implementation Details:**
- Add to `web/src/i18n/index.ts`:
  - `usage_stats.providers_table_title`: "AI Providers" / "AI 提供商" / "AI 提供商"
  - `usage_stats.providers_table_subtitle`: "Click Detail to view keys"
  - `usage_stats.providers_detail_title`: "Provider: {name}"
  - `usage_stats.providers_key_count`: "Keys" / "密钥数" / "密鑰數"
  - `usage_stats.providers_detail_action`: "Detail" / "详情" / "詳情"
  - `usage_stats.providers_back_to_list`: "Back to Providers"
  - Reuse existing: `credentials_sort_success_rate`, `credentials_empty_state`
- All three locales (en, zh, zh-TW) must be updated together

**Files Affected:**
- `web/src/i18n/index.ts`

**Acceptance Criteria:**
- [ ] All new UI strings have translations
- [ ] English, Simplified Chinese, Traditional Chinese all complete
- [ ] No missing translation warnings

**Dependencies:** FEAT-PROV-2.2, FEAT-PROV-2.3

---

### FEAT-PROV-3.2 — Backend Tests ✅

**Status:** Complete  
**Objective:** Comprehensive backend test coverage for all new/changed code.

**Implementation Details:**
- Unit tests:
  - Repository: Provider grouping, key filtering, success rate calculation
  - Service: Role-based scoping, credential masking, aggregation
  - Handlers: Request parsing, response mapping, error handling
- Integration tests:
  - Provider list endpoint: Admin, API-key viewer, unauthenticated
  - Provider keys endpoint: Admin, API-key viewer, cross-key denial
  - Sorting, pagination, credential masking
- Follow existing test patterns in `internal/api/test/`, `internal/repository/test/`

**Files Affected:**
- `internal/api/test/*_test.go`
- `internal/repository/test/*_test.go`

**Acceptance Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage ≥80% for new code
- [ ] `go test ./cmd/... ./internal/...` passes

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3, FEAT-PROV-1.4

---

### FEAT-PROV-3.3 — Frontend Tests ✅

**Status:** Complete  
**Objective:** Comprehensive frontend test coverage for all new/changed UI components.

**Implementation Details:**
- Unit tests:
  - API client: `fetchProviders`, `fetchProviderKeys` (mocked fetch)
  - Components: Provider table rendering, detail page rendering
  - Navigation: Link clicks, route changes
  - Sorting: Dropdown selection, data re-sorting
- Integration tests (vitest + happy-dom):
  - Full page rendering with mocked API
  - Navigation flow: main → detail → back
- Follow existing test patterns in `web/src/components/usage/credentials/test/`

**Files Affected:**
- `web/src/lib/test/api.test.ts`
- `web/src/components/usage/credentials/test/*`
- `web/src/pages/test/ProviderDetailPage.test.tsx` (new)

**Acceptance Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage ≥80% for new code
- [ ] `npm --prefix ./web run test` passes

**Dependencies:** FEAT-PROV-2.1, FEAT-PROV-2.2, FEAT-PROV-2.3, FEAT-PROV-2.4

---

### FEAT-PROV-3.4 — Verification Baseline ✅

**Status:** Complete  
**Objective:** Run full project verification baseline to ensure no regressions.

**Implementation Details:**
- Run `make verify`:
  - `go test ./cmd/... ./internal/...` (backend tests)
  - `npm --prefix ./web ci` (install deps)
  - `npm --prefix ./web run test` (vitest)
  - `npm --prefix ./web run lint` (eslint)
  - `npm --prefix ./web run typecheck` (tsc --noEmit strict)
  - `npm --prefix ./web run build` (vite production build)
- All steps must pass with no errors

**Acceptance Criteria:**
- [ ] `make verify` passes completely
- [ ] No new lint errors
- [ ] No new type errors
- [ ] Production build succeeds

**Dependencies:** FEAT-PROV-3.2, FEAT-PROV-3.3

---

### FEAT-PROV-3.5 — Documentation Updates ✅

**Status:** Complete  
**Objective:** Update all relevant documentation.

**Implementation Details:**
- Update `docs/SPECIFICATION.md`:
  - §3 Use Cases: Update UC-8 to mention provider detail view
  - §5 API Surface: Document new endpoints
- Update `docs/ARCHITECTURE.md`:
  - §7 Frontend: Document new `ProviderDetailPage` route
- Update `docs/CODE_GRAPH.md`:
  - §7 API Surface Map: Add provider endpoints
- Update `session.md`:
  - Record implementation decisions and results
- Update `CHANGELOG.md`:
  - Add entry under `[Unreleased]`

**Files Affected:**
- `docs/SPECIFICATION.md`
- `docs/ARCHITECTURE.md`
- `docs/CODE_GRAPH.md`
- `session.md`
- `CHANGELOG.md`

**Acceptance Criteria:**
- [ ] Spec accurately describes new feature
- [ ] Architecture reflects new structure
- [ ] Code graph includes new components
- [ ] Session log updated
- [ ] Changelog entry complete

**Dependencies:** All feature tasks complete

---

## Summary & Open Questions

### Summary of Phases

| Phase | Description | Tasks | Status |
|-------|-------------|-------|--------|
| Phase 1 | Backend — Provider Grouping & Key Listing API | 4 | ✅ Complete |
| Phase 2 | Frontend — Provider Table Redesign | 4 | ✅ Complete |
| Phase 3 | Internationalization, Testing & Documentation | 5 | ✅ Complete |

**Total Tasks:** 13  
**Completed:** 13  
**Remaining:** 0

> **⚠️ Protected feature:** This phase is a **permanent core feature** and is
> protected from removal or replacement by upstream merges. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) PF-06 for the complete policy.

---

### Open Questions & Decisions

**Q1: Provider Detail — Full Page or Drawer/Modal?**  
**Decision:** Full page (`/ai-provider/:providerId`) per user request.

**Q2: API-key Viewer Scoping — See All Providers or Only Own?**  
**Decision:** API-key viewers see only providers with keys they own.

**Q3: Success Rate Sorting — Provider Level or Key Level?**  
**Decision:** Both. Provider list can sort by aggregate success rate; detail page sorts keys by individual success rate.

**Q4: URL Structure for Detail Page?**  
**Decision:** `/ai-provider/:providerId` (provider name URL-encoded)

---

### Known Limitations & Future Work

- Provider-level success rate: Aggregate calculation may not reflect individual key performance
- Empty provider filtering: Handle gracefully when API-key viewer has keys but provider has no metadata
- Provider naming: Assumes provider names are unique and stable
- Mobile optimization: Detail page table may need responsive layout adjustments

---

## Related Documents

- [SPECIFICATION.md](docs/SPECIFICATION.md) — Functional contract
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — System architecture
- [CODE_GRAPH.md](docs/CODE_GRAPH.md) — Codebase structure
- [CONTRIBUTING.md](docs/CONTRIBUTING.md) — Development workflow
- [session.md](session.md) — Session context and decisions
- [CHANGELOG.md](CHANGELOG.md) — Notable changes

---

**End of Roadmap**

# CPA Usage Keeper — Development Roadmap

> This document tracks planned features, current work, and completed milestones.
> Items are organized by phase with dependencies and acceptance criteria. Tasks
> marked ✅ are complete; ⬜ are pending; 🚧 are in progress; ⚠️ are blocked.
>
> **Status:** Active development
> **Last updated:** 2026-10-19
>
> **⚠️ Policy reminder — Provider Model GNN permanence.** The
> `ProviderModelGNNPanel` diagram (Usage overview tab,
> `GET /api/provider-model-gnn`) is protected by a project permanence
> requirement — removal or replacement requires an explicit documented
> architectural decision recorded in `docs/ARCHITECTURE.md`,
> `docs/SPECIFICATION.md` (UC-12), and `session.md`. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) for the complete policy.
>
> **⚠️ Policy reminder — AI Provider Table Redesign & Detail Navigation
> permanence.** The `/ai-provider` provider grouping table and the
> `/ai-provider/:providerId` detail navigation are protected by a project
> permanence requirement — removal or replacement requires an explicit
> documented architectural decision recorded in `docs/ARCHITECTURE.md`,
> `docs/SPECIFICATION.md` (UC-7a), and `session.md`. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) PF-06 for the complete policy.

---

## Task ID Naming Convention

- **Feature phases:** `FEAT-PROV-X.Y` (AI Provider table redesign)
- **General improvements:** `IMP-X.Y`
- **Bug fixes:** `BUG-X`
- **Documentation:** `DOC-X`

---

## Current Phase: AI Provider Table Redesign & Detail Navigation

**Phase Goal:** Simplify the `/ai-provider` table to show only provider names (without API keys), add a detail page/view for each provider showing all keys with success rate sorting, and ensure proper access control for Admin and API-key viewer roles.

**Status:** ✅ **Complete** — Protected as a permanent core feature (see `docs/MERGE-PRESERVATION.md` PF-06).

**Context:**  
The current `/ai-provider` table shows all providers and their keys in a single flat table, which becomes bloated when many providers have many keys. This phase restructures the view to:
1. Show one row per provider in the main table
2. Add a "Detail" action to navigate to a provider-specific view
3. Display all keys for that provider on the detail page
4. Support success rate sorting in both views
5. Enforce proper access control (Admin + API-key viewer only)

**Background:**  
- Success rate sorting is already implemented (see `session.md`, 2026-10-19)
- Role-based credential masking exists for API-key viewers
- Current credential sections use `CredentialDetailDrawer` for identity details
- Need to preserve admin functionality while adding API-key viewer scoping

---

## Phase 1: Backend — Provider Grouping & Key Listing API

### FEAT-PROV-1.1 — Design Provider Grouping Model ✅

**Status:** Complete  
**Objective:** Define the data model for provider grouping and key listing to support the new UI structure.

**Implementation Details:**
- Review existing `UsageIdentity` model and determine how to group by provider
- Provider dimension: Use `provider` field from `usage_events` / `usage_identities`
- Key listing: Filter `UsageIdentity` records by `auth_type='apikey'` and `provider` field
- Success rate calculation: Already exists in `internal/repository/usage_identities.go`
- Scoping for API-key viewers: Filter by session's `api_key_id`

**Files Affected:**
- Review: `internal/entities/usage_identity.go`
- Review: `internal/repository/usage_identities.go`
- Review: `internal/service/usage_identities.go`

**Acceptance Criteria:**
- [ ] Data model design documented
- [ ] Grouping strategy defined
- [ ] API-key viewer scoping rules defined
- [ ] Success rate sorting approach confirmed

**Dependencies:** None

---

### FEAT-PROV-1.2 — Backend Endpoint: List Providers ✅

**Status:** Complete  
**Objective:** Create or extend an endpoint to list unique providers (grouped view) for the main `/ai-provider` table.

**Implementation Details:**
- Endpoint: `GET /api/v1/usage/providers` (or extend existing identity endpoint)
- Return: List of providers with aggregate stats (total keys, success rate, last used, etc.)
- SQL: `SELECT provider, COUNT(*) as key_count, ... FROM usage_identities WHERE auth_type='apikey' GROUP BY provider`
- Include success rate calculation per provider (average or aggregate)
- Role-based filtering:
  - Admin: See all providers
  - API-key viewer: See only providers with keys owned by their API key
- Pagination support (optional, if many providers)

**Files Affected:**
- New or extend: `internal/api/usage_identities.go` (handler)
- New or extend: `internal/service/usage_identities.go` (service method)
- New or extend: `internal/repository/usage_identities.go` (query)

**Acceptance Criteria:**
- [ ] Endpoint returns unique provider list with aggregate stats
- [ ] Success rate included per provider
- [ ] Admin sees all providers
- [ ] API-key viewer sees only their providers
- [ ] Response includes necessary fields for UI rendering
- [ ] Unit tests for service and repository layers
- [ ] API integration tests for both roles

**Dependencies:** FEAT-PROV-1.1

**Tests:**
- Unit: Repository query returns correct provider groups
- Unit: Service applies role-based filtering
- Integration: Admin endpoint returns all providers
- Integration: API-key viewer endpoint returns scoped providers
- Integration: Success rate calculation is accurate

---

### FEAT-PROV-1.3 — Backend Endpoint: List Keys by Provider ✅

**Status:** Complete  
**Objective:** Create an endpoint to fetch all keys for a specific provider, supporting the provider detail page.

**Implementation Details:**
- Endpoint: `GET /api/v1/usage/providers/:provider/keys` (or similar)
- Query: `SELECT * FROM usage_identities WHERE auth_type='apikey' AND provider=:provider`
- Support sorting by success rate (reuse `applyUsageIdentityPageSort`)
- Pagination support (reuse existing `UsageIdentityPageQuery`)
- Role-based scoping:
  - Admin: All keys for the provider
  - API-key viewer: Only keys owned by their API key
- Credential masking: Apply existing masking for API-key viewers

**Files Affected:**
- New: `internal/api/usage_identities.go` (handler for provider keys)
- Extend: `internal/service/usage_identities.go` (service method)
- Extend: `internal/repository/usage_identities.go` (query with provider filter)

**Acceptance Criteria:**
- [ ] Endpoint returns keys filtered by provider
- [ ] Sorting by success rate works
- [ ] Pagination works
- [ ] Admin sees all keys for provider
- [ ] API-key viewer sees only their keys for provider
- [ ] Credential masking applied for API-key viewers
- [ ] Unit and integration tests for both roles

**Dependencies:** FEAT-PROV-1.1, FEAT-PROV-1.2

---

### FEAT-PROV-1.4 — Backend Route Registration & Middleware ✅

**Status:** Complete  
**Objective:** Register new provider endpoints with correct auth middleware.

**Implementation Details:**
- Register in `internal/api/router.go`:
  - Provider list: `adminProtected` or `versionProtected` (both admin + viewer)
  - Provider keys: `adminProtected` or `versionProtected` (both admin + viewer)
- Reuse existing middleware:
  - `authHandler.roleMiddleware(auth.RoleAdmin, auth.RoleAPIKeyViewer)`
  - `costReadMiddleware` if cost data is included
- Ensure rate limiting applies to API-key viewers

**Files Affected:**
- `internal/api/router.go`

**Acceptance Criteria:**
- [ ] Routes registered with correct path and handler
- [ ] Auth middleware enforces admin + API-key viewer only
- [ ] Rate limiting applied
- [ ] Routes accessible to both roles, denied to unauthenticated

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3

**Tests:**
- Integration: Unauthenticated request returns 401
- Integration: Admin can access routes
- Integration: API-key viewer can access routes

---

## Phase 2: Frontend — Provider Table Redesign

### FEAT-PROV-2.1 — API Client: Provider List & Keys ✅

**Status:** Complete  
**Objective:** Add frontend API client functions for the new provider endpoints.

**Implementation Details:**
- Add to `web/src/lib/api.ts`:
  - `fetchProviders(query?: ProviderListQuery): Promise<ProviderListResponse>`
  - `fetchProviderKeys(provider: string, query?: UsageIdentityPageQuery): Promise<UsageIdentityPageResponse>`
- Define TypeScript types in `web/src/lib/types.ts`:
  - `ProviderListResponse` (provider name, key count, success rate, etc.)
  - Reuse existing `UsageIdentityPageQuery`, `UsageIdentityPageResponse`
- Use existing `buildQueryString`, `credentials: 'include'`, error handling

**Files Affected:**
- `web/src/lib/api.ts`
- `web/src/lib/types.ts`

**Acceptance Criteria:**
- [ ] API client functions implemented
- [ ] TypeScript types defined
- [ ] Error handling works
- [ ] Unit tests for API client (mocked fetch)

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3

**Tests:**
- Unit: `fetchProviders` constructs correct URL and handles response
- Unit: `fetchProviderKeys` constructs correct URL with provider param
- Unit: Error responses handled correctly

---

### FEAT-PROV-2.2 — Redesign AI Provider Table (Main View) ✅

**Status:** Complete  
**Objective:** Update `/ai-provider` table to show only provider names, remove key details, add "Detail" action column.

**Implementation Details:**
- Modify `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`:
  - Change data source from `aiProviderRows` (keys) to new `providerRows` (grouped)
  - Table columns: Provider Name, Key Count, Success Rate, Last Used, Detail Action
  - Remove columns: Identity, Prefix, Base URL (moved to detail page)
  - "Detail" action: Link or button to navigate to provider detail page
- Update state management in `web/src/pages/UsagePage.tsx`:
  - Fetch provider list instead of flat key list
  - Store in `credentialsData.providerRows`
- Preserve sorting by success rate in provider list
- Preserve pagination if many providers

**Files Affected:**
- `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`
- `web/src/pages/UsagePage.tsx`
- `web/src/components/usage/credentials/CredentialSectionShell.tsx` (may need adjustments)

**Acceptance Criteria:**
- [ ] Main table shows provider-level rows only
- [ ] "Detail" column/action present for each provider
- [ ] Success rate sorting works
- [ ] Pagination works (if implemented)
- [ ] No key-level details visible in main table
- [ ] UI rendering matches design

**Dependencies:** FEAT-PROV-2.1

**Tests:**
- Unit: Provider table renders correct columns
- Unit: "Detail" action triggers navigation
- Integration: Table displays provider list from API
- Integration: Sorting by success rate works

---

### FEAT-PROV-2.3 — Provider Detail Page/Component ✅

**Status:** Complete  
**Objective:** Create a new page to show all keys for a selected provider.

**Implementation Details:**
- New full page: `web/src/pages/ProviderDetailPage.tsx`
- Route: `/ai-provider/:providerId` or `/usage/provider/:providerId`
- Content:
  - Breadcrumb or back button: Back to AI Providers
  - Provider name header
  - Table: All keys for this provider (identity, alias, prefix, base_url, stats, health)
  - Sorting: By success rate, requests, tokens, last used
  - Pagination: Reuse existing key pagination
- Fetch keys via `fetchProviderKeys(providerId, query)`
- Apply role-based scoping (API-key viewers see only their keys)
- Credential masking for API-key viewers

**Files Affected:**
- New: `web/src/pages/ProviderDetailPage.tsx`
- Update: `web/src/App.tsx` (add route)
- Reuse: `web/src/components/usage/credentials/` (table components)

**Acceptance Criteria:**
- [ ] Detail page accessible via route
- [ ] Displays all keys for selected provider
- [ ] Sorting by success rate works
- [ ] Pagination works
- [ ] Admin sees all keys
- [ ] API-key viewer sees only their keys
- [ ] Credential masking works for viewers
- [ ] Back navigation works
- [ ] Empty state if no keys

**Dependencies:** FEAT-PROV-2.1, FEAT-PROV-2.2

---

### FEAT-PROV-2.4 — Navigation & Routing Updates ✅

**Status:** Complete  
**Objective:** Wire up navigation from main table to detail page, update routing.

**Implementation Details:**
- Update `web/src/App.tsx`:
  - Add route: `<Route path="/ai-provider/:providerId" element={<ProviderDetailPage />} />`
- Update `AiProviderCredentialsSection`:
  - "Detail" action: `<Link to={`/ai-provider/${encodeURIComponent(provider.name)}`}>Detail</Link>`
- Update breadcrumbs or header in `ProviderDetailPage`:
  - Show current provider name, back link to `/ai-provider`

**Files Affected:**
- `web/src/App.tsx`
- `web/src/components/usage/credentials/AiProviderCredentialsSection.tsx`
- `web/src/pages/ProviderDetailPage.tsx`

**Acceptance Criteria:**
- [ ] Clicking "Detail" navigates to provider detail page
- [ ] Browser back button works
- [ ] Direct URL access to detail page works
- [ ] Breadcrumbs/back navigation works
- [ ] URL encoding handles special characters

**Dependencies:** FEAT-PROV-2.3

---

## Phase 3: Internationalization, Testing & Documentation

### FEAT-PROV-3.1 — i18n: Provider Table & Detail Page ✅

**Status:** Complete  
**Objective:** Add translations for all new UI strings in English, Simplified Chinese, and Traditional Chinese.

**Implementation Details:**
- Add to `web/src/i18n/index.ts`:
  - `usage_stats.providers_table_title`: "AI Providers" / "AI 提供商" / "AI 提供商"
  - `usage_stats.providers_table_subtitle`: "Click Detail to view keys"
  - `usage_stats.providers_detail_title`: "Provider: {name}"
  - `usage_stats.providers_key_count`: "Keys" / "密钥数" / "密鑰數"
  - `usage_stats.providers_detail_action`: "Detail" / "详情" / "詳情"
  - `usage_stats.providers_back_to_list`: "Back to Providers"
  - Reuse existing: `credentials_sort_success_rate`, `credentials_empty_state`
- All three locales (en, zh, zh-TW) must be updated together

**Files Affected:**
- `web/src/i18n/index.ts`

**Acceptance Criteria:**
- [ ] All new UI strings have translations
- [ ] English, Simplified Chinese, Traditional Chinese all complete
- [ ] No missing translation warnings

**Dependencies:** FEAT-PROV-2.2, FEAT-PROV-2.3

---

### FEAT-PROV-3.2 — Backend Tests ✅

**Status:** Complete  
**Objective:** Comprehensive backend test coverage for all new/changed code.

**Implementation Details:**
- Unit tests:
  - Repository: Provider grouping, key filtering, success rate calculation
  - Service: Role-based scoping, credential masking, aggregation
  - Handlers: Request parsing, response mapping, error handling
- Integration tests:
  - Provider list endpoint: Admin, API-key viewer, unauthenticated
  - Provider keys endpoint: Admin, API-key viewer, cross-key denial
  - Sorting, pagination, credential masking
- Follow existing test patterns in `internal/api/test/`, `internal/repository/test/`

**Files Affected:**
- `internal/api/test/*_test.go`
- `internal/repository/test/*_test.go`

**Acceptance Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage ≥80% for new code
- [ ] `go test ./cmd/... ./internal/...` passes

**Dependencies:** FEAT-PROV-1.2, FEAT-PROV-1.3, FEAT-PROV-1.4

---

### FEAT-PROV-3.3 — Frontend Tests ✅

**Status:** Complete  
**Objective:** Comprehensive frontend test coverage for all new/changed UI components.

**Implementation Details:**
- Unit tests:
  - API client: `fetchProviders`, `fetchProviderKeys` (mocked fetch)
  - Components: Provider table rendering, detail page rendering
  - Navigation: Link clicks, route changes
  - Sorting: Dropdown selection, data re-sorting
- Integration tests (vitest + happy-dom):
  - Full page rendering with mocked API
  - Navigation flow: main → detail → back
- Follow existing test patterns in `web/src/components/usage/credentials/test/`

**Files Affected:**
- `web/src/lib/test/api.test.ts`
- `web/src/components/usage/credentials/test/*`
- `web/src/pages/test/ProviderDetailPage.test.tsx` (new)

**Acceptance Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Test coverage ≥80% for new code
- [ ] `npm --prefix ./web run test` passes

**Dependencies:** FEAT-PROV-2.1, FEAT-PROV-2.2, FEAT-PROV-2.3, FEAT-PROV-2.4

---

### FEAT-PROV-3.4 — Verification Baseline ✅

**Status:** Complete  
**Objective:** Run full project verification baseline to ensure no regressions.

**Implementation Details:**
- Run `make verify`:
  - `go test ./cmd/... ./internal/...` (backend tests)
  - `npm --prefix ./web ci` (install deps)
  - `npm --prefix ./web run test` (vitest)
  - `npm --prefix ./web run lint` (eslint)
  - `npm --prefix ./web run typecheck` (tsc --noEmit strict)
  - `npm --prefix ./web run build` (vite production build)
- All steps must pass with no errors

**Acceptance Criteria:**
- [ ] `make verify` passes completely
- [ ] No new lint errors
- [ ] No new type errors
- [ ] Production build succeeds

**Dependencies:** FEAT-PROV-3.2, FEAT-PROV-3.3

---

### FEAT-PROV-3.5 — Documentation Updates ✅

**Status:** Complete  
**Objective:** Update all relevant documentation.

**Implementation Details:**
- Update `docs/SPECIFICATION.md`:
  - §3 Use Cases: Update UC-8 to mention provider detail view
  - §5 API Surface: Document new endpoints
- Update `docs/ARCHITECTURE.md`:
  - §7 Frontend: Document new `ProviderDetailPage` route
- Update `docs/CODE_GRAPH.md`:
  - §7 API Surface Map: Add provider endpoints
- Update `session.md`:
  - Record implementation decisions and results
- Update `CHANGELOG.md`:
  - Add entry under `[Unreleased]`

**Files Affected:**
- `docs/SPECIFICATION.md`
- `docs/ARCHITECTURE.md`
- `docs/CODE_GRAPH.md`
- `session.md`
- `CHANGELOG.md`

**Acceptance Criteria:**
- [ ] Spec accurately describes new feature
- [ ] Architecture reflects new structure
- [ ] Code graph includes new components
- [ ] Session log updated
- [ ] Changelog entry complete

**Dependencies:** All feature tasks complete

---

## Summary & Open Questions

### Summary of Phases

| Phase | Description | Tasks | Status |
|-------|-------------|-------|--------|
| Phase 1 | Backend — Provider Grouping & Key Listing API | 4 | ✅ Complete |
| Phase 2 | Frontend — Provider Table Redesign | 4 | ✅ Complete |
| Phase 3 | Internationalization, Testing & Documentation | 5 | ✅ Complete |

**Total Tasks:** 13  
**Completed:** 13  
**Remaining:** 0

> **⚠️ Protected feature:** This phase is a **permanent core feature** and is
> protected from removal or replacement by upstream merges. See
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) PF-06 for the complete policy.

---

### Open Questions & Decisions

**Q1: Provider Detail — Full Page or Drawer/Modal?**  
**Decision:** Full page (`/ai-provider/:providerId`) per user request.

**Q2: API-key Viewer Scoping — See All Providers or Only Own?**  
**Decision:** API-key viewers see only providers with keys they own.

**Q3: Success Rate Sorting — Provider Level or Key Level?**  
**Decision:** Both. Provider list can sort by aggregate success rate; detail page sorts keys by individual success rate.

**Q4: URL Structure for Detail Page?**  
**Decision:** `/ai-provider/:providerId` (provider name URL-encoded)

---

### Known Limitations & Future Work

- Provider-level success rate: Aggregate calculation may not reflect individual key performance
- Empty provider filtering: Handle gracefully when API-key viewer has keys but provider has no metadata
- Provider naming: Assumes provider names are unique and stable
- Mobile optimization: Detail page table may need responsive layout adjustments

---

## Related Documents

- [SPECIFICATION.md](docs/SPECIFICATION.md) — Functional contract
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) — System architecture
- [CODE_GRAPH.md](docs/CODE_GRAPH.md) — Codebase structure
- [CONTRIBUTING.md](docs/CONTRIBUTING.md) — Development workflow
- [session.md](session.md) — Session context and decisions
- [CHANGELOG.md](CHANGELOG.md) — Notable changes

---

**End of Roadmap**
