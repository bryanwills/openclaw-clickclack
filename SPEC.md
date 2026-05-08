# ClickClack Spec

ClickClack is a self-hostable, API-first chat app for internal testing, small teams, and communities. It mixes Slack-style productivity with Discord-style warmth, plus a light crustacean theme.

## Goals

- Run as a tiny single binary with first-class SQLite storage.
- Offer a hosted/server deployment path with Postgres later.
- Provide reliable realtime text chat with Slack-style threads.
- Keep the backend API-first and frontend-framework-independent.
- Ship a TypeScript SDK for bots, integrations, and community tooling.
- Feel playful and memorable without sacrificing dense, practical chat workflows.

## Non-Goals For V1

- Voice/video rooms.
- Full Slack, Discord, or Mattermost server compatibility.
- Federation.
- End-to-end encryption.
- Enterprise compliance features.
- Multi-node websocket fanout.

## Product Shape

### Naming

- Product: ClickClack.
- Primary domain: `clickclack.chat`.
- Backend/protocol codename, if needed: Clawwire.
- Theme: lobster/crustacean accents, not renamed core UX primitives.

### First Users

- Internal testing groups.
- Self-hosted teams.
- Small communities.
- Bot-heavy hacker spaces.

### UX Model

- Multi-workspace.
- Workspace contains channels.
- Channel timeline shows root messages only.
- Every root message can have one Slack-style thread.
- Thread opens in a right-side pane.
- Thread replies are one-level only; no nested reply trees.
- Presence and typing are ephemeral.
- Light/dark themes from day one.

Use familiar terms for core navigation:

- Workspace
- Channel
- Thread
- Message
- Reaction
- Bot

Use crustacean flavor in:

- Logo/mascot.
- Empty states.
- Loading states.
- Reaction pack.
- Sounds.
- Onboarding copy.
- Optional statuses like `molting`, `lurking`, `afk`.

## V1 Vertical Slice

The first useful build should support:

- Create/select workspace.
- Create/select channel.
- Send text message.
- Realtime message delivery over WebSocket.
- Open message thread in right pane.
- Send thread reply.
- Persist everything in SQLite.
- Reload/reconnect and recover state.
- Basic dev/local auth.
- Embedded web app served by Go.

## Architecture

```text
clickclack/
  apps/
    api/              # Go backend and single-binary entrypoint
    web/              # Svelte SPA
  packages/
    protocol/         # OpenAPI spec and event schemas
    sdk-ts/           # TypeScript SDK, generated client + friendly wrapper
  docs/
    architecture/
    api/
  infra/
    migrations/
      sqlite/
      postgres/       # later
```

## Backend

Language: Go.

Initial runtime:

- Single Go process.
- `modernc.org/sqlite`.
- Embedded migrations.
- Embedded Svelte build via `go:embed`.
- Local upload storage.
- In-process websocket hub.

Future hosted runtime:

- Postgres.
- Object storage.
- External queue/pubsub only when needed.
- Multi-node websocket fanout later.

### Suggested Go Libraries

- HTTP router: `chi`.
- SQLite: `modernc.org/sqlite`.
- Postgres later: `pgx`.
- Queries: start handwritten or `sqlc` once schema settles.
- Migrations: embedded SQL migrations with a tiny internal runner, or `goose` if the runner grows.
- IDs: UUIDv7 or ULID.

### CLI

```text
clickclack serve
  --addr :8080
  --data ./data
  --db sqlite://./data/clickclack.db

clickclack migrate
  --db sqlite://./data/clickclack.db

clickclack admin invite
```

Default `clickclack serve` should be enough for local use.

## Frontend

Framework: Svelte 5 SPA.

Use plain Svelte + Vite unless SvelteKit offers clear value without adding server-side complexity. The Go server owns HTTP/API/auth and serves static assets.

Frontend responsibilities:

- Render workspace/channel/thread UI.
- Keep local client cache/projection.
- Use HTTP API for writes and fetches.
- Use WebSocket for realtime events.
- Recover by refetching from API after reconnect.

Frontend should not own durable chat truth.

## API

Contract: OpenAPI first.

Source of truth:

```text
packages/protocol/openapi.yaml
```

Generate:

- Go request/response types or validators where useful.
- TypeScript API client.
- SDK docs.

Initial REST shape:

```text
GET    /api/me

GET    /api/workspaces
POST   /api/workspaces
GET    /api/workspaces/{workspace_id}

GET    /api/workspaces/{workspace_id}/channels
POST   /api/workspaces/{workspace_id}/channels

GET    /api/channels/{channel_id}/messages?before=&after_seq=&limit=
POST   /api/channels/{channel_id}/messages

GET    /api/messages/{message_id}/thread
POST   /api/messages/{message_id}/thread/replies

POST   /api/messages/{message_id}/reactions
DELETE /api/messages/{message_id}/reactions/{emoji}

GET    /api/realtime/events?after_cursor=
GET    /api/realtime/ws
```

## Realtime

Realtime must be recoverable.

Rules:

- WebSocket is a notification/update pipe.
- SQLite/Postgres is source of truth.
- Every durable event is recoverable through HTTP.
- Client reconnects with last seen cursor.
- If cursor is too old or unknown, server returns `resync_required`.

Send flow:

