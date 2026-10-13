# CPA Usage Keeper — Code Graph & Project Map

> **Purpose:** This document provides a high-level map of the entire codebase structure, module relationships, and data flow. Use it as your first reference when exploring the project—it minimizes tool calls and accelerates orientation for agents and developers.

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
│ Browser│◄──────►│  CPA Usage Keeper        │◄──────►│  Optional:     │
│ (SPA)  │  HTTP  │  Go server + SQLite      │ HTTPS  │  Ranking Center│
└────────┘        │  (embedded web/dist)     │        │  GitHub Updates│
                  └──────────┬───────────────┘        └────────────────┘
                             │ file backups / logs / archive
                             ▼
                        WORK_DIR (./data)
```

**Key facts:**
- Single Go process embedding the built React SPA
- SQLite as the only persistent store (under `WORK_DIR`)
- Optional Redis for ingestion (alternatives: HTTP pull, Redis list queue)
- Optional egress: ranking center, GitHub update checks

---

## 2. Backend Code Structure

### Entry Point → Assembly → Layers

```
cmd/server/main.go
    │
    ├─ Parse flags: --env, --host, -v/--version
    ├─ Configure bootstrap logging
    └─ Call internal/app.New() + Run()
            │
            ▼
internal/app/app.go (APPLICATION ASSEMBLY)
    │
    ├─ Database pool (writer/reader via dbresolver)
    ├─ Repositories (data access layer)
    ├─ Services (business logic)
    ├─ Runners (background agents)
    ├─ API router (Gin)
    └─ HTTP server
            │
            ▼
        ┌───────────────────────────────────────┐
        │         API LAYER (thin handlers)     │
        │       internal/api/router.go          │
        │   - parse/validate → call service →   │
        │     map DTO, never see GORM types     │
        └───────────────────────────────────────┘
                        │
                        ▼
        ┌───────────────────────────────────────┐
        │      SERVICE LAYER (business logic)   │
        │         internal/service/             │
        │   - usage, identities, pricing,       │
        │     quota, auth, metadata sync        │
        └───────────────────────────────────────┘
                        │
                        ▼
        ┌───────────────────────────────────────┐
        │    REPOSITORY LAYER (data access)     │
        │       internal/repository/            │
        │   - All SQL/GORM lives here           │
        │   - Repository pattern per entity     │
        │   - Migrations in migration/          │
        └───────────────────────────────────────┘
                        │
                        ▼
        ┌───────────────────────────────────────┐
        │      ENTITIES (schema source)         │
        │         internal/entities/            │
        │   - GORM models, single source of     │
        │     schema truth                      │
        └───────────────────────────────────────┘
```

### Backend Module Map

| Package | Role |
|---------|------|
| `cmd/server` | Binary entry: CLI flags, bootstrap, version |
| `internal/app` | Application assembly, wiring, lifecycle (Run/Close) |
| `internal/api` | Gin router, thin handlers, auth middleware, DTO mapping |
| `internal/service` | Business services: usage, identities, pricing, quota, auth, metadata sync, token processing |
| `internal/repository` | All SQL/GORM access (repository pattern) + migrations |
| `internal/entities` | GORM models — single source of schema truth |
| `internal/poller` | Ingestion sources, inbox writer, ingest/process/aggregation runners |
| `internal/quota` | Provider quota clients (Claude, Codex, Gemini CLI, Kimi, xAI, Antigravity), refresh engine, history |
| `internal/gnn` | **Provider Model GNN feature** (permanent): sanitized graph, node/edge embeddings, HTTP surface in `gnn/httpapi` |
| `internal/ranking` | Opt-in community ranking client/service |
| `internal/auth` | Session management (memory/persistent), login IP/user-agent capture, activity tracking |
| `internal/enrichgeo` | Opt-in IP geo/enrichment (reverse DNS PTR resolver, async cache) |
| `internal/pricing` | Price catalog, resolver, snapshot |
| `internal/cpa` | CPA management API client (HTTP + Redis queue client) |
| `internal/activity` | Activity aggregation (short/medium/long/daily grains), bucket logic |
| `internal/benchmark` | **Production capacity suite** (test-only, Linux amd64, not in baseline CI) |
| `internal/helper` | Redaction, utilities |
| `internal/config` | Configuration loading, validation |
| `internal/logging` | Structured logging setup |
| `internal/timeutil` | Time utilities, storage serialization |
| `internal/updatecheck` | GitHub release update checker |
| `internal/version` | Version injection point |

---

## 3. Frontend Code Structure

```
web/
├── src/
│   ├── App.tsx                     # Root component, routing
│   ├── main.tsx                    # Entry point (React 19, Vite)
│   ├── pages/                      # Top-level route pages
│   │   ├── UsagePage.tsx           # Main admin dashboard
│   │   ├── LoginPage.tsx           # Authentication
│   │   ├── KeyOverviewPage.tsx    # API-key viewer (scoped)
│   │   ├── KeyAnalysisPage.tsx    # API-key analysis (scoped)
│   │   └── KeyRankingPage.tsx     # API-key ranking (scoped)
│   ├── components/                 # Reusable UI components
│   │   ├── ui/                     # Bespoke UI kit (no library)
│   │   ├── dashboard/              # Dashboard shell components
│   │   └── usage/                  # Usage-specific components
│   ├── features/                   # Feature-specific domains
│   │   ├── ranking/                # Community ranking (opt-in)
│   │   ├── model-gnn/              # Provider Model GNN (permanent)
│   │   └── usage/                  # Live Flow WebSocket
│   ├── stores/                     # Zustand state management
│   ├── lib/                        # Core libraries (api, types, chartjs)
│   ├── i18n/                       # i18next (en, zh, zh-TW)
│   ├── utils/                      # Utility functions
│   ├── hooks/                      # Custom React hooks
│   └── styles/                     # Global SCSS
└── dist/                           # Vite build (embedded into Go binary)
```

### Frontend Data Flow

```
User Action
    │
    ▼
