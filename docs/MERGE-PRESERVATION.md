# Merge Preservation Policy

This document lists features and architectural decisions that **must not be
removed or replaced** during upstream merges from the original CLIProxyAPI
Usage Keeper repository. These features were introduced in this fork since
commit `bcc2d4c4b28e4ec6e8e6367b0831a313a4347f8c` and represent core functional
requirements for this deployment.

## Policy Statement

Any removal, replacement, or significant modification of the features listed
below requires:

1. An explicit, documented architectural decision
2. Approval from the maintainers
3. Updates to all affected documentation:
   - `docs/ARCHITECTURE.md`
   - `docs/SPECIFICATION.md`
   - `docs/ROADMAP.md`
   - `README.md` / `README.zh.md`
   - `session.md`
   - This file (`docs/MERGE-PRESERVATION.md`)

## Protected Features

### 7. Model Test & API-key Viewer Models Access

The authenticated `/models` catalog and per-model test controls must remain
available to API-key viewers without exposing pricing settings. The server
selects credentials by role: the active viewer key for viewers and the
encrypted Admin Settings key for administrators. Probes use the configured CPA
base URL, a fixed bounded prompt, a maximum output budget, and a 1–600 second
timeout.

- Backend: `internal/modeltest/`, `internal/modeltest/httpapi/`,
  `internal/adminsettings/`, `internal/cpa/client.go`
- Frontend: `web/src/features/model-test/`, `web/src/pages/KeyModelsPage.tsx`,
  `web/src/components/usage/AdminSettingsCard.tsx`, and `ModelCatalog.tsx`
- Routes: authenticated `GET /api/v1/models`, `POST /api/v1/model-test`, and
  admin-only `GET/PUT /api/v1/admin-settings`
- Security: raw credentials are never returned; viewer requests cannot choose a
  credential, host, path, or header.

### 8. Model Test Failure Telegram Notifications

**Status:** Permanent (Phase 2.24)  
**Domain:** `internal/telegram/` + `internal/telegram/httpapi/`

A user whose model test failed can notify the administrator through a configured
Telegram bot. The message reports the reporter as the **API Key Name** (never
the key value), the tested model, and a bounded failure reason. Telegram is
configured in `/settings` → **Admin Settings** with an encrypted Bot Token, a
Chat ID, and an optional Topic ID (`message_thread_id`).

- **Backend:** `internal/telegram/` (contract, rendering, Bot API client,
  limiter), `internal/telegram/httpapi/` (route registration),
  `internal/adminsettings/` (`admin.telegram.bot_token`,
  `admin.telegram.chat_id`, `admin.telegram.message_thread_id` settings)
- **Frontend:** `web/src/features/model-test/` (notify action),
  `web/src/components/usage/AdminSettingsCard.tsx` (Telegram section)
- **Routes:** `POST /api/v1/model-test/notify` (admin + API-key viewer),
  `POST /api/v1/telegram/test-message` (admin only), extended
  `GET/PUT /api/v1/admin-settings` (admin only)
- **Wiring:** `internal/app/app.go` (`telegram.NewService` +
  `telegramSettingsSource` adapter), `internal/api/router.go`
  (`OptionalProviders.Telegram` + `telegramhttpapi.RegisterRoutes` /
  `RegisterAdminRoutes`), `internal/api/telegram_wiring.go` (session-derived
  reporter/scope and the automatic-send observer on the model-test route)
