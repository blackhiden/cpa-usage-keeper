# Merge Preservation Manifest

> ## 🛑 FORK-OWNED FILE — DO NOT DELETE, REPLACE, OR LET UPSTREAM OVERWRITE
>
> This file is the **authoritative upstream-merge gate** for CPA Usage Keeper.
> It is local source of truth, not a mirror of upstream documentation and not a
> changelog. An upstream merge that deletes, renames, empties, or rewrites this
> file is **an invalid merge** until the local manifest is restored.
>
> If upstream touches this file, keep the local version and port only genuinely
> new upstream text into the **Revision history** at the bottom. Never accept an
> upstream deletion of a protected feature record.

**Canonical path:** `docs/MERGE-PRESERVATION.md`
**Policy owner:** fork maintainers
**Format revision:** 2026-11-23 (stable-ID manifest)
**Fork baseline:** features introduced after `bcc2d4c4b28e4ec6e8e6367b0831a313a4347f8c`
**Legacy alias:** previously titled “Merge Preservation Policy”. Old `§1`–`§10`
references are mapped to stable `PF-xx` IDs in §2 and remain valid aliases; new
references must use `PF-xx` IDs.

---

## 0. How to use this manifest

This document is a **merge gate**, not background reading. Follow it in order.

1. **Before merging upstream:** read §2 (index) and §4 (feature records). Compare
   the upstream diff against every `PF-xx` ID. A diff that removes a file,
   route, mount point, marker, or invariant named in a record is a feature-loss
   candidate and must be resolved in favor of the fork.
2. **During conflict resolution:** never choose “theirs” for a protected file or
   integration point. Adapt upstream code around the protected contract. If a
   contract genuinely must change, stop and follow §1.
3. **After merging:** run the single gate in §5, then the targeted tests in §5,
   then `make verify`. A failed gate blocks the release; do not “fix” it by
   deleting the check.
4. **When adding a feature:** append a new `PF-xx` record using §3. Never reuse
   or renumber an ID, never edit an old record to describe a different feature,
   and never remove a record without the §1 decision process.
5. **If this file is missing after a merge:** restore it from the merge base or
   `main` before doing anything else, then re-run §5. See §6.4.

---

## 1. Policy statement

Any removal, replacement, rename, hiding, or significant modification of a
feature listed in §2 requires **all** of the following, in one reviewed change:

1. An explicit, documented architectural decision explaining why the feature is
   no longer required.
2. Approval from the fork maintainers.
3. Updates to every affected document:
   - `docs/ARCHITECTURE.md`
   - `docs/SPECIFICATION.md`
   - `docs/ROADMAP.md`
   - `README.md` / `README.zh.md`
   - `session.md`
   - `CHANGELOG.md`
   - This manifest (record marked `RETIRED` with the decision link; the ID is
     never deleted or reused).
4. A passing §5 gate **after** the decision, with the retired check explicitly
   removed in the same reviewed change.

Until that process is complete, the default answer to “did upstream remove or
replace this?” is **restore the fork behavior**.

---

## 2. Protected feature index

This table is the completeness check. If a row is missing, the manifest has been
damaged. `Legacy` exists only to resolve older prose references.

| ID | Legacy | Feature | Permanence | Owner domains | Contract anchor | Gate |
|---|---|---|---|---|---|---|
| `PF-01` | §1 | Provider Model GNN Panel | Permanent core | `internal/gnn`, `web/src/features/model-gnn` | `SPECIFICATION.md` UC-12 | §5.1 |
| `PF-02` | §2 | API Key Viewer Full Request Logs | Fork-only requirement | `internal/api` | `SPECIFICATION.md` §7 request logs | §5.2 |
| `PF-03` | §3 | Local Ranking as Default | Fork-specific configuration | `internal/ranking`, `web/src/pages` | `SPECIFICATION.md` UC-9 / UC-14 | §5.3 |
| `PF-04` | §4 | Live Flow Visualization + History | Permanent | `internal/liveflow`, `web/src/features/live-flow`, `web/src/features/usage` | `SPECIFICATION.md` UC-2a/2c/2d | §5.4 |
| `PF-05` | §5 | API Key Viewer Enhancements | Fork-specific viewer capability | `internal/api`, `web/src/features/key-viewer` | `SPECIFICATION.md` UC-13/15/16 | §5.5 |
| `PF-06` | §6 | AI Provider Table Redesign & Detail Navigation | Permanent core | `internal/api`, `internal/service`, `internal/repository`, `web/src/components/usage/credentials`, `web/src/pages` | `SPECIFICATION.md` UC-7a | §5.6 |
| `PF-07` | §7 (first) | API Key Name Display Priority | Fork-specific display rule | `internal/entities`, `internal/repository`, `internal/helper`, `internal/cpa` | `SPECIFICATION.md` UC-9 | §5.7 |
| `PF-08` | §7 (second) | Model Test & API-key Viewer Models Access | Permanent | `internal/modeltest`, `internal/adminsettings`, `web/src/features/model-test` | `SPECIFICATION.md` §1 / §5 | §5.8 |
| `PF-09` | §8 | Model Test Failure Telegram Notifications | Permanent | `internal/telegram`, `internal/adminsettings`, `web/src/features/model-test` | `SPECIFICATION.md` UC-17 | §5.9 |
| `PF-10` | §9 | Model Ranking on `/ranking` | Permanent | `internal/modelranking`, `internal/repository`, `web/src/features/model-ranking` | `SPECIFICATION.md` UC-9a/9b | §5.10 |
| `PF-11` | §10 | Model Ranking Top Spender Column | Permanent | `internal/modelranking`, `internal/repository/model_ranking_rollup.go`, `web/src/features/model-ranking` | `SPECIFICATION.md` §6.6 | §5.11 |

