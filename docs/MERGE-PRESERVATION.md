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
metadata-only request stream with lifecycle trails (pending/success/failure).

- **Backend relay:** `GET /api/v1/live-flow/ws` (authenticated WebSocket proxy to CPA `/live-flow/ws`)
- **Frontend page:** `/live-flow` standalone menu page with always-mounted canvas
- **Features:** Generation-only stream, lifecycle trails (amber pending → green success / red failure), model fidelity
- **Files:**
  - `internal/api/live_flow.go` (relay handler)
  - `web/src/pages/LiveFlowPage.tsx` (standalone page)
  - `web/src/components/usage/LiveFlowCanvas.tsx` (visualization)

**Documentation:**
- `docs/ARCHITECTURE.md` §4 (Live Flow relay)
- `docs/SPECIFICATION.md` UC-2a (inspect Live Flow)
- `CHANGELOG.md` (Live Flow features)
- `session.md` (Live Flow implementation notes)

**Verification after merge:**
```bash
# Verify backend relay
test -f internal/api/live_flow.go && echo "✓ Relay handler exists" || echo "✗ MISSING"
grep -q "/live-flow/ws" internal/api/router.go && echo "✓ Relay route exists" || echo "✗ MISSING"

# Verify frontend
test -f web/src/pages/LiveFlowPage.tsx && echo "✓ Page exists" || echo "✗ MISSING"
test -f web/src/components/usage/LiveFlowCanvas.tsx && echo "✓ Canvas exists" || echo "✗ MISSING"

# Verify menu entry
grep -q "/live-flow" web/src/components/layout/Sidebar.tsx || grep -q "live-flow" web/src/components/layout/Header.tsx
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
**Commit baseline:** Since fork inception (restored through upstream merge v2.0.2)

The administrator `/ai-provider` view must render **one server-aggregated row per
provider** (not one row per key), with success-rate sorting and a Detail link to
`/ai-provider/:providerId`. The detail route renders that provider's paginated
key list with server-side sorting. API-key viewers are restricted to providers
evidenced by their own usage events and receive event-scoped statistics.

- **Backend endpoints:**
  - `GET /api/v1/usage/providers` — grouped provider list with aggregate stats and success-rate sorting
  - `GET /api/v1/usage/providers/:providerId/keys` — paginated, sortable keys for a provider
- **Backend files:**
  - `internal/api/usage_identities.go` — `registerProviderRoutes` handler
  - `internal/service/usage_identities_service.go` — `ProviderReader` interface, `ListProviders`, `ListProviderKeys`
  - `internal/repository/usage_identities.go` — `ListProviders`, `ListProviderKeys` SQL queries
- **Frontend files:**
  - `web/src/components/usage/credentials/ProviderSummarySection.tsx` — admin `/ai-provider` aggregate table
  - `web/src/pages/ProviderDetailPage.tsx` — `/ai-provider/:providerId` key detail page
  - `web/src/lib/api.ts` — `listProviders`, `listProviderKeys` API clients
  - `web/src/lib/usageNavigation.ts` — `getProviderDetailPath`, `resolveProviderDetailIdFromPath`
  - `web/src/App.tsx` — route `/ai-provider/:providerId` → `ProviderDetailPage`
- **Access control:** Provider routes are accepted only for administrators; API-key viewer path normalization redirects them to their scoped dashboard. Backend authorization enforces API-key viewer scoping (only providers with keys they own).
- **Tests:**
  - `internal/api/test/usage_providers_test.go` — route auth, admin/viewer scoping, 404 handling
  - `internal/repository/test/usage_identities_test.go` — `TestListProvidersAggregatesActiveAIProviderPeriodStatsAndSortsBySuccessRate`
  - `web/src/pages/test/ProviderDetailPage.test.tsx` — detail page load, sort, CPAMC embed
  - `web/src/components/usage/credentials/test/ProviderSummarySection.test.tsx` — aggregate table

**Documentation:**
- `docs/ARCHITECTURE.md` §7 (AI Provider usage, permanence requirement)
- `docs/SPECIFICATION.md` UC-7a (inspect provider usage, permanent contract)
- `docs/CODE_GRAPH.md` §7 (provider grouping frontend conventions)
- `docs/ROADMAP.md` (policy reminder)
- `README.md` / `README.zh.md` (permanent feature note)
- `session.md` (restoration and protection decision)

**Verification after merge:**
```bash
# Verify backend endpoints
grep -q '"/usage/providers"' internal/api/usage_identities.go && echo "✓ Provider list route exists" || echo "✗ MISSING"
grep -q '"/usage/providers/:providerId/keys"' internal/api/usage_identities.go && echo "✓ Provider keys route exists" || echo "✗ MISSING"

