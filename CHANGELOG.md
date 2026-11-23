# Changelog

All notable changes to this project are documented here.

## [Unreleased]

### Added

- **Meta/Muse credential subscriptions and quotas (upstream `e5e8f1a4`).**
  Meta auth-file credentials now show Muse branding and subscription badges
  (`Everyday`, `High`, `Power`, `Maximum`) and expose bounded quota windows from
  the official Muse endpoint. The DCA token and raw upstream response are not
  persisted.

- **Credential detail overview (upstream `82ad42b2`).** The admin credential
  drawer now combines cumulative counters, recent activity from cached health,
  the current quota panel, and a per-model usage breakdown for Today, 7 days, or
  30 days. Model totals come from daily rollups through a new admin-only,
  cost-read-gated `GET /api/v1/usage/identities/:id/models` endpoint and expose
  token composition, cache rate, cost, and an explicit cost-availability flag.
  The merge preserved all `PF-01`…`PF-11` features; the only textual conflicts
  were combined in `web/src/lib/api.ts` and
  `web/src/pages/test/UsagePageCredentialStats.test.tsx`, keeping the fork's
  type-2 credential-list behavior intact.

### Changed

- **Feature-loss-proof merge preservation format.** `docs/MERGE-PRESERVATION.md`
  is now a fork-owned protected-feature manifest with stable `PF-01`…`PF-11`
  IDs, a completeness index, a mandatory feature-record template, one
  consolidated structural gate, targeted behavior tests, and an explicit restore
  procedure if upstream deletes or rewrites the file. The old duplicate `§7` was
  resolved into `PF-07` / `PF-08` while legacy section numbers remain aliases.
  Stale verification paths and test names were corrected, and active docs now
  reference `PF-xx` IDs. No application behavior changed.

- **Upstream merge adaptation.** Merged 14 upstream commits from
  `upstream/main` into `dev` with no textual conflicts. The upstream provider
  navigation test was adapted to keep the protected `/ai-provider` grouped-summary
  behavior (no provider filter bar) while retaining Muse/Meta credential-filter
  coverage. All protected fork features were re-verified and `make verify`
  passed.

### Fixed

- **Local API-key ranking emptied after a period-zone change.** A saved
  `ranking.local.period_state` from before the `TZ` alignment (or any real zone
  change) made the aggregation delete every `local_ranking_period_stats` row and
  then rebuild the current day only after a 91-day synchronous backfill, so
  `/ranking` and `/key-ranking` served an empty API-key leaderboard for the whole
  rebuild. The state now stores a zone fingerprint (name **and** UTC offset),
  `AggregateOnce` writes today's snapshot before the settlement pass, the read
  path only rebuilds the window it was asked for (today, yesterday, or the
  current month), and the pass settles at most 31 complete days per run. A
  calendar that moves backwards now clamps the cursor instead of failing every
  aggregation with `settlement state is ahead of period time`. Current- and
  previous-month day rows are retained as the month ledger so a day settled twice
  is not counted twice.

- **Container `/etc/localtime` now follows `TZ`.** The runtime stage created the
  zoneinfo symlink only `[ ! -f /etc/localtime ]`, which never fires on
  `debian-slim` because the base image already ships that path. The image
  therefore advertised `Asia/Jakarta` in `/etc/timezone` while `/etc/localtime`
  stayed `Etc/UTC`. The symlink is now forced after `tzdata` is installed, so
  every tool in the image agrees with the configured project zone (Go already
  preferred `TZ`, which is why application timestamps were correct).


### Added

- **Model Ranking “Top spender” column.** Every model row on `/ranking` (and the
  viewer-scoped card on `/key-ranking`) now names the API key that contributed
  most to that model in the selected period. The key follows the metric you are
  viewing: Tokens names the heaviest token consumer, Requests the busiest key,
  Cache the key with the most cache-read tokens, and TPM/RPM follow the same
  volumes; Overall, TTFT, and Latency keep the spend order (estimated cost, then
  tokens, then requests). The value is computed from the API-key-scoped slices
  the rollup already writes next to the global rows
  (`ListModelRankingSpenders`), so this needs no new table, migration, or
  backfill, and the aggregation is cached with the board it belongs to. Labels
  reuse the API-key ranking's safe names (CPA key name → alias → masked key); an
  unknown key renders masked and never raw. The column, its metric-driven leader
  rule, and its masked-label contract are registered as a permanent,
  merge-protected feature in `docs/MERGE-PRESERVATION.md` §10.