---

## 3. Feature record format (mandatory for new entries)

Every `PF-xx` record in §4 must contain these fields, in this order. This is what
makes a silent feature loss visible during review: a missing field is a broken
record, just like a missing file is a broken gate.

```text
### PF-xx — Feature name
**Status:** Permanent | Fork-only | Fork-specific configuration
**Legacy:** §old-section (or "new in this revision")
**Introduced:** date / commit / phase

Protected contract:
  - One sentence stating the user-visible behavior that must survive.
  - Explicit "must not be removed / renamed / hidden / replaced" wording.

Owner domains and files:
  - Backend domain(s), frontend domain(s), routes, migrations, tests.

Integration points:
  - Exact files and symbols that connect the domain to the app. These are the
    lines upstream merges most often revert; list them all.

Invariants:
  - Security, scoping, masking, failure-isolation, and data-source rules that
    must not be weakened.

Documentation:
  - Every authoritative doc that repeats this contract.

Verification:
  - Gate subsection in §5 (structural check).
  - Targeted test command(s) that prove behavior, not just file presence.
```

Rules for record edits:

- **Append-only IDs.** `PF-12` is next; `PF-07` never becomes something else.
- **No silent weakening.** Changing an invariant is a §1 decision, even if the
  code still compiles.
- **No path-only protection.** A file existing is not enough; name the mount
  point, route registration, or marker that proves the feature is wired.
- **No orphan checks.** Every §5 check must map to a §4 record and vice versa.
- **No stale paths.** If a file moves, update the record and the gate in the same
  commit (see the `KeyViewerPage.tsx` drift fixed in this revision).

---

## 4. Protected feature records

### PF-01 — Provider Model GNN Panel

**Status:** Permanent core
**Legacy:** §1
**Introduced:** Since fork inception

**Protected contract:** The Provider Model GNN visualization panel must remain
mounted on the Usage overview tab. Both renderers — React Flow grid and Reagraph
WebGL (lazy-loaded) — must be preserved. The panel must not be removed, renamed,
hidden, or replaced.

- **Backend domain:** `internal/gnn/` (self-contained service + HTTP API)
- **Frontend domain:** `web/src/features/model-gnn/` (panel, renderers, API client, types)
- **API endpoints:** `GET /api/v1/provider-model-gnn` (canonical), `/provider-model-graph` (alias)
- **Integration points:**
  - `internal/app/app.go`: `gnn.NewService`
  - `internal/api/router.go`: `OptionalProviders.ProviderModelGraph` + `gnnhttpapi.RegisterRoutes`
  - `web/src/pages/UsagePage.tsx`: `ProviderModelGNNPanel` import from `@/features/model-gnn`
  - i18n keys: `usage_stats.provider_model_graph.*` (en/zh/zh-TW)
- **Invariants:** the CPA config proxy decodes only whitelisted fields via
  `internal/cpa/dto/providerconfig`; `api-key`, `base-url`, and `headers` are
  never parsed, stored, or forwarded. Both renderers reuse the same fetched
  graph and `buildProviderModelGraph` helper.
- **Documentation:** `ARCHITECTURE.md` §7, `SPECIFICATION.md` UC-12,
  `ROADMAP.md`, `README.md` / `README.zh.md`, `AGENTS.md` §3, `session.md`.
- **Verification:** §5.1; `go build ./...`, `npm --prefix ./web run typecheck`,
  `npm --prefix ./web run build`.

### PF-02 — API Key Viewer Full Request Logs

**Status:** Fork-only requirement
**Legacy:** §2
**Introduced:** Since fork inception

**Protected contract:** API key viewers must see the **full request/response log
content** for their own events — request/response bodies, headers, prompts, and
upstream payloads. No redaction layer may be inserted for this viewer role.

- **Route:** `GET /api/v1/key-events/:id/request-log`
- **File:** `internal/api/key_viewer_request_events.go`
- **Marker:** `FORK-ONLY: The owning API-key viewer receives every request-log section`
- **Access control:** ownership verified (cross-key → 404); requires
  `CPA_REQUEST_LOG_ACCESS_ENABLED=true` (disabled → 403).
- **Test:** `TestAPIKeyViewerRequestLogFullContentPersistence` compares every
  returned section with the original title and content.
- **Invariants:** the fork marker stays in the handler; no viewer-specific
  redaction helper may be added; cross-key access stays 404.
- **Documentation:** `SPECIFICATION.md` §7 request logs, `session.md`,
  `README.md` / `README.zh.md` (`CPA_REQUEST_LOG_ACCESS_ENABLED`).
- **Verification:** §5.2; run the three request-log protection tests in §5.

### PF-03 — Local Ranking as Default

**Status:** Fork-specific feature configuration
**Legacy:** §3
**Introduced:** Phase 2.19 (October 2026)

**Protected contract:** Ranking displays **local API key competition by
default**. Community ranking code and endpoints remain intact but are not wired
into the UI. No scope toggle may be reintroduced.

