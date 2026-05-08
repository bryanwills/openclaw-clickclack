---
read_when:
  - changing backend storage, realtime events, auth, or embedded serving
  - adding a second database implementation
---

# Architecture Overview

ClickClack ships as one Go binary serving a Svelte SPA, JSON API, websocket endpoint, embedded SQLite migrations, local SQLite data, and local upload files.

Durable state lives in SQLite. WebSockets are an update pipe only; clients recover missed durable events through `GET /api/realtime/events?after_cursor=...`.

Core layers:

- `apps/api/cmd/clickclack`: CLI and single-binary entrypoint.
- `apps/api/internal/httpapi`: chi routes, auth, API handlers, SPA serving.
- `apps/api/internal/store`: backend-facing store contracts and domain types.
- `apps/api/internal/store/sqlite`: SQLite implementation, embedded migrations, backups, exports.
- `apps/api/internal/realtime`: in-process workspace event hub.
- `apps/web`: Svelte 5 SPA with API-only client behavior.
- `packages/protocol`: OpenAPI contract.
- `packages/sdk-ts`: generated OpenAPI types plus framework-neutral TypeScript wrapper.

Storage constraints:

- SQLite uses `modernc.org/sqlite` and WAL.
- Transactions stay short and issue outbox events inside the same commit as durable writes.
- IDs are sortable ULID text with semantic prefixes.
- Postgres should be added behind the store layer without changing API handlers.
