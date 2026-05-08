---
read_when:
  - changing backend storage, realtime events, auth, or embedded serving
  - adding a second database implementation
---

# Architecture Overview

ClickClack ships as one Go binary serving a Svelte SPA, JSON API, websocket
endpoint, embedded SQLite migrations, local SQLite data, and local upload
files.

Durable state lives in SQLite. WebSockets are an update pipe only; clients
recover missed durable events through `GET /api/realtime/events?after_cursor=...`
or by reconnecting the websocket with a cursor.

## Layers

- `apps/api/cmd/clickclack` — CLI and single-binary entrypoint. See
  [../cli.md](../cli.md).
- `apps/api/internal/httpapi` — chi router, auth resolution, REST/WS
  handlers, SPA serving.
- `apps/api/internal/store` — backend-facing store contract and domain types
  (`Store` interface in `types.go`).
- `apps/api/internal/store/sqlite` — SQLite implementation, embedded SQL
  migrations, search/FTS, backup, JSON export.
- `apps/api/internal/realtime` — in-process workspace event hub (`Hub`).
- `apps/api/internal/config` — flag/env/file resolution.
- `apps/api/internal/webassets` — `go:embed` for the built SPA.
- `apps/web` — Svelte 5 SPA, API-only client behavior.
- `packages/protocol` — OpenAPI contract, source of truth.
- `packages/sdk-ts` — generated OpenAPI types plus framework-neutral
  TypeScript wrapper.

## Storage rules

- SQLite uses `modernc.org/sqlite` and WAL.
- Foreign keys on; `busy_timeout=5000`.
- Single writer discipline: `db.SetMaxOpenConns(1)`.
- Transactions stay short. Outbox `events` rows are inserted in the same
  commit as the durable write that produced them, so subscribers can't see a
  message that isn't in the DB.
- IDs are sortable ULID text with semantic prefixes (see
  [../data-model.md](../data-model.md)).
- Postgres should be added behind the store layer without changing API
  handlers.

## Cross-cutting docs

- [../api/overview.md](../api/overview.md) — REST + WS surface.
- [../data-model.md](../data-model.md) — tables, IDs, invariants.
- [../features/realtime.md](../features/realtime.md) — durable vs ephemeral
  events, cursor recovery.
- [../features/auth.md](../features/auth.md) — auth resolution and
  precedence.