- **Admin page:** `web/src/pages/UsagePage.tsx` — `rankingScope: RankingScope = 'local'`
- **Key viewer page:** `web/src/pages/KeyRankingPage.tsx` — `effectiveScope: RankingScope = 'local'`
- **Backend:** `rankinghttpapi.RegisterLocalRoutes` and
  `rankinghttpapi.RegisterKeyViewerLocalRoutes` stay mounted; community
  `RegisterRoutes` / `RegisterKeyViewerRoutes` stay intentionally commented out.
- **Display:** CPA-provided key names → user aliases → masked keys (see PF-07).
- **Invariants:** no `handleRankingScopeChange`; local ranking remains available
  to viewers; a period-zone change invalidates snapshots and rebuilds only the
  requested window (see `SPECIFICATION.md` §6.6).
- **Documentation:** `session.md`, `CHANGELOG.md`, `ARCHITECTURE.md` §3.
- **Verification:** §5.3; `go test ./internal/ranking/... -race -count=1`.

### PF-04 — Live Flow Visualization + History

**Status:** Permanent
**Legacy:** §4
**Introduced:** October 2026; history in Phase 2.22

**Protected contract:** Authenticated Live Flow relay and dashboard pages show
CPA's metadata-only request stream with lifecycle trails, plus persistent
per-model history for six selectable calendar periods. The relay contract must
not change and the canvas must stay mounted before traffic arrives.

- **Backend relay:** `GET /api/v1/live-flow/ws` (`internal/api/live_flow.go`)
- **Backend history:** `GET /api/v1/live-flow/models?period=<value>`
  (`internal/liveflow/`, `internal/liveflow/httpapi/`,
  `internal/repository/liveflow_models.go`)
- **Frontend:** `/live-flow` and `/key-live-flow` pages;
  `web/src/features/usage/LiveFlowPanel.tsx`, `liveFlow.ts`, `FlowTopology.tsx`;
  `web/src/features/live-flow/` (types, API client, alias-aware merge, trend,
  period select); `web/src/pages/KeyLiveFlowPage.tsx`
- **Features:** generation-only stream; amber pending → green success / red
  failure trails; model fidelity; six periods (today, yesterday, current week,
  previous week, current month, previous month); per-period model count, request
  total, and previous-period trend.
- **Alias naming (must survive):** blocks group by
  `COALESCE(NULLIF(TRIM(model_alias), ''), model)`; `source_models` is reported;
  `buildLiveFlowAliasMap` folds live events onto the alias block.
- **Canvas layout (must survive):** `liveFlowRingRadius` grows with model count;
  `liveFlowNodeZIndex` (10 vs 1) plus `live-flow-node-active` keeps active blocks
  above idle neighbours.
- **Invariants:** metadata only — no prompts, bodies, headers, or credentials;
  history is on-demand over `usage_events` (generation only, reader pool, bounded
  `LIMIT`); no new table, no background runner; `usage_events_archive` is never
  queried; history shares the relay's admin + viewer authorization.
- **Documentation:** `ARCHITECTURE.md` §4/§7, `SPECIFICATION.md` UC-2a/2c/2d,
  `CHANGELOG.md`, `session.md`, `CODE_GRAPH.md`.
- **Verification:** §5.4; `go test ./internal/liveflow/...`,
  `go test ./internal/repository/test -run TestListLiveFlowModels`,
  `go test ./internal/api/test -run TestLiveFlowModels`,
  `npx vitest run src/features/live-flow src/features/usage`.

### PF-05 — API Key Viewer Enhancements

**Status:** Fork-specific viewer capability
**Legacy:** §5
**Introduced:** Phase 2.19 (October 2026)

**Protected contract:** API key viewers retain request events, full request logs
(PF-02), and Live Flow access (PF-04). Their shell and navigation must not be
replaced by the admin dashboard or a flat upstream viewer page.

- **Request events endpoints:** `GET /api/v1/key-events` (own events, paginated),
  `GET /api/v1/key-events/:id` (own detail),
  `GET /api/v1/key-events/:id/request-log` (full log, PF-02)
- **Router:** `registerKeyRequestEventsRoute` stays on `keyViewerProtected`
- **Frontend:** `web/src/features/key-viewer/KeyViewerShell.tsx`,
  `web/src/features/key-viewer/navigation.ts`, `web/src/pages/KeyEventsPage.tsx`,
  `web/src/pages/KeyLiveFlowPage.tsx`, `web/src/pages/KeyRankingPage.tsx`
  (the former `web/src/pages/KeyViewerPage.tsx` no longer exists — do not restore
  that stale path).
- **Invariants:** viewer scope is session-derived; cross-key event access is 404;
  viewer endpoints are rate-limited; Live Flow is a global metadata view with a
  privacy notice.
- **Documentation:** `SPECIFICATION.md` UC-13/15/16, `docs/PHASE_2_19_PLAN.md`,
  `session.md`.
- **Verification:** §5.5; `go test ./internal/api/test/ -run 'TestAPIKeyViewer' -count=1`.

### PF-06 — AI Provider Table Redesign & Detail Navigation

**Status:** Permanent core
**Legacy:** §6
**Introduced:** October 2026; restored and protected after upstream regression