- **Security invariants:**
  - The bot token is encrypted at rest with the existing AES-GCM key derived
    from `CPA_MANAGEMENT_KEY`; no endpoint returns it. `GET /admin-settings`
    exposes only `telegram.bot_token_configured`.
  - The reporter is resolved server-side from `activeAPIKeyViewerContext` +
    `helper.CPAAPIKeyDisplayName`; administrators report as `Administrator`.
    The request body has no reporter field.
  - Failure content is server-authored: the browser sends only `model` and
    `mode`, and the reason/status/error code come from the in-memory,
    per-session failure record (10-minute TTL). The reason is re-bounded to 240
    characters, control characters are stripped, and key-like substrings are
    redacted before delivery.
  - Server-enforced per-reporter-per-model cooldown plus a per-reporter hourly
    cap; `429` carries `notify_rate_limited` and `Retry-After`.
  - The `admin.telegram.enabled` toggle and the `admin.telegram.muted_models`
    list are enforced server-side (`telegram_disabled`, `telegram_model_muted`).
  - Error responses use stable codes (`telegram_not_configured`,
    `telegram_disabled`, `telegram_model_muted`, `telegram_send_failed`,
    `notify_rate_limited`, `invalid_notify_request`) and never echo the token,
    the chat id, or the Bot API response body.

Verification:

```bash
go test ./internal/telegram/... ./internal/adminsettings/... ./internal/api/...
npm --prefix ./web run test -- src/features/model-test src/components/usage/test/AdminSettingsCard.test.tsx
npm --prefix ./web run typecheck
make verify
```

### 9. Model Ranking on `/ranking` (Phase 2.25)

**Status:** Permanent (Phase 2.25)
**Domain:** `internal/modelranking/` + `internal/modelranking/httpapi/`

`/ranking` shows a second leaderboard that ranks **models** over six periods
(Today, Yesterday, Current Week, Previous Week, Current Month, Previous Month;
default Today) with seven quantitative metrics (default `total_tokens`). The
same model served through different providers under one alias collapses into a
single row whose `providers[]` lists the contributors. The endpoint is
admin-only because it aggregates usage across every API key. `/key-ranking`
also mounts a separate model card whose endpoint is strictly scoped to the
authenticated viewer's API group.

- **Backend:** `internal/modelranking/` (period resolution with project `TZ`,
  metric projection including model overall score, provider/cost/trend fields,
  deterministic ordering, scoped 30 s in-memory read cache),
  `internal/modelranking/httpapi/` (route registration),
  `internal/repository/model_ranking.go` +
  `internal/repository/model_ranking_rollup.go` +
  `internal/repository/dto/model_ranking.go` (reader-pool aggregation and
  checkpointed rollup over `usage_events`, grouped by
  `COALESCE(NULLIF(TRIM(model_alias), ''), model)`).
- **Schema/pipeline:** migration `20261119_model_ranking_rollups` adds
  `model_ranking_period_stats` and the checkpoint table; the existing serial
  usage aggregation runner commits rollup pages and the model cursor atomically.
  `repository.BackfillModelRankingRollups(ctx, db, now, maxPages)` is the
  bounded resumable recovery/backfill entry point.
- **Frontend:** `web/src/features/model-ranking/` (types, period helpers,
  formatting, API client, data hook, `ModelRankingSection` + styles).
- **Routes:** global admin `GET /api/v1/model-ranking/leaderboards?period=<value>&metric=<value>` and viewer-scoped `GET /api/v1/key-model-ranking/leaderboards?period=<value>&metric=<value>`
  with `Cache-Control: no-store`, `400 invalid_model_ranking_selection`, and
  `503 model_ranking_unavailable`.
- **Mount points:** `<ModelRankingSection />` immediately after
  `<RankingPage />` inside `activeTab === 'ranking'` in
  `web/src/pages/UsagePage.tsx`, and a viewer-scoped card in
  `web/src/pages/KeyRankingPage.tsx`.
- **Wiring:** `internal/api/router.go` (`OptionalProviders.ModelRanking` +
  `modelrankinghttpapi.RegisterRoutes(adminProtected, ...)`) and
  `internal/app/app.go` (`modelranking.NewService`, no runner).
- **i18n namespace:** `model_ranking` in `web/src/i18n/index.ts` for en, zh,
  and zh-TW.
- **Invariants:** metadata only — no prompts, request bodies, headers, or
  credentials; alias matching is exact after `TRIM`; the reader pool is used;
  `usage_events_archive` is never queried by online ranking; rollup rows and
  cursor advance atomically; viewer scope is session-derived only; missing cost
  is not zero; both ranking domains use project `TZ`; the API-key leaderboard
  remains regression-tested.