Component (React)
    │
    ├─ Call API (lib/api.ts)
    │       │
    │       ▼
    │   HTTP Request → /api/v1/... (backend)
    │       │
    │       ▼
    │   Response (JSON)
    │
    ├─ Update Store (Zustand)
    │       │
    │       └─ useUsageStatsStore, useThemeStore, etc.
    │
    └─ Re-render with new data
```

---

## 4. Data Flow: Ingestion → Storage → API → Frontend

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INGESTION SOURCES                            │
│  - Redis Pub/Sub (subscribe to "usage")                            │
│  - Redis List Queue (BLPOP from list)                              │
│  - HTTP Pull (poll /api/usage endpoint)                            │
└────────────────────────┬────────────────────────────────────────────┘
                         │ Raw usage events
                         ▼
        ┌────────────────────────────────────┐
        │   RedisIngest Runner               │
        │   (internal/poller)                │
        │   - Validates payloads             │
        │   - Writes to redis_usage_inboxes  │
        │   - Backoff 1s→30s on failure     │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   RedisProcess Runner              │
        │   (internal/poller)                │
        │   - Reads from redis_usage_inboxes │
        │   - Token processing (per provider)│
        │   - Writes to usage_events         │
        │   - At-least-once semantics        │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   SQLITE DATABASE                  │
        │   - usage_events (90-day hot)      │
        │   - usage_events_archive (cold)    │
        │   - usage_identities               │
        │   - usage_activity_stats           │
        │   - overview hourly/daily          │
        │   - latency_stats                  │
        │   - pricing, quota, metadata, etc. │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   UsageAggregation Runner          │
        │   (internal/poller)                │
        │   - Serial runner, per-pipeline    │
        │     checkpoints                    │
        │   - Overview hourly/daily rollups  │
        │   - Activity (short/medium/long)   │
        │   - Latency stats                  │
        └────────────────┬───────────────────┘
                         │
                         ▼
        ┌────────────────────────────────────┐
        │   API LAYER (internal/api)         │
        │   - GET /api/v1/usage/overview     │
        │   - GET /api/v1/usage/events       │
        │   - GET /api/v1/usage/activity     │
        │   - GET /api/v1/usage/analysis     │
        │   - GET /api/v1/quota/...          │
        └────────────────┬───────────────────┘
                         │ JSON responses
                         ▼
        ┌────────────────────────────────────┐
        │   FRONTEND (web/src)               │
        │   - API calls (lib/api.ts)         │
        │   - State management (Zustand)     │
        │   - Charts (Chart.js)              │
        │   - Tables, forms, modals          │
        └────────────────────────────────────┘
```

---

## 5. Background Agents (Runners)

**Started by `internal/app/app.go` in `Run()`, stopped in strict reverse order by `Close()`:**

| Runner | Package | Role |
|--------|---------|------|
| **RedisIngest** | `internal/poller` | Subscribe to CPA usage events (Redis pub/sub or list queue), write to `redis_usage_inboxes`, backoff on failure (1s→30s) |
| **RedisProcess** | `internal/poller` | Read from `redis_usage_inboxes`, token processing, write to `usage_events`, at-least-once |
| **UsageAggregation** | `internal/poller` | Serial runner with per-pipeline checkpoints (overview hourly/daily, activity, latency stats) |
| **MetadataSync** | `internal/app` | Sync Auth Files / API Keys / AI Providers / pricing from CPA (interval + control-message triggered) |
| **QuotaAutoRefresh** | `internal/quota` | Provider quota refresh workers (manual + auto), shared queue |
| **Ranking** | `internal/ranking` | Opt-in community ranking client; inert unless enabled |
| **Maintenance** | `internal/app` | Retention (90-day hot/cold archive move), log cleanup |
| **BackupMaintenance** | `internal/app` | Scheduled SQLite backups |
| **CodexQuotaHistory** | `internal/quota` | Codex quota history scheduler + runner (flush, cleanup) |
| **UsageHeaderSnapshot** | `internal/quota` | Batch cache writes for usage response header snapshots |