**Protected contract:** `/ai-provider` must render the server-grouped
`ProviderSummarySection` (one row per provider), and `/ai-provider/:providerId`
must render `ProviderDetailPage` with paginated, server-sorted keys. Upstream
must not revert to the flat key list, remove the grid/flexbox layout, or remove
the provider API endpoints.

- **Backend routes:** `GET /api/v1/usage/providers`,
  `GET /api/v1/usage/providers/:providerId/keys`,
  `GET /api/v1/usage/providers/:providerId/models`
- **Backend files:** `internal/api/usage_identities.go` (`registerProviderRoutes`),
  `internal/service/usage_identities_service.go`,
  `internal/repository/usage_identities.go`, `internal/repository/provider_models.go`;
  tests in `internal/api/test/usage_providers_test.go` and
  `internal/repository/test/usage_identities_test.go`
- **Frontend files:** `web/src/components/usage/credentials/ProviderSummarySection.tsx`,
  `web/src/pages/ProviderDetailPage.tsx`, `web/src/lib/api.ts`
  (`listProviders`, `listProviderKeys`), `web/src/lib/types.ts`
  (`ProviderSummary`, `ProvidersResponse`), `web/src/components/ui/ProviderDataDialog.tsx`,
  `ProviderKeysDialog`, `ProviderModelsDialog`
- **Integration points:** exports from `web/src/components/usage/credentials/index.ts`
  and `web/src/components/usage/index.ts`; mount in `web/src/pages/UsagePage.tsx`;
  route in `web/src/App.tsx`; SCSS classes `.credentialMetricHeaderGroup` and
  `.credentialMetricGroup` in `CredentialSections.module.scss`
- **Invariants:** both endpoints accept admin + active API-key viewer; viewer
  scope is session-derived; viewers see only their own keys; credential masking
  is applied per role; the provider filter bar stays on the `credentials` tab,
  not `/ai-provider`.
- **Documentation:** `ARCHITECTURE.md` §7, `SPECIFICATION.md` UC-7a,
  `ROADMAP.md`, `CODE_GRAPH.md`, `AGENTS.md` §3, `README.md` / `README.zh.md`,
  `session.md`, `BUGFIX.md`.
- **Verification:** §5.6; `npm --prefix ./web run test -- src/pages/test/UsagePageProviderNavigation.test.tsx src/components/usage/credentials/test/ProviderSummarySection.test.tsx src/pages/test/ProviderDetailPage.test.tsx`.

### PF-07 — API Key Name Display Priority

**Status:** Fork-specific display rule
**Legacy:** §7 (first)
**Introduced:** October 2026

**Protected contract:** API keys display with the three-tier priority **CPA Key
Name → User Alias → Masked Key** everywhere. A merge must not drop `KeyName` from
the entity, sync, helper, or CPA client.

- **Schema:** `cpa_api_keys.key_name` (migration `20261013_add_cpa_api_key_name`)
- **Entity:** `entities.CPAAPIKey.KeyName`
- **Sync:** `repository.SyncCPAAPIKeysWithNames()` writes `key_name`
- **Display logic:** `helper.CPAAPIKeyDisplayName()` implements
  `KeyName → KeyAlias → MaskedKey`
- **CPA client:** `client.FetchAuthAPIKeys()` → `/v0/management/auth-api-keys`
- **Display locations:** ranking leaderboard, API key settings card, key viewer
  dashboard, session management, Telegram reporter (PF-09), Top Spender labels (PF-11)
- **Invariants:** the raw key is never used as a display label; unknown keys fall
  back to `helper.RedactSensitiveValue`; soft-deleted rows stay resolvable for
  attribution.
- **Required tests:** `TestCPAAPIKeyDisplayNamePrefersKeyName`,
  `TestCPAAPIKeyDisplayNameFallsBackToAlias`,
  `TestCPAAPIKeyDisplayNameFallsBackToMaskedRawKey`,
  `TestCPAAPIKeyDisplayNameThreeTierPriority`, and the sync write test in
  `internal/repository/test/`.
- **Documentation:** `SPECIFICATION.md` UC-9, `ROADMAP.md` Phase 2.20,
  `CHANGELOG.md`.
- **Verification:** §5.7; `go test ./internal/helper/test/redact_test.go`.

### PF-08 — Model Test & API-key Viewer Models Access

**Status:** Permanent
**Legacy:** §7 (second)
**Introduced:** Phase 2.23 (November 2026)

**Protected contract:** The authenticated `/models` catalog and per-model test
controls remain available to API-key viewers without exposing pricing settings.
The server selects credentials by role; the browser cannot choose a credential,
host, path, or header.

- **Backend:** `internal/modeltest/`, `internal/modeltest/httpapi/`,
  `internal/adminsettings/`, `internal/cpa/client.go`
- **Frontend:** `web/src/features/model-test/`, `web/src/pages/KeyModelsPage.tsx`,
  `web/src/components/usage/AdminSettingsCard.tsx`,
  `web/src/components/usage/models/ModelCatalog.tsx`
- **Routes:** authenticated `GET /api/v1/models`, `POST /api/v1/model-test`,
  admin-only `GET/PUT /api/v1/admin-settings`; registered via
  `modeltesthttpapi.RegisterAuthenticatedRoutes` and
  `adminsettingshttpapi.RegisterRoutes`
- **Request contract:** configured CPA base URL, fixed bounded prompt, maximum
  output budget, regular/streaming modes, 1–600 second timeout (default 30).