Verification:

```bash
go test ./internal/modelranking/... ./internal/repository/... ./internal/api/...
npm --prefix ./web run test -- src/features/model-ranking src/i18n/test/modelRankingTranslations.test.ts
npm --prefix ./web run typecheck
grep -q "modelrankinghttpapi.RegisterRoutes(adminProtected" internal/api/router.go && echo "✓ route"
grep -q "ModelRankingSection" web/src/pages/UsagePage.tsx && echo "✓ mount"
grep -q "key-model-ranking/leaderboards" internal/modelranking/httpapi/routes.go && echo "✓ viewer route"
grep -q "model_ranking_period_stats" internal/entities/model_ranking_period_stat.go && echo "✓ rollup schema"
grep -q "model_ranking: {" web/src/i18n/index.ts && echo "✓ i18n namespace"
make verify
```

### 1. Provider Model GNN Panel

**Status:** Permanent core feature  
**Commit baseline:** Since fork inception

The Provider Model GNN visualization panel must remain mounted on the Usage
overview tab. Both renderers (React Flow grid + Reagraph WebGL) must be
preserved.

- **Backend domain:** `internal/gnn/` (self-contained: service, HTTP API)
- **Frontend domain:** `web/src/features/model-gnn/` (panel, renderers, API client, types)
- **API endpoints:** `GET /api/v1/provider-model-gnn` (canonical), `/provider-model-graph` (alias)
- **Integration points:**
  - `internal/app/app.go`: `gnn.NewService` constructor
  - `internal/api/router.go`: `OptionalProviders.ProviderModelGraph` + `gnnhttpapi.RegisterRoutes`
  - `web/src/pages/UsagePage.tsx`: `ProviderModelGNNPanel` import from `@/features/model-gnn`
  - i18n keys: `usage_stats.provider_model_graph.*` (en/zh/zh-TW)

**Documentation:**
- `docs/ARCHITECTURE.md` §7 (permanence requirement)
- `docs/SPECIFICATION.md` UC-12 (permanent contract)
- `docs/ROADMAP.md` (policy reminder)
- `README.md` / `README.zh.md` (permanent feature note)
- `AGENTS.md` §3 (frontend GNN guidance)
- `session.md` (domain isolation decision)

**Verification after merge:**
```bash
# Verify backend domain intact
test -f internal/gnn/gnn.go && echo "✓ GNN service exists" || echo "✗ MISSING"
test -f internal/gnn/httpapi/routes.go && echo "✓ GNN HTTP API exists" || echo "✗ MISSING"

# Verify frontend domain intact
test -f web/src/features/model-gnn/ProviderModelGNNPanel.tsx && echo "✓ GNN panel exists" || echo "✗ MISSING"
test -f web/src/features/model-gnn/ProviderModelReagraphPanel.tsx && echo "✓ Reagraph renderer exists" || echo "✗ MISSING"

# Verify integration points
grep -q "gnn.NewService" internal/app/app.go && echo "✓ Service wired" || echo "✗ MISSING"
grep -q "gnnhttpapi.RegisterRoutes" internal/api/router.go && echo "✓ Routes registered" || echo "✗ MISSING"
grep -q "@/features/model-gnn" web/src/pages/UsagePage.tsx && echo "✓ Panel imported" || echo "✗ MISSING"

# Build verification
go build ./...
npm --prefix ./web run typecheck
npm --prefix ./web run build
```

---

### 2. API Key Viewer Full Request Logs

**Status:** Fork-only requirement  
**Commit baseline:** Since fork inception

API key viewers must see the **full request/response log content** for their
own events (no redaction). This includes request/response bodies, headers,
prompts, and upstream payloads.

- **Route:** `GET /api/v1/key-events/:id/request-log`
- **File:** `internal/api/key_viewer_request_events.go`
- **Marker:** `FORK-ONLY: The owning API-key viewer receives every request-log section` comment
- **Access control:** Ownership verified (cross-key → 404); requires `CPA_REQUEST_LOG_ACCESS_ENABLED=true` (disabled → 403)
- **Test:** `TestAPIKeyViewerRequestLogFullContentPersistence` compares every returned section with original title and content