**Failure isolation:** Optional subsystems (ranking, update check, quota, IP enrichment) must never break ingestion. Follow "inert when disabled" and failure-tolerant-cache patterns.

---

## 6. Layering Rules (Backend)

1. **Handlers are thin:** parse/validate → call service → map DTO. No business logic in `internal/api`.
2. **All SQL lives in `internal/repository`** — never above it. Handlers and services never see GORM types.
3. **Repository pattern per entity family; composition over inheritance.** Services and runners are composed in `internal/app/app.go` via constructor injection.
4. **`internal/entities` is the schema source of truth.** Schema changes go through additive migrations in `internal/repository/migration`.
5. **Single writer.** Writes go through the writer pool; reads use the reader pool. Aggregation checkpoint updates stay inside the caller's transaction with the name + expected-cursor optimistic-concurrency check.
6. **Failure isolation.** Optional subsystems must never break ingestion.
7. **Redaction.** Never log secrets — route anything key-like through `internal/helper/redact.go`.
8. **Test placement.** Co-locate `*_test.go` with the code; tests that need isolation live in a `test/` subpackage (e.g. `internal/api/test/`).

---

## 7. Frontend Conventions

- **TypeScript strict.** No `any` without justification; `typecheck` must pass.
- **No UI framework.** Extend the bespoke kit in `src/components/ui`; do not add a component-library dependency without discussion.
- **Styling:** CSS Modules per component; global layers via SCSS (dart-sass). Theming uses the `data-theme` attribute — components must look correct in light, dark, and auto.
- **i18n:** every user-facing string goes through i18next (`src/i18n`); `en`, `zh`, and `zh-TW` locale files must all be updated together.
- **Charts:** Chart.js registered only via `src/lib/chartjs.ts`.
- **Provider Model GNN panel (permanent feature):** `ProviderModelGNNPanel` must stay mounted on the Usage overview tab. Removal or replacement requires an explicit documented architectural decision recorded in `docs/ARCHITECTURE.md`, `docs/SPECIFICATION.md` (UC-12), and `session.md`.

---

## 8. Extensibility Points

| Extension | Hook |
|-----------|------|
| New quota provider | Add a client under `internal/quota`, register in `registry.go` |
| New ingest source | Implement the `IngestSource` interface in `internal/poller` |
| New analytics | Add a store under `internal/repository` + extend the aggregation runner with a new checkpoint name |
| New API surface | Thin handler in `internal/api` + service in `internal/service` |
| Storage backends | Repository layer isolates GORM/dialect specifics (single seam for a future multi-DB port) |
| UI features | Pages under `web/src/pages`, shared widgets in `components/ui`, feature folders like `features/ranking` |

---

## 9. Key Files for Agent Onboarding

| File | Purpose |
|------|---------|
| `AGENTS.md` | AI agent guidance: project overview, layering rules, verification baseline, security notes |
| `docs/ARCHITECTURE.md` | Detailed architecture: runtime topology, module layering, data flow, deployment models |
| `docs/SPECIFICATION.md` | Functional contract: purpose, users, use cases, business rules, security requirements |
| `docs/CONFIGURATION.md` | All environment variables and CLI flags |
| `docs/ROADMAP.md` | Phased future work, planned features, maintenance track |
| `docs/CONTRIBUTING.md` | Development setup, verification baseline, commit conventions, documentation policy |
| `docs/CODE_GRAPH.md` | **This file** — project map for quick orientation |
| `session.md` | AI session context: recent decisions, discoveries, pending work |
| `.env.example` | Configuration template with placeholders |
| `Makefile` | Common tasks: `make verify`, `make build`, `make test` |

---

## 10. Verification Baseline

Run before opening a PR:

```bash
make verify
```

Which executes:

```bash
go test ./cmd/... ./internal/...      # backend tests (Go 1.26, CGO on)
npm --prefix ./web ci                 # install web deps
npm --prefix ./web run test           # vitest
npm --prefix ./web run lint           # eslint
npm --prefix ./web run typecheck      # tsc --noEmit (strict)
npm --prefix ./web run build          # vite production build
```

**Note:** The production capacity suite under `internal/benchmark/` is **not** part of the baseline. It is a Linux-only, on-demand harness (amd64 + cgroup v2 + systemd) run via `internal/benchmark/scripts/run-capacity.sh`.

---

## 11. Related Documents

- [README.md](../README.md) — quick start & deployment recipes
- [ARCHITECTURE.md](ARCHITECTURE.md) — module map, wiring, data flow (detailed)
- [SPECIFICATION.md](SPECIFICATION.md) — functional contract, use cases, business rules
- [CONFIGURATION.md](CONFIGURATION.md) — env vars & CLI flags reference
- [ROADMAP.md](ROADMAP.md) — phased future work
- [CONTRIBUTING.md](CONTRIBUTING.md) — dev workflow & conventions
- [AGENTS.md](../AGENTS.md) — AI agent guidance summary
- [CHANGELOG.md](../CHANGELOG.md) — notable changes
- [session.md](../session.md) — AI session context (kept current)

---

**End of Code Graph. Use this as your primary reference when exploring the codebase.**