- **Invariants:** raw credentials are never returned; the admin key is encrypted
  with the AES-GCM key derived from `CPA_MANAGEMENT_KEY`; viewer requests use only
  the active session key; pricing routes stay admin-only; all three locales carry
  the strings.
- **Documentation:** `ARCHITECTURE.md` §2/§7, `SPECIFICATION.md` §1/§5,
  `ROADMAP.md` Phase 2.23, `CHANGELOG.md`, `session.md`.
- **Verification:** §5.8; `go test ./internal/modeltest/... ./internal/adminsettings/... ./internal/api/...`.

### PF-09 — Model Test Failure Telegram Notifications

**Status:** Permanent
**Legacy:** §8
**Introduced:** Phase 2.24 (November 2026)

**Protected contract:** A user whose model test failed can notify the
administrator through a configured Telegram bot. The message reports the
reporter as the **API Key Name** (never the key value), the tested model, and a
bounded failure reason. Configuration lives in `/settings` → **Admin Settings**.

- **Backend:** `internal/telegram/` (contract, rendering, Bot API client,
  limiter, `FailureStore`), `internal/telegram/httpapi/`,
  `internal/adminsettings/` (`admin.telegram.bot_token`, `chat_id`,
  `message_thread_id`, `enabled`, `muted_models`)
- **Frontend:** `web/src/features/model-test/` (notify action),
  `web/src/components/usage/AdminSettingsCard.tsx` (Telegram section)
- **Routes:** `POST /api/v1/model-test/notify` (admin + viewer),
  `POST /api/v1/telegram/test-message` (admin only), extended
  `GET/PUT /api/v1/admin-settings` (admin only)
- **Wiring:** `internal/app/app.go` (`telegram.NewService` +
  `telegramSettingsSource`), `internal/api/router.go`
  (`telegramhttpapi.RegisterRoutes` / `RegisterAdminRoutes`),
  `internal/api/telegram_wiring.go` (session-derived reporter, automatic-send
  observer)
- **Invariants:** bot token encrypted at rest and never returned (`GET` exposes
  only `telegram.bot_token_configured`); reporter resolved server-side via
  `activeAPIKeyViewerContext` + `helper.CPAAPIKeyDisplayName`; administrators
  report as `Administrator`; browser sends only `model` and `mode`; reason is
  re-bounded to 240 chars, control characters stripped, key-like substrings
  redacted; per-reporter-per-model cooldown plus hourly cap; `enabled` and
  `muted_models` enforced server-side; stable error codes never echo the token,
  chat id, or Bot API body.
- **Documentation:** `ARCHITECTURE.md` §7, `SPECIFICATION.md` UC-17/§7,
  `ROADMAP.md` Phase 2.24, `CHANGELOG.md`, `session.md`.
- **Verification:** §5.9; `go test ./internal/telegram/... ./internal/adminsettings/... ./internal/api/...`.

### PF-10 — Model Ranking on `/ranking`

**Status:** Permanent
**Legacy:** §9
**Introduced:** Phase 2.25 (November 2026)

**Protected contract:** `/ranking` keeps a second, model-ranking leaderboard over
six periods and seven quantitative metrics; `/key-ranking` keeps a strictly
viewer-scoped model card. The API-key leaderboard must keep working unchanged.

- **Backend:** `internal/modelranking/`, `internal/modelranking/httpapi/`,
  `internal/repository/model_ranking.go`, `internal/repository/model_ranking_rollup.go`,
  `internal/repository/dto/model_ranking.go`
- **Schema/pipeline:** migration `20261119_model_ranking_rollups` adds
  `model_ranking_period_stats` and the checkpoint table; the serial usage
  aggregation runner commits rollup pages and the model cursor atomically;
  `repository.BackfillModelRankingRollups(ctx, db, now, maxPages)` is the bounded
  resumable recovery entry point.
- **Frontend:** `web/src/features/model-ranking/` (types, periods, formatting,
  API client, hook, `ModelRankingSection` + styles)
- **Routes:** global admin `GET /api/v1/model-ranking/leaderboards` and viewer
  `GET /api/v1/key-model-ranking/leaderboards`, with `Cache-Control: no-store`,
  `400 invalid_model_ranking_selection`, `503 model_ranking_unavailable`
- **Mount points:** `<ModelRankingSection />` immediately after `<RankingPage />`
  in `web/src/pages/UsagePage.tsx`; viewer card in `web/src/pages/KeyRankingPage.tsx`
- **Wiring:** `modelrankinghttpapi.RegisterRoutes(adminProtected, ...)` and
  `modelrankinghttpapi.RegisterKeyViewerRoutes(...)` in `internal/api/router.go`;
  `modelranking.NewService` in `internal/app/app.go`
- **Invariants:** metadata only — no prompts, bodies, headers, or credentials;
  alias matching is exact after `TRIM`; reader pool only; archive never queried
  online; rollup rows and cursor advance atomically; viewer scope is
  session-derived; missing cost is not zero; both ranking domains use project
  `TZ`; i18n namespace `model_ranking` exists in en/zh/zh-TW.
- **Documentation:** `ARCHITECTURE.md` §7, `SPECIFICATION.md` UC-9a/9b/§6.6,
  `ROADMAP.md`, `CHANGELOG.md`, `session.md`, `CODE_GRAPH.md`.