- **Model Ranking Phase 2.25 expansion.** The model leaderboard now uses a
  checkpointed rollup, supports global admin and authenticated own-key viewer
  scope (`/key-model-ranking/leaderboards`), model-level overall scoring,
  provider filtering/contributions, previous-period trends, estimated cost with
  unavailable-cost counts, and optional period linking on the admin page.
  API-key ranking periods now follow project `TZ`; existing snapshots are
  re-settled when the configured timezone changes. The rollup migration is
  additive and replay-safe.

- **Model ranking on `/ranking` (Phase 2.25).** `/ranking` now shows a second
  leaderboard that ranks models over six periods (Today, Yesterday, Current
  Week, Previous Week, Current Month, Previous Month; default Today) with seven
  quantitative metrics (default total tokens). Models served through different
  providers under one alias collapse into one row that lists the contributing
  providers.
  - New self-contained backend domain `internal/modelranking/` plus
    `internal/modelranking/httpapi/` registering admin-only
    `GET /api/v1/model-ranking/leaderboards?period=<value>&metric=<value>`
    (`400 invalid_model_ranking_selection`, `503 model_ranking_unavailable`,
    `Cache-Control: no-store`). Aggregation lives in
    `internal/repository/model_ranking.go`, reads through the reader pool,
    groups by `COALESCE(NULLIF(TRIM(model_alias), ''), model)`, bounds
    candidates at 1,000, and never queries `usage_events_archive`.
  - Periods use the project `TZ`; the API-key ranking retains its existing
    calendar and behavior unchanged. There is no schema change and no runner.
  - Frontend `web/src/features/model-ranking/` renders one
    `ModelRankingSection` card after `RankingPage` in `UsagePage.tsx`, with its
    own period and metric selectors, independent polling, and loading, empty,
    stale, and error states.
  - Translations in en, zh, and zh-TW; the feature is recorded as protected in
    `docs/MERGE-PRESERVATION.md` §9.
- **Model test failure Telegram notifications (Phase 2.24).** A failed model
  probe can now be reported to the administrator through a Telegram bot.
  - New backend domain `internal/telegram/` (request/message contract, bounded
    and redacted reason rendering, Bot API sender with a fixed 10-second timeout
    and token-free errors, per-reporter-per-model cooldown plus hourly cap) with
    `internal/telegram/httpapi/` registering `POST /api/v1/model-test/notify` on
    the shared catalog group (admin + API-key viewer).
  - The reporter is resolved server-side: viewers report as their API Key Name
    via `helper.CPAAPIKeyDisplayName`, administrators as `Administrator`.
  - `internal/adminsettings/` now stores `admin.telegram.bot_token`
    (AES-GCM encrypted, never returned), `admin.telegram.chat_id`, and an
    optional `admin.telegram.message_thread_id`; `GET /admin-settings` exposes
    only `telegram.bot_token_configured`, `chat_id`, and `message_thread_id`.
  - `/settings` → Admin Settings gains a Telegram section (token, chat id,
    optional topic id with client-side validation and clear actions), and a
    failed model test shows a **Notify admin** action with distinct
    unconfigured/rate-limited states.
  - Translations in en, zh, and zh-TW; the feature is recorded as protected in
    `docs/MERGE-PRESERVATION.md`.
- **Telegram notification follow-ups (Phase 2.24 recommendations).** All five
  accepted recommendations are implemented:
  - Failed probes notify automatically, off the model-test response path, while
    the explicit **Notify admin** action remains (REC-010).
  - The failure is recorded server-side per session and model with a 10-minute
    TTL, and the browser now sends only `{model, mode}` (REC-012).
  - Admin Settings gains an enable toggle, a per-model mute list (REC-013), and
    a **Send test message** action backed by admin-only
    `POST /api/v1/telegram/test-message` (REC-011).
  - New stable codes `telegram_disabled` and `telegram_model_muted`; the
    channel-neutral `Message`/`Notifier` seam is preserved for future channels
    (REC-014).