**Documentation:**
- `docs/SPECIFICATION.md` (key-viewer request-log row)
- `session.md` (full-content decision)
- `README.md` / `README.zh.md` (env var `CPA_REQUEST_LOG_ACCESS_ENABLED` notes API key viewers)

**Verification after merge:**
```bash
# Verify fork marker
grep -n 'FORK-ONLY: The owning API-key viewer receives every request-log section' internal/api/key_viewer_request_events.go

# Verify no redaction logic added
! grep -q 'redactRequestLogForKeyViewer' internal/api/key_viewer_request_events.go && echo "✓ No redaction" || echo "✗ REDACTION ADDED"

# Run protection tests
go test ./internal/api/test/ -run 'TestAPIKeyViewerRequestLogFullContentPersistence|TestAPIKeyViewerRequestLogRejectsCrossKeyAccess|TestAPIKeyViewerRequestLogRejectsDisabledAccess' -count=1
```

---

### 3. Local Ranking as Default (Community Ranking UI Disabled)

**Status:** Fork-specific feature configuration  
**Commit baseline:** Phase 2.19 (October 2026)

The ranking feature displays **local API key competition by default**.
Community ranking remains in the backend but is not wired in the UI.

- **Admin page:** `web/src/pages/UsagePage.tsx` - `rankingScope` constant locked to `'local'`
- **Key viewer page:** `web/src/pages/KeyViewerPage.tsx` - `rankingScope` constant locked to `'local'`
- **Backend:** Community ranking service and endpoints remain intact but unused by UI
- **Display:** API keys shown with CPA-provided names → user aliases → masked keys

**Documentation:**
- `session.md` (community ranking UI disable decision)
- `CHANGELOG.md` (ranking feature conversion)
- `docs/ARCHITECTURE.md` §3 (local ranking as default)

**Verification after merge:**
```bash
# Verify UI locked to local scope
grep -A2 "const rankingScope.*RankingScope = 'local'" web/src/pages/UsagePage.tsx
grep -A2 "const rankingScope.*RankingScope = 'local'" web/src/pages/KeyViewerPage.tsx

# Verify no scope toggle switches
! grep -q "handleRankingScopeChange" web/src/pages/UsagePage.tsx && echo "✓ No toggle in admin" || echo "✗ TOGGLE FOUND"
! grep -q "handleRankingScopeChange" web/src/pages/KeyViewerPage.tsx && echo "✓ No toggle in viewer" || echo "✗ TOGGLE FOUND"

# Backend endpoints should still exist (unused)
grep -q "/api/v1/ranking/leaderboard" internal/api/router.go && echo "✓ Endpoints intact" || echo "✗ MISSING"
```

---

### 4. Live Flow Visualization

**Status:** Permanent feature  
**Commit baseline:** October 2026

Authenticated Live Flow relay and dashboard page showing CLIProxyAPI's
metadata-only request stream with lifecycle trails (pending/success/failure),
plus a persistent per-model history for the six selectable calendar periods.

