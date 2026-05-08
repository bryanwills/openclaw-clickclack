# Changelog

## Unreleased

- Added inline quote-replies in channels, DMs, and threads. Every
  message-create endpoint now accepts an optional `quoted_message_id`; the
  server captures a 280-rune trimmed snapshot of the quoted body plus the
  quoted author at send time, and rejects cross-context quotes with
  `HTTP 400`. The chat UI exposes a hover-revealed "Quote" affordance,
  composer chip, click-to-jump quote block, and an "[original deleted]"
  fallback when the source is hard-deleted (FK is `ON DELETE SET NULL` so
  the snapshot survives). The `clickclack send` and `clickclack threads
  reply` CLI commands gain `--reply-to msg_...` flags. See
  [docs/features/replies.md](docs/features/replies.md). Thanks @shakkernerd.
- Refined the chat app shell with denser Slack/Discord-style navigation,
  grouped message timelines, clearer empty states, responsive sidebars, and a
  send button that no longer appears active for attachment-only drafts.
- Added richer Slack-like chat affordances: animated sidebar/thread panels,
  inline image attachment cards, Markdown composer controls, and a GIF picker
  that inserts animated GIF Markdown.
- Added Slack-style user profile side panes, automatic People shortcuts in the
  sidebar, inline video playback, tighter message spacing, and an image viewer
  modal for inline images.
- Added a public product website at the web root while keeping the chat app at
  `/app` locally and on `app.clickclack.chat` when served from that host.
- Added an agent-friendly ClickClack client mode to the Go binary with
  `login`, `logout`, `whoami`, `status`, workspace/channel listing, message
  send/list, and thread open/reply commands.
- Scoped stored CLI credentials and workspace/channel defaults to the saved
  server URL, with `--user` / `CLICKCLACK_USER_ID` taking precedence over
  stored bearer tokens unless `--token` is explicitly supplied.
- Documented the `clickclack.chat` product domain, `app.clickclack.chat` app
  domain, `docs.clickclack.chat` docs domain, and recommended bearer-token auth
  flow for hosted agents.
- Added more visible GitHub links to the product website and improved the docs
  quickstart CTA contrast in dark mode.
- Split GitHub Actions into explicit Go, TypeScript, Playwright, and Docker
  gates, with `gofmt` and `oxfmt --check` enforced in CI.
- Added GoReleaser config and release workflow for Linux, macOS, Windows, and
  FreeBSD archives, plus Linux `.deb` and `.rpm` packages.
