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
**Commit baseline:** October 2026

The `/ai-provider` provider grouping table and the `/ai-provider/:providerId`
detail navigation must be preserved. This feature groups API keys by provider
in the main view and provides per-provider detail pages with key-level
statistics and sorting.

- **Backend domain:**
  - Routes: `GET /api/v1/usage/providers` (list), `GET /api/v1/usage/providers/:providerId/keys` (detail)
  - Handler: `internal/api/usage_identities.go` (registerProviderRoutes)
  - Service: `internal/service/usage_identities_service.go` (ListProviders, ListProviderKeys)
  - Repository: `internal/repository/usage_identities.go` (ListProviders, ListProviderKeys)
  - Tests: `internal/api/test/usage_providers_test.go`, `internal/repository/test/usage_identities_test.go`
- **Frontend domain:**
  - Main table: `web/src/components/usage/credentials/ProviderSummarySection.tsx`
  - Detail page: `web/src/pages/ProviderDetailPage.tsx`
  - API client: `web/src/lib/api.ts` (listProviders, listProviderKeys)
  - Types: `web/src/lib/types.ts` (ProviderSummary, ProvidersResponse)
  - Tests: `web/src/components/usage/credentials/test/ProviderSummarySection.test.tsx`, `web/src/pages/test/ProviderDetailPage.test.tsx`
- **Integration points:**
  - `web/src/App.tsx`: Route `/ai-provider/:providerId` → `ProviderDetailPage`
  - `web/src/pages/UsagePage.tsx`: Mounts `ProviderSummarySection` on `/ai-provider` tab
  - `web/src/components/usage/credentials/index.ts`: Exports `ProviderSummarySection`
  - `web/src/components/usage/index.ts`: Exports `ProviderSummarySection`
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