- **Backend relay:** `GET /api/v1/live-flow/ws` (authenticated WebSocket proxy to CPA `/live-flow/ws`)
- **Backend history:** `GET /api/v1/live-flow/models?period=<value>` (admin + API-key viewer, metadata only)
- **Frontend page:** `/live-flow` standalone menu page and `/key-live-flow` viewer page, both with an always-mounted canvas
- **Features:** Generation-only stream, lifecycle trails (amber pending → green success / red failure), model fidelity, persistent period history with six periods (today, yesterday, current week, previous week, current month, previous month), per-period model count, request total, and previous-period trend
- **Model block naming:** blocks are named by model alias, falling back to the real model name when no alias exists. Grouping uses `COALESCE(NULLIF(TRIM(model_alias), ''), model)`, so prefixed real names sharing one alias collapse into a single block. History rows report `source_models` so live stream events (which carry only the routed model name) fold onto the same alias block.
- **Canvas layout:** blocks are centered on a ring whose radius grows with the model count so neighbours never overlap. Blocks with in-flight requests or a pending trail are raised above idle blocks (`zIndex` 10 vs 1, plus the `live-flow-node-active` class) so a busy block and its label are never covered.
- **Files:**
  - `internal/api/live_flow.go` (relay handler)
  - `internal/liveflow/` (period model + history service), `internal/liveflow/httpapi/` (history route)
  - `internal/repository/liveflow_models.go` (alias-grouped aggregation over `usage_events`)
  - `web/src/features/usage/LiveFlowPanel.tsx`, `liveFlow.ts`, `FlowTopology.tsx` (panel, canvas, ring layout, active-block z-index)
  - `web/src/features/live-flow/` (history types, API client, alias-aware merge, trend helpers, period select)
  - `web/src/pages/KeyLiveFlowPage.tsx` (API-key viewer page)

**Documentation:**
- `docs/ARCHITECTURE.md` §4 (Live Flow relay) and §7 (Live Flow history domain)
- `docs/SPECIFICATION.md` UC-2a (inspect Live Flow) and UC-2d (historical model view)
- `CHANGELOG.md` (Live Flow features)
- `session.md` (Live Flow implementation notes)

**Verification after merge:**
```bash
# Verify backend relay and history domain
test -f internal/api/live_flow.go && echo "✓ Relay handler exists" || echo "✗ MISSING"
grep -q "/live-flow/ws" internal/api/live_flow.go && echo "✓ Relay route exists" || echo "✗ MISSING"
grep -q "registerLiveFlowRoutes" internal/api/router.go && echo "✓ Relay route registered" || echo "✗ MISSING"
test -f internal/liveflow/period.go && echo "✓ Period model exists" || echo "✗ MISSING"
test -f internal/liveflow/httpapi/routes.go && echo "✓ History route exists" || echo "✗ MISSING"
test -f internal/repository/liveflow_models.go && echo "✓ Aggregation exists" || echo "✗ MISSING"

# Verify frontend
test -f web/src/features/usage/LiveFlowPanel.tsx && echo "✓ Panel exists" || echo "✗ MISSING"
test -f web/src/features/usage/FlowTopology.tsx && echo "✓ Canvas exists" || echo "✗ MISSING"
test -f web/src/features/live-flow/merge.ts && echo "✓ History domain exists" || echo "✗ MISSING"
test -f web/src/pages/KeyLiveFlowPage.tsx && echo "✓ Viewer page exists" || echo "✗ MISSING"

# Verify alias-based block naming survives
grep -q "model_alias" internal/repository/liveflow_models.go && echo "✓ Alias grouping present" || echo "✗ MISSING"
grep -q "buildLiveFlowAliasMap" web/src/features/live-flow/merge.ts && echo "✓ Alias merge present" || echo "✗ MISSING"

# Verify canvas layout guards survive
grep -q "liveFlowRingRadius" web/src/features/usage/FlowTopology.tsx && echo "✓ Ring layout present" || echo "✗ MISSING"
grep -q "liveFlowNodeZIndex" web/src/features/usage/FlowTopology.tsx && echo "✓ Active z-index present" || echo "✗ MISSING"

# Verify navigation entry
grep -q "'live-flow'" web/src/lib/usageNavigation.ts && echo "✓ Tab registered" || echo "✗ MISSING"
grep -q "live-flow" web/src/App.tsx && echo "✓ Route registered" || echo "✗ MISSING"
```

---

### 5. API Key Viewer Enhancements (Phase 2.19)

**Status:** Fork-specific viewer capabilities  
**Commit baseline:** October 2026

API key viewers have access to request events and live flow (global metadata
view).

- **Request events endpoints:**
  - `GET /api/v1/key-events` (own events, paginated)
  - `GET /api/v1/key-events/:id` (own event details)
  - `GET /api/v1/key-events/:id/request-log` (full log, see §2)