- **Verification:** §5.10; `go test ./internal/modelranking/... ./internal/repository/... ./internal/api/...`,
  `npm --prefix ./web run test -- src/features/model-ranking src/i18n/test/modelRankingTranslations.test.ts`.

### PF-11 — Model Ranking Top Spender Column

**Status:** Permanent
**Legacy:** §10
**Introduced:** 2026-11-19 (`feat(model-ranking): add top spender column`)

**Protected contract:** Every row of the Model Ranking card (admin `/ranking` and
viewer `/key-ranking`) carries a **Top spender** value naming the API key that
contributed most to that model in the selected period and provider filter. The
column must stay mounted on both cards, keep following the selected metric, and
keep its masked-label contract. Removing the column, dropping the per-metric
leader rule, or exposing the raw key requires §1.

- **Backend:** `internal/modelranking/service.go` (`Entry.TopSpender`,
  `attachTopSpenders`, `pickModelRankingSpenders`,
  `modelRankingSpenderDimensions`, `modelRankingSpenderOutranks`,
  `loadModelRankingSpenderLabels`) and `internal/repository/model_ranking_rollup.go`
  (`ListModelRankingSpenders`, `ModelRankingSpenderRow` including
  `CacheReadTokens`), with `repository.FindCPAAPIKeysByValues` for labels.
- **Leader rule (must survive):** `total_tokens`/`peak_tpm` → total tokens;
  `request_count`/`peak_rpm` → request count; `cache_read_rate` → cache-read
  tokens; `overall`, `ttft_average`, `latency_average` keep the spend order
  (estimated cost → total tokens → request count); API group key breaks every tie.
- **Attribution source:** only API-key-scoped rollup slices
  (`model_ranking_period_stats.scope_key <> ''`, reader pool). Global rows are
  never read; no new table, migration, or backfill may be added for this column.
- **Failure behavior:** an attribution read error fails the request instead of
  serving a silently unlabeled board.
- **Frontend:** `web/src/features/model-ranking/ModelRankingSection.tsx`
  (`data-model-ranking-top-spender`, `model_ranking.top_spender_hint` tooltip,
  `—` fallback), `types.ts` (`top_spender`), `ModelRankingSection.module.scss`,
  and `top_spender` / `top_spender_hint` in en/zh/zh-TW.
- **Security invariants:** payload exposes only `display_name` and `masked_key`;
  labels resolve through `helper.CPAAPIKeyDisplayName` (PF-07) and unknown keys
  fall back to `helper.RedactSensitiveValue`; a raw key never reaches the payload,
  DOM, or logs; viewer scope stays session-derived and the column is included in
  the scoped 30 s board cache only.
- **Regression tests:** `internal/modelranking/service_top_spender_test.go`,
  `internal/repository/test/model_ranking_top_spender_test.go`.
- **Documentation:** `SPECIFICATION.md` §6.6, `ARCHITECTURE.md` §7,
  `README.md` / `README.zh.md`, `CHANGELOG.md`, `AGENTS.md` §3, `session.md`.
- **Verification:** §5.11; `go test ./internal/modelranking/... ./internal/repository/...`,
  `npm --prefix ./web run test -- src/features/model-ranking src/i18n/test/modelRankingTranslations.test.ts`.

---

## 5. Merge gate (single source of truth)

Run this block from the repository root after every upstream merge. It replaces
the old per-feature ad-hoc `grep` blocks: one failure means one protected feature
is no longer wired, and the release must stop. The checks are grouped by `PF-xx`
so a failure points directly at the §4 record to restore.

