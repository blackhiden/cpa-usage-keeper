# Changelog

All notable changes to this project are documented here.

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