- **Live flow access:** Key viewers can access `/live-flow` page (global CPA metadata stream)
- **Files:**
  - `internal/api/key_viewer_request_events.go` (endpoints)
  - `web/src/pages/KeyViewerPage.tsx` (viewer dashboard)

**Documentation:**
- `docs/SPECIFICATION.md` (API-key viewer capabilities table)
- `docs/PHASE_2_19_PLAN.md` (Phase 2.19 requirements)
- `session.md` (Phase 2.19 implementation)

**Verification after merge:**
```bash
# Verify key-viewer request events endpoints
grep -q "GET /api/v1/key-events" internal/api/router.go && echo "✓ Key-events routes exist" || echo "✗ MISSING"
test -f internal/api/key_viewer_request_events.go && echo "✓ Handler exists" || echo "✗ MISSING"

# Verify viewer can access live flow
grep -q "live.*flow" web/src/pages/KeyViewerPage.tsx || grep -q "LiveFlow" web/src/pages/KeyViewerPage.tsx
```

---

### 6. AI Provider Table Redesign & Detail Navigation

**Status:** Permanent core feature  
**Commit baseline:** October 2026

The `/ai-provider` provider grouping table and the `/ai-provider/:providerId` detail navigation must remain implemented as the `ProviderSummarySection` and detail page. Removal or replacement requires an explicit, documented architectural decision and approval from maintainers. Upstream merges must not revert to the flat key list or remove the grid/flexbox layout.

- **Backend domain:**
  - Routes: `GET /api/v1/usage/providers` (list), `GET /api/v1/usage/providers/:providerId/keys` (detail)
  - Handler: `internal/api/usage_identities.go` (registerProviderRoutes)
  - Service: `internal/service/usage_identities_service.go` (ListProviders, ListProviderKeys)
  - Repository: `internal/repository/usage_identities.go` (ListProviders, ListProviderKeys)
  - Tests: `internal/api/test/usage_providers_test.go`, `internal/repository/test/usage_identities_test.go`
- **Frontend domain:**
  - Main table: `web/src/components/usage/credentials/ProviderSummarySection.tsx`, `web/src/pages/UsagePage.tsx`
  - SCSS:  `web/src/components/usage/credentials/CredentialSections.module.scss` (must contain `.credentialMetricHeaderGroup` and `.credentialMetricGroup` grid/flexbox rules)
  - Detail page: `web/src/pages/ProviderDetailPage.tsx`
  - API client: `web/src/lib/api.ts` (listProviders, listProviderKeys)
  - Types: `web/src/lib/types.ts` (ProviderSummary, ProvidersResponse)
  - Tests: `web/src/components/usage/credentials/test/ProviderSummarySection.test.tsx`, `web/src/pages/test/ProviderDetailPage.test.tsx`
- **Integration points:**
  - `web/src/components/usage/index.ts`: must export `ProviderSummarySection`
  - `web/src/components/usage/credentials/index.ts`: must export `ProviderSummarySection`
  - `web/src/pages/UsagePage.tsx`: must render `ProviderSummarySection` for `/ai-provider`
  - `web/src/App.tsx`: Route `/ai-provider/:providerId` → `ProviderDetailPage`
  - `web/src/pages/UsagePage.tsx`: Mounts `ProviderSummarySection` on `/ai-provider` tab
  - `web/src/components/usage/credentials/index.ts`: Exports `ProviderSummarySection`
  - `web/src/components/usage/index.ts`: Exports `ProviderSummarySection`
- **Tests:**
  - `ProviderSummarySection.test.tsx` (unit test)
  - `/ai-provider` integration test (table layout)
- **Access control:**
  - Both endpoints protected by `versionProtected` middleware (Admin + API-key viewer)
  - API-key viewers see only their own keys, scoped by session `api_key_id`
  - Credential masking applied per role (viewers see masked keys)