```bash
set -u
fail=0
need_file(){ [ -e "$1" ] || { echo "MISSING file: $1"; fail=1; }; }
need_dir(){ [ -d "$1" ] || { echo "MISSING dir: $1"; fail=1; }; }
need_grep(){ grep -q -- "$2" "$1" || { echo "MISSING marker: '$2' in $1"; fail=1; }; }
need_no_grep(){ grep -q -- "$2" "$1" && { echo "FORBIDDEN marker: '$2' in $1"; fail=1; }; return 0; }

# PF-01 — Provider Model GNN Panel
need_file internal/gnn/gnn.go; need_file internal/gnn/httpapi/routes.go
need_file web/src/features/model-gnn/ProviderModelGNNPanel.tsx
need_file web/src/features/model-gnn/ProviderModelReagraphPanel.tsx
need_grep internal/app/app.go 'gnn.NewService'
need_grep internal/api/router.go 'gnnhttpapi.RegisterRoutes'
need_grep web/src/pages/UsagePage.tsx '@/features/model-gnn'

# PF-02 — API Key Viewer Full Request Logs
need_file internal/api/key_viewer_request_events.go
need_grep internal/api/key_viewer_request_events.go 'FORK-ONLY: The owning API-key viewer receives every request-log section'
need_no_grep internal/api/key_viewer_request_events.go 'redactRequestLogForKeyViewer'
need_grep internal/api/test/key_viewer_request_events_test.go 'TestAPIKeyViewerRequestLogFullContentPersistence'

# PF-03 — Local Ranking as Default
need_grep web/src/pages/UsagePage.tsx "const rankingScope: RankingScope = 'local'"
need_grep web/src/pages/KeyRankingPage.tsx "const effectiveScope: RankingScope = 'local'"
need_no_grep web/src/pages/UsagePage.tsx 'handleRankingScopeChange'
need_no_grep web/src/pages/KeyRankingPage.tsx 'handleRankingScopeChange'
need_grep internal/api/router.go 'rankinghttpapi.RegisterLocalRoutes'
need_grep internal/api/router.go 'rankinghttpapi.RegisterKeyViewerLocalRoutes'

# PF-04 — Live Flow Visualization + History
need_file internal/api/live_flow.go; need_file internal/liveflow/period.go
need_file internal/liveflow/httpapi/routes.go; need_file internal/repository/liveflow_models.go
need_file web/src/features/usage/LiveFlowPanel.tsx; need_file web/src/features/usage/FlowTopology.tsx
need_file web/src/features/live-flow/merge.ts; need_file web/src/pages/KeyLiveFlowPage.tsx
need_grep internal/repository/liveflow_models.go 'model_alias'
need_grep web/src/features/live-flow/merge.ts 'buildLiveFlowAliasMap'
need_grep web/src/features/usage/FlowTopology.tsx 'liveFlowRingRadius'
need_grep web/src/features/usage/FlowTopology.tsx 'liveFlowNodeZIndex'
need_grep web/src/lib/usageNavigation.ts "'live-flow'"

# PF-05 — API Key Viewer Enhancements
need_grep internal/api/router.go 'registerKeyRequestEventsRoute'
need_file web/src/features/key-viewer/KeyViewerShell.tsx
need_file web/src/pages/KeyEventsPage.tsx
need_grep web/src/features/key-viewer/navigation.ts 'events'

# PF-06 — AI Provider Table Redesign & Detail Navigation
need_file web/src/components/usage/credentials/ProviderSummarySection.tsx
need_file web/src/pages/ProviderDetailPage.tsx
need_grep internal/api/usage_identities.go 'registerProviderRoutes'
need_grep internal/api/usage_identities.go '/usage/providers/:providerId/keys'
need_grep web/src/pages/UsagePage.tsx 'ProviderSummarySection'
need_grep web/src/App.tsx 'ProviderDetailPage'
need_grep web/src/components/usage/credentials/CredentialSections.module.scss 'credentialMetricHeaderGroup'
need_grep web/src/components/usage/credentials/CredentialSections.module.scss 'credentialMetricGroup'

# PF-07 — API Key Name Display Priority
need_grep internal/entities/cpa_api_key.go 'KeyName.*string'
need_file internal/repository/migration/20261013_add_cpa_api_key_name.go
need_grep internal/repository/cpa_api_key.go 'key_name'
need_grep internal/helper/redact.go 'CPAAPIKeyDisplayName'
need_grep internal/cpa/client.go 'FetchAuthAPIKeys'
need_grep internal/cpa/endpoints.go '/v0/management/auth-api-keys'
need_grep internal/helper/test/redact_test.go 'TestCPAAPIKeyDisplayNamePrefersKeyName'

# PF-08 — Model Test & API-key Viewer Models Access
need_dir internal/modeltest; need_dir internal/modeltest/httpapi; need_dir internal/adminsettings
need_dir web/src/features/model-test; need_file web/src/pages/KeyModelsPage.tsx
need_file web/src/components/usage/AdminSettingsCard.tsx
need_grep internal/api/router.go 'modeltesthttpapi.RegisterAuthenticatedRoutes'
need_grep internal/api/router.go 'adminsettingshttpapi.RegisterRoutes'
need_grep web/src/App.tsx 'KeyModelsPage'

# PF-09 — Model Test Failure Telegram Notifications
need_dir internal/telegram; need_dir internal/telegram/httpapi
need_file internal/api/telegram_wiring.go
need_grep internal/app/app.go 'telegram.NewService'
need_grep internal/api/router.go 'telegramhttpapi.RegisterRoutes'
need_grep internal/api/router.go 'telegramhttpapi.RegisterAdminRoutes'
need_grep internal/api/telegram_wiring.go 'CPAAPIKeyDisplayName'

# PF-10 — Model Ranking on `/ranking`
need_dir internal/modelranking; need_dir internal/modelranking/httpapi
need_file internal/repository/model_ranking.go; need_file internal/repository/model_ranking_rollup.go
need_file web/src/features/model-ranking/ModelRankingSection.tsx
need_grep internal/api/router.go 'modelrankinghttpapi.RegisterRoutes(adminProtected'
need_grep web/src/pages/UsagePage.tsx 'ModelRankingSection'
need_grep internal/modelranking/httpapi/routes.go 'key-model-ranking/leaderboards'
need_grep internal/entities/model_ranking_period_stat.go 'model_ranking_period_stats'
need_grep web/src/i18n/index.ts 'model_ranking: {'

# PF-11 — Model Ranking Top Spender Column
need_grep internal/repository/model_ranking_rollup.go 'ListModelRankingSpenders'
need_grep internal/modelranking/service.go 'attachTopSpenders'
need_grep internal/modelranking/service.go 'modelRankingSpenderDimensions'
need_grep internal/modelranking/service.go 'modelRankingSpenderOutranks'
need_grep web/src/features/model-ranking/ModelRankingSection.tsx 'data-model-ranking-top-spender'
need_grep web/src/i18n/index.ts 'top_spender_hint'

if [ "$fail" -ne 0 ]; then echo 'MERGE GATE FAILED'; exit 1; fi
echo 'MERGE GATE PASSED'
```

### Targeted behavior tests

