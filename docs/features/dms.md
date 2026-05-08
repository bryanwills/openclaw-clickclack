---
read_when:
  - changing direct message conversations or DM listing
---

# Direct Messages

DMs are workspace-scoped multi-party conversations. They reuse the `messages`
table — every DM message sets `direct_conversation_id` and leaves
`channel_id` null.

## Endpoints

```http
GET  /api/dms?workspace_id=                              # caller's conversations in a workspace
POST /api/dms                                            # { workspace_id, member_ids }
GET  /api/dms/{conversation_id}/messages?after_seq=&limit=
POST /api/dms/{conversation_id}/messages                 # { body }
```

Conversations include their members hydrated from `users`. The `member_ids`
list on create is deduplicated and the caller is added automatically.

The web sidebar lists existing DMs and also derives a People section from DM
members and hydrated message authors. Users appear there automatically as
conversation context is loaded; clicking a person opens their DM when one
exists, otherwise it opens the profile pane with a Message action.

`POST` to `/dms/{id}/messages` increments a per-conversation sequence on
`messages.channel_seq` and emits a durable event into the workspace event
stream so DM lists and unread counts stay live.

## Membership

- Listing conversations requires workspace membership.
- Sending a DM requires membership in the conversation
  (`direct_conversation_members`).
- DM creation requires that all `member_ids` are members of the same
  workspace.

## What is intentionally missing

- DM-only auth tokens.
- One-on-one vs group distinctions in the API surface — the client decides
  based on member count.
- Message reactions/edits in DMs. The schema supports it, but the API surface
  for DM-specific mutations is intentionally minimal in V1.
