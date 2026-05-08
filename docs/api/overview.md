---
read_when:
  - changing REST endpoints, websocket behavior, SDK methods, or OpenAPI
  - adding integrations or bots
---

# API Overview

`packages/protocol/openapi.yaml` is the API contract source of truth.

Main API groups:

- Auth: local dev headers, bearer sessions, magic-link token request/consume, optional GitHub OAuth.
- Workspaces/channels: create/list workspaces and create/list/update channels.
- Messages: create/list/update/soft-delete root channel messages, reactions, attachments.
- Threads: one-level thread replies per root message.
- Realtime: websocket notifications, HTTP event recovery by cursor, and non-durable typing/presence publish.
- Search/uploads/DMs: SQLite FTS5 search, local upload storage, direct conversations.
- Integrations: Mattermost-compatible incoming webhook shape and simple slash command callbacks.

TypeScript consumers should use `@clickclack/sdk-ts`. The SDK has no Svelte dependency and exposes HTTP helpers plus `events.subscribe(...)`.

The bot example in `examples/bot-ts` sends a channel message using the SDK and either `CLICKCLACK_TOKEN` or `CLICKCLACK_USER_ID`.