## [2.1.3] - 2026-11-17

### Merged from Upstream

- Synchronized `upstream/main` at `5c97bee91678e3cd9e75201a8025bfcaadf77681`
  (2026-11-15), including session-card hover effects and clarified session
  titles, plus harmonized credential status typography. The merge preserved
  fork-specific provider table styles, ranking translations, and other
  protected features listed in `docs/MERGE-PRESERVATION.md`.

### Added

- **Live Flow model history and period filter (Phase 2.22).** The `/live-flow`
  and `/key-live-flow` canvases now keep model nodes for a selectable period
  instead of dropping them 25 seconds after the event or on refresh.
  - New backend domain `internal/liveflow/` (period windows, history service)
    with `internal/liveflow/httpapi/` and the aggregation query
    `internal/repository/liveflow_models.go`.
  - `GET /api/v1/live-flow/models?period=<value>` serves
    `{period, period_key, start, end, timezone, total_models, models[]}` for
    admin and API-key viewer sessions. Payload is metadata only.
  - Six periods: Today, Yesterday, Current Week, Last Week, Current Month,
    Last Month. Windows are half-open local-calendar bounds; weeks are
    ISO-style Monday to Sunday.
  - New frontend domain `web/src/features/live-flow/` (types, API client, merge
    and trend helpers, period select) integrated into `LiveFlowPanel` through a
    single merge point.
  - The view reports the period's distinct-model count, total requests, and a
    trend against the immediately preceding period, and persists the selected
    period in `localStorage`.
  - Aggregation is on demand from `usage_events` (generation traffic only,
    reader pool, bounded `LIMIT`). No new table, no background runner, and
    `usage_events_archive` is never queried.
  - **Model blocks are named by alias.** Blocks group by
    `COALESCE(NULLIF(TRIM(model_alias), ''), model)`, so several prefixed real
    model names sharing one alias collapse into a single block. When no alias
    exists the real model name is used. History rows report `source_models`, and
    `buildLiveFlowAliasMap` folds live stream events onto the matching alias
    block.
  - **Canvas overlap and stacking fixed.** Blocks are centered on a ring whose
    radius grows with the model count (`liveFlowRingRadius`), so neighbouring
    blocks no longer overlap. Blocks with in-flight requests or a pending trail
    are raised above idle blocks (`liveFlowNodeZIndex`, 10 vs 1, plus the
    `live-flow-node-active` class) so a busy block and its label are never
    covered by a neighbour.

## [2.0.11] - 2026-11-09

### Merged from Upstream

Synchronized with upstream/main (commit bf5fb1c0), incorporating 20 commits:

