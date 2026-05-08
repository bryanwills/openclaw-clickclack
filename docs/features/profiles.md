---
read_when:
  - changing user profile fields, account settings UI, or /api/me
---

# Profiles

Each user has a display name, optional handle, and optional avatar URL.

The handle is the human-friendly short name shown as `@name` in the app. The
API accepts it with or without the leading `@`, normalizes it to lowercase, and
stores it without the `@`.

## API

```http
GET /api/me

PATCH /api/me
{
  "display_name": "Peter Steinberger",
  "handle": "@steipete",
  "avatar_url": "https://example.com/avatar.png"
}
```

`PATCH /api/me` returns `{ "user": ... }`. Handles must be unique when set and
must be 2-32 characters using letters, numbers, `_`, or `-`. Avatar URLs can be
blank or an `http`/`https` URL.

## App

The current user's profile control sits at the bottom of the channel sidebar.
Click or right-click it to open account settings and edit display name, handle,
and avatar URL.

Message lists, search results, threads, DMs, and the profile control all hydrate
avatars from the user attached to each message or conversation member.
