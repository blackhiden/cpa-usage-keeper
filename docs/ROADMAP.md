
# CPA Usage Keeper — Roadmap (Compacted)

**Legend:** ⬜ not started · 🟡 in progress · ✅ done

**Policy:** Provider Model GNN Panel is a permanent, protected feature. Any changes require explicit, documented architectural decisions (see ARCHITECTURE.md §7, SPECIFICATION.md UC-12, session.md).

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
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) §6 for the complete policy.

---

## Task ID Naming Convention

- **Feature phases:** `FEAT-PROV-X.Y` (AI Provider table redesign)
- **General improvements:** `IMP-X.Y`
- **Bug fixes:** `BUG-X`
- **Documentation:** `DOC-X`

---

## Current Phase: AI Provider Table Redesign & Detail Navigation

**Phase Goal:** Simplify the `/ai-provider` table to show only provider names (without API keys), add a detail page/view for each provider showing all keys with success rate sorting, and ensure proper access control for Admin and API-key viewer roles.

**Status:** ✅ **Complete** — Protected as a permanent core feature (see `docs/MERGE-PRESERVATION.md` §6).

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
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) §6 for the complete policy.

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
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) §6 for the complete policy.

---

## Task ID Naming Convention

- **Feature phases:** `FEAT-PROV-X.Y` (AI Provider table redesign)
- **General improvements:** `IMP-X.Y`
- **Bug fixes:** `BUG-X`
- **Documentation:** `DOC-X`

---

## Current Phase: AI Provider Table Redesign & Detail Navigation

**Phase Goal:** Simplify the `/ai-provider` table to show only provider names (without API keys), add a detail page/view for each provider showing all keys with success rate sorting, and ensure proper access control for Admin and API-key viewer roles.

**Status:** ✅ **Complete** — Protected as a permanent core feature (see `docs/MERGE-PRESERVATION.md` §6).

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
> [MERGE-PRESERVATION.md](docs/MERGE-PRESERVATION.md) §6 for the complete policy.

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