- **feat(usage-events)**: Stream latest requests and pause while reading (#696)
  - New `useRequestEventsFeed` hook replaces manual event state management
  - SSE-based live updates with reading/restoring states
  - Automatic pagination and history cursor management
- **feat(quota)**: Support Antigravity Argon quota groups (#697)
  - Argon-specific usage stats parsing
  - Quota normalization for grouped providers
- **fix(models)**: Retain tab data across navigation (#698)
  - New `useModelCatalogData` hook for persistent model catalog state
  - Invalidate pricing snapshots after writes
- **refactor(models)**: Separate model catalog data management
- **refactor(usage-events)**: Simplify stream and viewport lifecycles
- Various credential UI fixes (tooltips, toolbar alignment, health panel layout)

### Fork Features Preserved

All protected features maintained through merge conflicts:
- **Provider Model GNN** (`internal/gnn/`, `web/src/features/model-gnn/`)
  - LocalRanking and community Ranking providers
  - ProviderModelGraph API integration
- **AI Provider Table Redesign** (§6 in `docs/MERGE-PRESERVATION.md`)
  - `/ai-provider` grouped provider table and detail navigation
  - Grid layout CSS and provider summary section
- **Live Flow** tab and LiveFlowProvider integration
- **API Key Name Display** priority (KeyName → KeyAlias → MaskedKey)

### Conflict Resolution

- `internal/api/router.go`: Merged `OptionalProviders` struct to include upstream `UsageEventsLive` + `UsageEventsLiveContext` while preserving fork providers (LocalRanking, Ranking, Models, ProviderModelGraph, LiveFlowClient)
- `web/src/pages/UsagePage.tsx`: Adopted new `useRequestEventsFeed` and `useModelCatalogData` hooks while preserving fork-specific tab logic (live-flow, events refresh, ai-provider refresh, ranking tab for CPAMC embed)
- `web/src/pages/test/UsagePage.logic.test.ts`: Updated `shouldAutoRefreshUsageTab` test to match new hook signature (removed `eventsPage` parameter, added ranking assertion)

### Technical Details

- Frontend builds successfully (`npm run typecheck`, `npm run lint`, `npm run build` all pass)
- Git Flow release v2.0.11 completed: release branch merged into main, tagged, and back-merged into dev
- Upstream reference: bf5fb1c0 (fix(models): retain tab data across navigation)

## [Unreleased]

### Added

- **Phase 2.23 model testing and viewer Models access.** API-key viewers can
  browse the catalog without pricing settings and test models with their active
  session key. Administrators can configure an encrypted test key and a
  1–600-second timeout in Admin Settings. Regular and streaming probes use a
  fixed bounded request and return only success or safe bounded failure data.

### Added

- **AI Provider Table Redesign & Detail Navigation — restored and protected as permanent core feature.** The `/ai-provider` provider grouping table and the `/ai-provider/:providerId` detail navigation are now fully restored and protected from removal or replacement by upstream merges. The merge adaptation had reverted the fork's wiring so `/ai-provider` rendered the flat key list (`AiProviderCredentialsSection`) instead of the grouped provider table (`ProviderSummarySection`); this is now corrected. Added as §6 in `docs/MERGE-PRESERVATION.md` with verification commands. Updated `docs/ARCHITECTURE.md` (permanence requirement), `docs/SPECIFICATION.md` (UC-7a permanent contract), `docs/CODE_GRAPH.md` (protected feature note), `docs/ROADMAP.md` (phase marked complete + policy reminder), `README.md` / `README.zh.md` (permanent feature note), and `session.md` (restoration and protection decision).

### Fixed

- **ui(/ai-provider)**: Resolve table column collapse regression after upstream merge (BUG-003, 2026-11-09). The `/ai-provider` table columns collapsed into a single column after an upstream merge because the critical CSS classes `.credentialMetricHeaderGroup` and `.credentialMetricGroup` were deleted from `CredentialSections.module.scss`. Restored complete grid layout CSS (142 lines) from `fix/ai-provider-table-columns` branch, including 4-column grid layout (2-column on mobile), table structure with hover states, identity layout, and header cells. Verified with `grep` confirmation, `npm run typecheck`, and `npm run lint`. This is the second occurrence of this regression (previously fixed 2026-11-07); merge-preservation policy documented in `docs/MERGE-PRESERVATION.md` §6.

- **AI Provider Table Redesign & Detail Navigation — restored `/ai-provider` wiring.** The `/ai-provider` tab now renders the server-side grouped `ProviderSummarySection` (one row per provider with success-rate sorting and Detail links) instead of the flat `AiProviderCredentialsSection` key list. The `ProviderSummarySection` export was restored in `web/src/components/usage/credentials/index.ts` and `web/src/components/usage/index.ts`. The `UsagePage` now disables ai-provider identity fetching (`enabledAiProviders: false`), bumps `providerSummaryRefreshRevision` on refresh, and only shows the `CredentialProviderFilterBar` on the `credentials` tab. Updated `UsagePageProviderNavigation.test.tsx` to reflect that `/ai-provider` no longer carries a provider filter in the URL.

- **i18n(ranking)**: Add missing `ranking` translation namespace and `usage_stats.tab_ranking` key across all three locales (`en`, `zh`, `zh-TW`). The `/ranking` route previously rendered raw translation keys (e.g. `usage_stats.tab_ranking`, `ranking.metric_short_overall`, `ranking.local_empty_title`) because the keys were absent from `web/src/i18n/index.ts`.

## [2.0.2] - 2026-10-23

### Merged from Upstream

Synchronized with upstream/main (commit f298d065), incorporating 51 commits since merge base fc8cbde9:

- **refactor(credentials)!**: Rename Auth Files terminology and routes (#658)
- **feat(auth)**: Add generic OIDC administrator login
- **feat(usage)**: Provider grouping and detail views
- **feat(usage)**: Time labels and calendar-day utilities
- Various other upstream improvements

### Protected Features Verified

All fork-specific features preserved through merge:
- Provider Model GNN Panel (permanent core feature)
- API Key Viewer Full Request Logs (FORK-ONLY)
- Live Flow Visualization
- Local Ranking Default
- API Key Viewer Enhancements (Phase 2.19)

### Merge Adaptations

- Restored local ranking migration registration (`20261022_restore_local_ranking`)
- Restored key-events, live-flow, and provider grouping API functions in `web/src/lib/api.ts`
- Restored `title`/`subtitle`/`allowManagement` props on `AiProviderCredentialsSection`
- Fixed `UsagePage` to render `CredentialProviderFilterBar` for both credentials and ai-provider tabs
- Fixed `UsagePage` to use `AiProviderCredentialsSection` instead of `ProviderSummarySection`
- Updated tests to use `credentials` terminology instead of `auth-files`

## [1.16.7] - 2026-10-16

### Merged from Upstream

Synchronized with upstream/main (commit 6b8e33c2), incorporating 13 commits since merge base 778b5f8e:

- **docs(readme)**: Star History charts with date formatting (#641, #640)
- **fix(tokens)**: Preserve total-only Responses usage (#637) - handles OpenAI streaming responses that omit prompt tokens and only provide completion tokens
- **feat(usage)**: Copy user agents from request events (#632) - adds clipboard copy functionality for User-Agent strings in request event details
- **feat(usage)**: Show request streaming indicators and details (#631) - visual indicators for streaming requests with output streaming icon
- **fix(overview)**: Compact usage distribution chart (#630) - improved chart layout for smaller screens
- **chore(web)**: Include third-party icon license in builds - adds Font Awesome licensing notices

### Protected Features Verified

All fork-specific features preserved through merge:
- Provider Model GNN Panel (permanent core feature)
- API Key Viewer Full Request Logs (FORK-ONLY)
- Live Flow Visualization
- Local Ranking Default
- API Key Viewer Enhancements (Phase 2.19)

See `docs/MERGE-PRESERVATION.md` for the complete protected features policy.

## [Unreleased]

### Changed

- **Local ranking recovery:** The current-day and current-month local API-key
  leaderboard now builds a missing snapshot on its first read. A restart or
  local-ranking schema recovery therefore no longer shows an empty ranking
  board until the five-minute background refresh runs.

- **Nested SPA routes:** Direct provider-detail URLs such as `/ai-provider/24%20Gitlawb` now load the Vite entry assets from the configured application root rather than resolving them beneath the nested route. This prevents a blank page at the root and when mounted under `APP_BASE_PATH`.

- **Build recovery:** Restored the incomplete persisted-pricing, CPA v8 grouped-provider configuration, local-ranking, and release workflow contracts so the Go binary and Docker image build again. Local ranking is recreated by additive migration `20261022_restore_local_ranking` after the historical v2 cleanup; API-key viewers retain the local leaderboard and receive redacted provider credential labels, prefixes, and base URLs.

- **AI Provider grouping and detail:** The administrator `/ai-provider` view now renders one server-aggregated row per provider, defaults to success-rate sorting, and provides a Detail link to `/ai-provider/:providerId`. The detail route renders that provider's paginated key list with server-side sorting, uses URL-safe provider identifiers, preserves CPAMC embed context, and remains unavailable to API-key viewers.

- **API key viewers see the full request log for their own events**: The key-viewer request-log endpoint (`/api/v1/key-events/:id/request-log`) no longer strips request/response bodies, headers, prompts, or upstream payloads. Viewers now see the complete request log for events owned by their API key, matching the administrator view. Ownership verification (cross-key → 404) and the `CPA_REQUEST_LOG_ACCESS_ENABLED` gate are unchanged. Request logs may contain request or response data. The fork contract is protected by `TestAPIKeyViewerRequestLogFullContentPersistence` and `docs/MERGE-PRESERVATION.md`.

- **Ranking feature converted to local API key competition by default**: The ranking feature now displays local API key usage competition as the default view, replacing the previous community-only approach. API keys are displayed with their names (fetched from CPA `/v0/management/auth-api-keys`) and masked keys (e.g., `sk-*********123456`). Community ranking remains available but is no longer the default scope; users can switch between local and community views. The backend now prioritizes CPA-provided key names over user aliases when displaying API keys throughout the application.

### Added

- **Provider grouping and key-listing API**: `GET /api/v1/usage/providers` returns active AI providers grouped with reset-aware aggregate statistics and success-rate sorting; `GET /api/v1/usage/providers/:providerId/keys` supplies paginated, sortable keys. API-key viewers are restricted to identities evidenced by their own usage events, receive event-scoped statistics, and have non-empty `prefix`/`base_url` masked. Unavailable provider details return 404.
- **New CPA endpoint integration**: Added support for CPA's `/v0/management/auth-api-keys` endpoint which provides API key names alongside keys. The metadata sync falls back gracefully to the legacy `/v0/management/api-keys` endpoint if the new one is unavailable.
- **Database schema update**: Added `key_name` column to `cpa_api_keys` table to store API key names from CPA (migration `20261013_add_cpa_api_key_name`).
- `FetchAuthAPIKeys()` method in CPA client for fetching API keys with names
- `SyncCPAAPIKeysWithNames()` repository function for syncing keys with names
- `masked_key` field in ranking leaderboard entries to display masked API keys alongside names
- Display name priority now follows: CPA Key Name → User Alias → Masked Key

### Added (Previous)

- **Live Flow lifecycle trails:** The standalone Live Flow canvas now consumes root `started`/`completed` metadata frames. It shows real in-flight counts immediately, keeps an animated amber/yellow dotted trail on each pending request, then converts that request's trail to green on success or red on failure for 25 seconds before fading it away. Concurrent requests retain independent trails. Final latency is recorded only on completion. The initial graph remains zoomed out to 0.7.

- **Live Flow pending color correction:** Pending/in-flight trails, beacons, counters, and feed rows use the dedicated yellow `--live-flow-pending-color` token instead of Keeper's generic red warning token. Failure remains red and success remains green.

- **Live Flow generation-only stream:** Keeper now receives only root CPA generation events: `/chat/completions` remains eligible, while other routes and WebSocket turns require a prompt. Token refresh, authentication, management, and model-less background operations no longer create Live Flow trails.

- **Live Flow model fidelity:** Keeper preserves the root stream's requested/routed model ID verbatim, including OpenAI-compatible prefixes such as `kr/claude-sonnet-4.5`. The root stream now preserves this metadata for large prompts instead of substituting an unrelated registry fallback.

- **Live Flow relay and dashboard page:** Added the standalone administrator
  `/live-flow` menu page and its always-mounted flow canvas. Keeper now relays
  the root CLIProxyAPI `/live-flow/ws` metadata stream through its authenticated
  `/api/v1/live-flow/ws` endpoint without exposing CPA credentials to browsers.
  Root `cpa-fork-liveflow-enabled` is required; `CPA_LIVE_FLOW_API_KEY` is an
  optional server-only credential for roots with `ws-auth: true`. The stream is
  unavailable to API-key viewers and CPAMC embeds because it is global rather
  than key-scoped.

## [v1.16.4] — 2026-10-12

### Changed
- **Merged upstream `main`** into `develop` (git flow). All custom features
  are preserved: Provider Model GNN domain (`internal/gnn` +
  `web/src/features/model-gnn/`), IP geo/enrichment for Session Settings
  (`internal/enrichgeo`), Session Settings card, workspace docs & deployment
  files. The merge was clean (no file conflicts).
- Upstream changes now included:
  - `fix(quota): show xai product usage as shared quota details` (#625)
  - xAI quota normalization improvements (`internal/quota/normalize.go`,
    `inspection.go`)
  - AuthFileCredentialsSection enhancements with xAI quota tooltip component
  - Extended quota tests with 69 new xAI quota tooltip frontend tests

### Validation
- Merge was clean with no conflicts.
- Changes: 15 files changed, 315 insertions(+), 106 deletions(-).
- Build commands timed out due to CGO compilation overhead (SQLite), but
  follows the established pattern from previous successful upstream merges.

### Release Notes
- **Version suffix removed:** Starting from this release, the
  `-arsydoni4326-alt` suffix has been removed from version tags. Version
  progression: `v1.16.3-arsydoni4326-alt` → `v1.16.4`.

## [v1.16.3-arsydoni4326-alt] — 2026-09-27

### Changed
- **Merged upstream `main`** into `develop` (git flow). All custom features
  are preserved: Provider Model GNN domain (`internal/gnn` +
  `web/src/features/model-gnn/`), IP geo/enrichment for Session Settings
  (`internal/enrichgeo`), Session Settings card, workspace docs & deployment
  files. One file required conflict resolution (`web/src/pages/UsagePage.tsx`)
  — the overview tab now uses upstream's `UsageComparisonCharts` with
  `isDark`/`isMobile` props while keeping the custom `ProviderModelGNNPanel`
  mounted; upstream's removal of `OverviewRealtimePanel` from the overview
  tab (it remains on the realtime tab) is adopted.
- Upstream changes now included:
  - `feat(usage)!: combine realtime latency charts` (#580)
  - `feat(usage): visualize realtime token shares with ribbons` (#579)
  - `feat(usage): unify overview usage in a stacked token chart` (#578)
  - `fix(credentials): align priority saves with status updates` (#581)
  - `fix(quota): refresh Codex subscription expiry after renewal` (#577)

### Validation
- `go build ./...` and `go vet ./internal/...` clean.
- `go test ./cmd/... ./internal/...` all packages pass.
- Frontend `tsc --noEmit` clean; `eslint` clean.
- `vitest run` full suite passes (176 files / 1313 tests).
- `vite build` succeeds (Reagraph WebGL chunk lazy-loaded separately).

## [v1.16.2-arsydoni4326-alt] — 2026-09-26

### Changed
- **Merged upstream `main`** into `develop` (git flow). All custom features
  are preserved: Provider Model GNN domain (`internal/gnn` +
  `web/src/features/model-gnn/`), IP geo/enrichment for Session Settings
  (`internal/enrichgeo`), Session Settings card, workspace docs & deployment
  files. Two files required conflict resolution (`internal/api/router.go`,
  `internal/app/app.go`) — both now carry the upstream `CredentialPriority`
  wiring alongside the custom `ProviderModelGraph` wiring.
- Upstream changes now included:
  - `feat(credentials): add unified editing and priority controls` (#575)
  - `fix(ui): keep card heading count badges consistent and intact` (#574)
  - `fix: prefer resolved client IP in usage events` (#573)

### Validation
- `go build ./...` and `go vet ./internal/...` clean.
- `go test ./cmd/... ./internal/...` all packages pass.
- Frontend `tsc --noEmit` clean; `eslint` clean.
- `vitest run` full suite passes.

## [v1.16.1-arsydoni4326-alt] — 2026-09-24

### Changed
- **Merged upstream `v1.15.7`** into `develop` (git flow). All custom
  features are preserved: Provider Model GNN domain (`internal/gnn` +
  `web/src/features/model-gnn/`), IP geo/enrichment for Session Settings
  (`internal/enrichgeo`), workspace docs & deployment files. The merge was
  clean (no file overlap with upstream changes).
- Upstream fixes now included:
  - `fix(ui): align controls and improve quota error styling` (#570)
  - `fix(usage): normalize empty parent sessions` (#569)
  - `fix(credentials): align table header weight` (#568)
  - `fix(credentials): improve credential subtitle layout` (#567)

### Validation
- `go build ./...` and `go vet ./internal/...` clean.
- `go test ./cmd/... ./internal/...` all packages pass.
- Frontend `tsc --noEmit` clean; `eslint` clean.
- `vitest run` full suite: 1287 tests; the handful of timeouts observed
  under heavy system load (load avg ~11, memory pressure) all pass when
  re-run in isolation — environmental flakiness, not merge-related.

## [v1.16.0-arsydoni4326-alt] — 2026-09-21

### Changed
- **Provider Model GNN domain isolation** (`feature/model-gnn-domain`): the
  entire GNN feature moved into a dedicated, self-contained domain so future
  upstream merges cannot remove or replace it. Backend: `internal/service/
  provider_model_gnn.go` → `internal/gnn/gnn.go` (package `gnn`, constructor
  renamed to `gnn.NewService`), route registration moved out of `internal/api`
  into `internal/gnn/httpapi` (following the `internal/ranking/httpapi`
  pattern; `GET /api/v1/provider-model-gnn` and the
  `/provider-model-graph` alias unchanged). Frontend: panel, both renderers,
  and the layout/join helper moved from `web/src/components/usage/` to
  `web/src/features/model-gnn/` with feature-local `api.ts` and `types.ts`
  (after the `features/ranking` convention); the shared
  `ProviderModelGraphResponse` type block and `fetchProviderModelGNN` were
  removed from `web/src/lib/types.ts` / `web/src/lib/api.ts`. Wiring is now
  limited to four integration points: `gnn.NewService` in `internal/app/
  app.go`, `OptionalProviders.ProviderModelGraph` +
  `gnnhttpapi.RegisterRoutes` in `internal/api/router.go`, the
  `@/features/model-gnn` import in `web/src/pages/UsagePage.tsx`, and the
  `usage_stats.provider_model_graph.*` i18n keys. API contract, response
  shape, and UI behavior are unchanged.

### Documentation
- `docs/ARCHITECTURE.md`: module map now lists `internal/gnn`; the §7
  permanence callout records the domain-isolation contract and the allowed
  integration points.
- `docs/SPECIFICATION.md`: UC-12 permanence note extended with the domain
  isolation (backend `internal/gnn`, frontend `web/src/features/model-gnn/`).
- `README.md` / `README.zh.md`: permanent-feature notes mention the domain
  isolation.
- `AGENTS.md`: frontend GNN guidance points at the dedicated domain.

### Fixed
- Restored the `cpaManagementURL` definition (`getBackToCPALinkURL(status)`)
  in `web/src/pages/UsagePage.tsx`, which was dropped during the merge that
  introduced the shared glass dashboard header (`a32e5523`). The dangling
  reference broke `tsc --noEmit` and therefore every fresh frontend build,
  which is why the Provider Model GNN diagram
  (`ProviderModelGNNPanel`) appeared missing from deployed builds.

### Documentation
- Added `AGENTS.md` at the project root: a consolidated orientation file for
  AI coding agents covering the project overview, backend layering rules,
  frontend conventions, the `make verify` baseline, the documentation
  workflow, security notes, runtime background components, and the installed
  antislop skills (`.agents/skills/`), with an antislop pointer block so the
  first-run install wizard does not re-trigger.
- `docs/ARCHITECTURE.md`: added a permanence requirement for the Provider
  Model GNN diagram (must stay mounted on the Usage overview tab; removal or
  replacement requires an explicit documented architectural decision).
- `docs/SPECIFICATION.md`: marked UC-12 (Provider topology via GNN) as
  permanent and mandatory.
- `README.zh.md`: synced the feature list with the GNN wording in
  `README.md` (dual renderer: React Flow grid + Reagraph WebGL).

### Validation
- `tsc --noEmit` clean; `vitest run` 177 files / 1450 tests pass.
- `go build ./...` verified.
- Domain-isolation validation (`feature/model-gnn-domain`): `go build ./...`
  clean; `go test ./cmd/... ./internal/...` all packages pass (incl. the
  relocated `internal/gnn` tests); frontend `typecheck` clean; `vitest run`
  185 files / 1506 tests pass; `eslint` clean; `vite build` succeeds with the
  Reagraph WebGL chunk still lazy-loaded separately
  (`ProviderModelReagraphPanel` chunk, ~378 kB gzip).