**Documentation:**
- `docs/ARCHITECTURE.md` §7 (permanence requirement, AI Provider usage)
- `docs/SPECIFICATION.md` UC-7a (permanent contract)
- `docs/ROADMAP.md` (policy reminder, phase documentation)
- `docs/CODE_GRAPH.md` §7 (provider grouping permanent feature)
- `AGENTS.md` §3 (permanent feature guidance)
- `session.md` (restoration history, October 2026)
- `README.md` / `README.zh.md` (compare providers feature)

**Verification after merge:**
```bash
# Verify backend routes and handlers
grep -q "registerProviderRoutes" internal/api/usage_identities.go && echo "✓ Provider routes function exists" || echo "✗ MISSING"
grep -q "/usage/providers" internal/api/usage_identities.go && echo "✓ Provider list route exists" || echo "✗ MISSING"
grep -q "/usage/providers/:providerId/keys" internal/api/usage_identities.go && echo "✓ Provider keys route exists" || echo "✗ MISSING"

# Verify service methods
grep -q "func.*ListProviders" internal/service/usage_identities_service.go && echo "✓ ListProviders service exists" || echo "✗ MISSING"
grep -q "func.*ListProviderKeys" internal/service/usage_identities_service.go && echo "✓ ListProviderKeys service exists" || echo "✗ MISSING"

# Verify repository functions
grep -q "func ListProviders" internal/repository/usage_identities.go && echo "✓ ListProviders repository exists" || echo "✗ MISSING"
grep -q "func ListProviderKeys" internal/repository/usage_identities.go && echo "✓ ListProviderKeys repository exists" || echo "✗ MISSING"

# Verify frontend components
test -f web/src/components/usage/credentials/ProviderSummarySection.tsx && echo "✓ ProviderSummarySection exists" || echo "✗ MISSING"
test -f web/src/pages/ProviderDetailPage.tsx && echo "✓ ProviderDetailPage exists" || echo "✗ MISSING"

# Verify frontend API client
grep -q "listProviders" web/src/lib/api.ts && echo "✓ listProviders API client exists" || echo "✗ MISSING"
grep -q "listProviderKeys" web/src/lib/api.ts && echo "✓ listProviderKeys API client exists" || echo "✗ MISSING"

# Verify integration points
grep -q "ProviderSummarySection" web/src/pages/UsagePage.tsx && echo "✓ ProviderSummarySection mounted" || echo "✗ MISSING"
grep -q "ProviderDetailPage" web/src/App.tsx && echo "✓ ProviderDetailPage route registered" || echo "✗ MISSING"
grep -q "ProviderSummarySection" web/src/components/usage/credentials/index.ts && echo "✓ Exported" || echo "✗ MISSING"

# Verify tests
test -f internal/api/test/usage_providers_test.go && echo "✓ Backend provider tests exist" || echo "✗ MISSING"
test -f web/src/components/usage/credentials/test/ProviderSummarySection.test.tsx && echo "✓ Frontend tests exist" || echo "✗ MISSING"
test -f web/src/pages/test/ProviderDetailPage.test.tsx && echo "✓ Detail page tests exist" || echo "✗ MISSING"

# Build verification
go build ./...
npm --prefix ./web run typecheck
npm --prefix ./web run build
```
```bash
# Verify SCSS grid/flexbox rules
grep -q "credentialMetricHeaderGroup" web/src/components/usage/credentials/CredentialSections.module.scss && echo "✓ Header group class exists" || echo "✗ MISSING"
grep -q "credentialMetricGroup" web/src/components/usage/credentials/CredentialSections.module.scss && echo "✓ Row group class exists" || echo "✗ MISSING"
# Build verification
npm --prefix ./web run typecheck
npm --prefix ./web run lint
npm --prefix ./web run test
npm --prefix ./web run build
```

### 7. API Key Name Display Priority

**Status:** Fork-specific display enhancement  
**Commit baseline:** October 2026

API keys must display with a three-tier priority: **CPA Key Name → User Alias → Masked Key**. This ensures users see meaningful names from CPA's `/v0/management/auth-api-keys` endpoint first, falling back to user-defined aliases, and finally to the masked key format.