# Verify service interface
grep -q "ListProviders" internal/service/usage_identities_service.go && echo "✓ ListProviders exists" || echo "✗ MISSING"
grep -q "ListProviderKeys" internal/service/usage_identities_service.go && echo "✓ ListProviderKeys exists" || echo "✗ MISSING"

# Verify frontend
test -f web/src/pages/ProviderDetailPage.tsx && echo "✓ ProviderDetailPage exists" || echo "✗ MISSING"
test -f web/src/components/usage/credentials/ProviderSummarySection.tsx && echo "✓ ProviderSummarySection exists" || echo "✗ MISSING"
grep -q "listProviders" web/src/lib/api.ts && echo "✓ listProviders API exists" || echo "✗ MISSING"
grep -q "listProviderKeys" web/src/lib/api.ts && echo "✓ listProviderKeys API exists" || echo "✗ MISSING"

# Verify routing
grep -q "ProviderDetailPage" web/src/App.tsx && echo "✓ Detail route wired" || echo "✗ MISSING"

# Run protection tests
go test ./internal/api/test/ -run 'TestProviderRoutes' -count=1
go test ./internal/repository/test/ -run 'TestListProviders' -count=1
npm --prefix ./web run test -- --run pages/test/ProviderDetailPage.test.tsx
npm --prefix ./web run test -- --run components/usage/credentials/test/ProviderSummarySection.test.tsx
```

---

### 7. Ranking Translation Keys

**Status:** Permanent i18n requirement  
**Commit baseline:** `9dd12e4a` (bugfix/translation-ranking branch)

The ranking feature requires a complete set of translation keys in `web/src/i18n/index.ts` for all three supported locales (`en`, `zh`, `zh-TW`). These keys were removed in upstream commit `3e79ce52` and must be preserved across all future merges.

- **Translation keys:**
  - `usage_stats.tab_ranking` — Tab label in all three locales
  - `ranking.*` — Complete namespace with 90+ keys for ranking UI, metrics, periods, profile management, and error states
- **File:** `web/src/i18n/index.ts`
- **Test:** `web/src/i18n/test/rankingTranslations.test.ts` — Verifies key presence, parity across locales, and interpolation parameter consistency
- **Origin:** Ranking UI was removed upstream (`3e79ce52`), restored in fork (`0e56c25d`, `557c002c`), but translations were not included. Full fix created in `9dd12e4a`.

**Documentation:**
- `docs/BUGFIX.md` BUG-002 (root cause analysis, fix plan)
- `docs/SPECIFICATION.md` (ranking feature contract)
- `session.md` (translation restoration decision)
- This file (`docs/MERGE-PRESERVATION.md` §7)

**Verification after merge:**
```bash
# Verify tab_ranking key exists in all locales
grep -q "tab_ranking: 'Ranking'" web/src/i18n/index.ts && echo "✓ en tab_ranking exists" || echo "✗ MISSING"
grep -q "tab_ranking: '排行'" web/src/i18n/index.ts && echo "✓ zh tab_ranking exists" || echo "✗ MISSING"

# Verify ranking namespace exists in all locales
grep -c "ranking: {" web/src/i18n/index.ts | grep -q "3" && echo "✓ ranking namespace exists in 3 locales" || echo "✗ MISSING"

# Verify test file exists
test -f web/src/i18n/test/rankingTranslations.test.ts && echo "✓ Test file exists" || echo "✗ MISSING"

# Run ranking translation tests
npm --prefix ./web run test -- --run i18n/test/rankingTranslations.test.ts

# Full verification
make verify
```

**Merge conflict resolution:**
When upstream modifies `web/src/i18n/index.ts`, the merge conflict resolution **must**:
1. Preserve upstream's additions and changes
2. Keep `usage_stats.tab_ranking` for all three locales
3. Keep the complete `ranking: { ... }` namespace for all three locales
4. Run `npm --prefix ./web run test` to verify key parity and interpolation parameters

**Failure mode:**
If these keys are omitted, the `/ranking` route will display raw placeholder keys like `ranking.title`, `ranking.metric_short_overall`, etc., instead of translated strings. The test suite will fail with key parity errors.

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
