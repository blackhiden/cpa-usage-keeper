# CPA Usage Keeper Schema

`internal/entities` is the schema source of truth. SQLite schema changes are
applied through ordered additive migrations in `internal/repository/migration`.
The application stores its database at `WORK_DIR/app.db` in this deployment;
fresh default deployments use the path documented in `docs/CONFIGURATION.md`.

## Core tables

| Table | Purpose |
| --- | --- |
| `usage_events` | Durable raw CPA usage events. |
| `redis_usage_inboxes` | At-least-once ingest inbox before event processing. |
| `cpa_api_keys` | CPA API-key metadata, display data, and local-ranking profile fields. |
| `usage_identities` | Aggregated credential and provider identity data. |
| `usage_overview_hourly_stats`, `usage_overview_daily_stats`, `usage_activity_stats`, `usage_latency_stats` | Derived analytics rollups. |
| `usage_aggregation_checkpoints` | Optimistic-concurrency progress cursors for rollup pipelines. |
| `model_ranking_period_stats` | Checkpointed model/provider/API-key contributions by day, ISO week, month, and five-minute peak bucket. The empty `scope_key` is the global admin population; an API group key scopes viewer rows. |
| `model_ranking_aggregation_checkpoints` | Incremental model-ranking raw-event cursor. Rollup rows and cursor advance in one writer transaction. |
| `app_settings` | Typed application state, including ranking state. |

## Local ranking storage

`local_ranking_period_stats` stores one aggregate per API key for each active
local-ranking period: today, yesterday, the current month, and the previous
month. Its composite primary key is `(period_kind, period_key, api_key_id)`.
The local leaderboard joins these rows to `cpa_api_keys` for display metadata.
The `ranking.local.period_state` JSON setting in `app_settings` records the
last settled complete day.

The `20261022_restore_local_ranking` migration recreates this table and the
nullable `cpa_api_keys.local_ranking_avatar_id` column after the v2 cleanup.
Current-day and current-month snapshots are created on the first applicable
leaderboard read when absent, then refreshed by the five-minute background
runner.

## Model ranking expansion storage

Migration `20261119_model_ranking_rollups` creates the model-ranking rollup and
checkpoint tables. Each retained usage event contributes to global and, when an
API group key exists, scoped rows for day/week/month periods. Provider is a row
dimension; the service folds providers into alias-collapsed models for display.
Known costs are summed and rows without an explicitly available cost increment
`unavailable_cost_count`; missing cost is never interpreted as zero.

The model-ranking cursor is advanced only after the corresponding rollup page
commits. Archive cleanup treats the rollup as an additional catch-up boundary
when its table exists, while legacy fixtures/databases without the table retain
the prior three-checkpoint behavior. The API-key ranking period state records
the project timezone and clears stale local snapshots when `TZ` changes so the
next aggregation settles them under the new calendar.

## Migration and operational rules

- Migrations are forward-only and recorded in `schema_migrations`.
- SQLite uses one writer connection and a read pool; SQL belongs in
  `internal/repository`.
- Backups contain all persisted data, including raw events and API-key values,
  and must be filesystem-protected.

For entity fields and migration ordering, consult
`/home/denny/Project/cpa/CLIProxyAPI/cpauk/internal/entities` and
`/home/denny/Project/cpa/CLIProxyAPI/cpauk/internal/repository/migration`.