The structural gate proves the features are still wired. These tests prove the
contracts still behave. Run them after the gate, then run `make verify`.

```bash
go test ./internal/api/test/ -run 'TestAPIKeyViewerRequestLogFullContentPersistence|TestAPIKeyViewerRequestLogRejectsCrossKeyAccess|TestAPIKeyViewerRequestLogRejectsDisabledAccess' -count=1
go test ./internal/ranking/... -race -count=1
go test ./internal/liveflow/... ./internal/repository/test -run 'TestListLiveFlowModels|TestListModelRankingSpenders' -count=1
go test ./internal/modelranking/... ./internal/repository/... -count=1
go test ./internal/telegram/... ./internal/adminsettings/... ./internal/modeltest/... -count=1
npm --prefix ./web run test -- src/features/model-ranking src/features/live-flow src/features/usage src/pages/test/UsagePageProviderNavigation.test.tsx src/i18n/test/modelRankingTranslations.test.ts
make verify
```

---

## 6. Merge workflow and recovery

### 6.1 Before merge

- Read this manifest in full; do not rely on memory or on upstream docs.
- Review the upstream diff for every path and symbol named in §4.
- Record any necessary adaptation in `session.md` before applying it.
- Confirm the working tree is clean and this file is unchanged.

### 6.2 During merge

- Preserve every protected domain (`internal/gnn/`, `web/src/features/model-gnn/`,
  `internal/liveflow/`, `web/src/features/live-flow/`, `internal/modelranking/`,
  `web/src/features/model-ranking/`, `internal/telegram/`, etc.).
- Maintain integration points in `internal/app/app.go` and `internal/api/router.go`.
- Keep fork markers (`FORK-ONLY`, `data-model-ranking-top-spender`) and protection
  tests.
- Never resolve a conflict by deleting a protected record, check, or feature to
  make the merge succeed.

### 6.3 After merge

1. Run the §5 structural gate. Any `MISSING` / `FORBIDDEN` line blocks the release.
2. Run the §5 targeted behavior tests, then `make verify`.
3. Update `session.md` with merge notes and adaptations.
4. Update `CHANGELOG.md` with the merge reference and preserved `PF-xx` IDs.
5. If a check had to change because a file legitimately moved, update the §4
   record, the §5 gate, and `CODE_GRAPH.md` in the same commit.

### 6.4 If upstream deletes or rewrites this manifest

This is a merge blocker, not a documentation nit. Recover in this order:

1. Stop the merge/release. Do not commit the deletion.
2. Restore the file from the merge base or `main`:
   `git checkout MERGE_HEAD -- docs/MERGE-PRESERVATION.md` is **wrong** (that is
   upstream); use `git checkout HEAD -- docs/MERGE-PRESERVATION.md` or
   `git show <last-good-commit>:docs/MERGE-PRESERVATION.md > docs/MERGE-PRESERVATION.md`.
3. Re-run §5. If a protected file was also deleted by upstream, restore it from
   the last good commit and re-apply only non-conflicting upstream changes around
   it.
4. Record the incident in `session.md` and `CHANGELOG.md` with the upstream ref.

### 6.5 If a conflict cannot be resolved confidently

Stop and request maintainer confirmation. Do not guess, do not choose “ours” or
“theirs” wholesale, and do not simplify a feature to make the merge pass.

---

## 7. Related documents

- [ARCHITECTURE.md](ARCHITECTURE.md) — module map, wiring, protected domains.
- [SPECIFICATION.md](SPECIFICATION.md) — functional contract for every `PF-xx`.
- [CODE_GRAPH.md](CODE_GRAPH.md) — codebase map and feature locations.
- [ROADMAP.md](ROADMAP.md) — planned work and policy reminders.
- [CONTRIBUTING.md](CONTRIBUTING.md) — development workflow and verification baseline.
- [merge-upstream.md](merge-upstream.md) — upstream merge procedure.
- [session.md](../session.md) — current session context and decisions.
- [CHANGELOG.md](../CHANGELOG.md) — feature history and merge records.
- [AGENTS.md](../AGENTS.md) — agent guidance; `PF-xx` IDs are authoritative here.

---

## 8. Revision history

- **2026-11-23 — Feature-loss-proof format.** Converted the prose policy into a
  stable-ID manifest: fork-owned banner, completeness index, mandatory record
  template, one consolidated merge gate, targeted behavior tests, and an explicit
  restore procedure. Resolved the duplicate `§7` by assigning `PF-07` / `PF-08`;
  old section numbers remain aliases in §2. Corrected the stale
  `web/src/pages/KeyViewerPage.tsx` path to the current key-viewer shell/pages and
  the stale masked-key test name to `TestCPAAPIKeyDisplayNameFallsBackToMaskedRawKey`.
  No application behavior changed.
- **2026-11-20 — Upstream `e5e8f1a4`.** Corrected the local-ranking verification
  path (`KeyRankingPage.tsx` / `effectiveScope`) and kept the protected provider
  navigation behavior.
- **2026-11-19 — `PF-11`.** Added the Model Ranking Top Spender column contract.
- **2026-11-09 — `PF-06`.** Restored the grouped provider table and detail page
  after an upstream regression; added structural verification.
- **Initial — fork baseline.** Policy introduced after
  `bcc2d4c4b28e4ec6e8e6367b0831a313a4347f8c`.