- **Database schema:** `cpa_api_keys.key_name` column (migration `20261013_add_cpa_api_key_name`)
- **Backend entity:** `entities.CPAAPIKey.KeyName` field must be present
- **Backend sync:** `repository.SyncCPAAPIKeysWithNames()` must write to `key_name` column
- **Backend display logic:** `helper.CPAAPIKeyDisplayName()` must implement priority: `KeyName` → `KeyAlias` → `MaskedKey`
- **CPA client:** `client.FetchAuthAPIKeys()` for `/v0/management/auth-api-keys` endpoint
- **Display locations:**
  - Ranking leaderboard (`web/src/features/ranking/components/RankingLeaderboardResults.tsx`)
  - API key settings card (`web/src/components/usage/ApiKeySettingsCard.tsx`)
  - Key viewer dashboard (all key display contexts)
  - Session management (where API keys are referenced)

**Documentation:**
- `docs/SPECIFICATION.md` (UC-9: Local ranking with key names)
- `docs/ROADMAP.md` (Phase 2.20: Restoration task)
- `CHANGELOG.md` (Display name priority documentation)

**Verification after merge:**
```bash
# Verify entity field exists
grep -q "KeyName.*string" internal/entities/cpa_api_key.go && echo "✓ Entity field exists" || echo "✗ MISSING"

# Verify migration exists
test -f internal/repository/migration/20261013_add_cpa_api_key_name.go && echo "✓ Migration exists" || echo "✗ MISSING"

# Verify sync function writes key_name
grep -q "key_name" internal/repository/cpa_api_key.go && echo "✓ Sync uses key_name column" || echo "✗ MISSING"

# Verify display priority function
grep -A5 "CPAAPIKeyDisplayName" internal/helper/redact.go | grep -q "KeyName" && echo "✓ Display uses KeyName" || echo "✗ MISSING"

# Verify CPA client endpoint
grep -q "FetchAuthAPIKeys" internal/cpa/client.go && echo "✓ CPA client method exists" || echo "✗ MISSING"
grep -q "/v0/management/auth-api-keys" internal/cpa/endpoints.go && echo "✓ Endpoint defined" || echo "✗ MISSING"

# Run tests
go test ./internal/helper/test/redact_test.go
make verify
```

**Test coverage required:**
- `TestCPAAPIKeyDisplayNamePrefersKeyName` (KeyName → display)
- `TestCPAAPIKeyDisplayNameFallsBackToAlias` (KeyName empty → KeyAlias → display)
- `TestCPAAPIKeyDisplayNameFallsBackToMaskedKey` (KeyName and KeyAlias empty → masked key)
- `TestSyncCPAAPIKeysWithNamesWritesToKeyNameColumn` (sync writes to database)

---

## Merge Workflow

When merging from upstream:

1. **Before merge:**
   - Review upstream changes for conflicts with protected features
   - Identify any removals or replacements that would affect listed features
   - Document any necessary adaptations

2. **During merge:**
   - Preserve all protected domains (`internal/gnn/`, `web/src/features/model-gnn/`, etc.)
   - Maintain integration points in `internal/app/app.go` and `internal/api/router.go`
   - Keep fork markers (`FORK-ONLY` comments) and protection tests

3. **After merge:**
   - Run all verification commands listed above for each feature
   - Execute full `make verify` baseline
   - Update `session.md` with merge notes and any adaptations made
   - Update `CHANGELOG.md` with merge reference and preserved features

4. **If conflicts arise:**
   - Document the conflict in `session.md`
   - Preserve the fork feature by adapting to upstream changes
   - If a feature must be modified significantly, follow the Policy Statement process

---

## Related Documents

- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — module map, wiring, protected domains
- [docs/SPECIFICATION.md](SPECIFICATION.md) — functional contract for protected features
- [docs/ROADMAP.md](ROADMAP.md) — future work policy reminders
- [session.md](../session.md) — current session context and decisions
- [CHANGELOG.md](../CHANGELOG.md) — feature history and merge records