1. Client calls `POST /api/channels/{id}/messages`.
2. Server validates auth and membership.
3. Server transaction:
   - insert message
   - assign per-channel sequence
   - insert event into outbox/events table
   - update thread/channel summary state
4. In-process dispatcher broadcasts event to websocket subscribers.
5. Client reconciles optimistic message with server event.

Event shape:

```json
{
  "id": "evt_...",
  "cursor": "...",
  "type": "message.created",
  "workspace_id": "w_...",
  "channel_id": "c_...",
  "seq": 124,
  "created_at": "2026-05-08T12:00:00Z",
  "payload": {
    "message_id": "m_..."
  }
}
```

Initial durable events:

- `message.created`
- `message.updated`
- `message.deleted`
- `thread.reply_created`
- `thread.state_updated`
- `reaction.added`
- `reaction.removed`
- `channel.created`
- `channel.updated`

Ephemeral events:

- `typing.started`
- `typing.stopped`
- `presence.changed`

Ephemeral events are not persisted and may be dropped.

## Data Model

Initial tables:

```text
users
  id
  display_name
  avatar_url
  created_at

identities
  id
  user_id
  provider
  provider_subject
  email
  created_at

workspaces
  id
  name
  slug
  created_at

workspace_members
  workspace_id
  user_id
  role
  created_at

channels
  id
  workspace_id
  name
  kind
  created_at
  archived_at

messages
  id
  workspace_id
  channel_id
  author_id
  parent_message_id
  thread_root_id
  channel_seq
  thread_seq
  body
  body_format
  created_at
  edited_at
  deleted_at

thread_state
  root_message_id
  reply_count
  last_reply_at
  last_reply_author_ids_json

reactions
  message_id
  user_id
  emoji
  created_at

events
  id
  cursor
  workspace_id
  channel_id
  type
  payload_json
  created_at
```

Thread rules:

- Root message has `parent_message_id = null`.
- Root message has `thread_root_id = id`.
- Thread reply has `parent_message_id = root_message_id`.
- Thread reply has `thread_root_id = root_message_id`.
- No nested replies in V1.

## Storage

SQLite is first-class.

SQLite requirements:

- Use `modernc.org/sqlite`.
- Enable WAL mode.
- Use a single writer discipline.
- Keep transactions short.
- Prefer portable SQL.
- Avoid Postgres-only behavior in core paths.
- Add separate Postgres migrations later rather than forcing one dialect.

Local file layout:

```text
data/
  clickclack.db
  uploads/
  logs/
```

## Auth

V0:

- Dev/local auth for quick testing.
- Owner bootstrap on first run.

V1:

- Magic links.
- GitHub OAuth.
- Optional local email/password only if needed for fully offline/self-hosted deployments.

Auth principles:

- Workspace membership checked on every API write.
- WebSocket subscribe validates workspace/channel access.
- Recheck permissions for channel/thread fetches.

## SDK

First SDK: TypeScript.

Location:

```text
packages/sdk-ts
```

Layering:

- Generated OpenAPI client.
- Friendly wrapper.
- WebSocket/event subscription helper.

Example API:

```ts
const client = new ClickClackClient({ baseUrl, token });

await client.channels.sendMessage(channelId, {
  body: "click clack",
});

client.events.subscribe({
  workspaceId,
  onEvent(event) {
    // handle event
  },
});
```

SDK must not depend on Svelte.

## Mattermost Compatibility

Do not clone the full Mattermost API in V1.

Do support:

- Incoming webhook compatibility.
- Simple slash-command callback shape.
- Import helpers for exports if useful.

Do not support early:

- Existing Mattermost clients connecting directly.
- Full REST API compatibility.
- Full permission/model compatibility.

## Design Direction

ClickClack should feel:

- Fast.
- Dense.
- Friendly.
- Slightly weird.
- More polished tool than joke app.

Visual direction:

- Light and dark themes.
- Neutral UI base.
- Coral, shell, brine, ink accents.
- Crustacean mascot and iconography used sparingly.
- Avoid novelty typography.
- Avoid making normal controls hard to understand.

UI layout:

```text
left sidebar: workspaces / channels
center: channel timeline
right pane: thread
bottom: composer
top: channel title, members, search
```

## Development Milestones

### M0: Skeleton

- Monorepo.
- Go server boots.
- Svelte app builds.
- Go embeds and serves web assets.
- SQLite opens and migrates.

### M1: Durable Chat

- Workspaces/channels/messages schema.
- REST create/list messages.
- Basic dev auth.
- Message timeline UI.

### M2: Realtime

- WebSocket endpoint.
- Event outbox.
- Live message updates.
- Reconnect and cursor recovery.

### M3: Threads

- Root messages and one-level replies.
- Thread pane.
- Thread reply counts and last reply state.

### M4: Self-Host Polish

- First-run owner setup.
- Config file/env.
- Local upload storage.
- Docker image.
- Backups/export.

### M5: SDK

- OpenAPI generation.
- TypeScript SDK.
- Incoming webhooks.
- Basic bot example.

## Open Questions

- Should `clickclack serve` expose setup UI on first run or require CLI owner bootstrap?
- How much markdown/rich text in V1?
- Do we need DMs in V1, or only channels and threads?
- Should search start with SQLite FTS5?
- Should uploads exist in V1 or wait until core chat is solid?
- Which generated OpenAPI toolchain is least annoying for Go plus TypeScript?
