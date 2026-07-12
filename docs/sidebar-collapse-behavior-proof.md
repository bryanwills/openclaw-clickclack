# Collapsible sidebar behavior proof

## Environment

- Original recording branch: `codex/collapsible-sections-pr`
- ClickClack server: locally built production binary serving the existing `/tmp/clickclack-data` database on port `18082`
- OpenClaw: `2026.7.2`; gateway `/healthz` and `/readyz` passed after the ClickClack restart
- Connected accounts: blackbird, clipper, dragon-lady, mustang, and nighthawk all reported `enabled, configured, running`
- Recording: [`docs/proof/collapsible-sidebar-real-behavior.mov`](https://github.com/openclaw/clickclack/blob/main/docs/proof/collapsible-sidebar-real-behavior.mov), captured against the running candidate

The live server binary and SQLite database were backed up before the candidate was started. No schema migration or data reset was required.

## Recorded behavior

The eight-second recording demonstrates the real sidebar with the existing ClickClack workspace, channels, connected status, and Blackbird person entry. It shows:

1. Channels, Direct messages, and People remain separate sidebar sections.
2. Each section heading is an interactive disclosure with a directional caret.
3. Collapsing a section changes only that section's rows.
4. Other sections remain visible and interactive.
5. Create-channel and start-DM controls remain available independently of disclosure state.
6. The controls and spacing remain usable in the narrow sidebar layout.

The recording predates the priority-row refinement below, so automated coverage is authoritative for which rows remain visible while a section is collapsed.

## Automated behavior coverage

`tests/e2e/sidebar-collapse.spec.ts` provides deterministic coverage for behavior that is difficult to show in one short recording:

- all sections default expanded with `aria-expanded=true` and stable `aria-controls` targets;
- each section collapses independently;
- collapsed Channels retain the active channel and unread channels while hiding read inactive channels;
- collapsed Direct messages retain the active conversation and unread conversations while hiding read inactive conversations;
- unread row badges remain visible, including the `99+` cap;
- collapsed People hides its rows;
- add-channel and add-DM actions remain available while collapsed;
- state survives reloads and remains isolated per workspace;
- direct URL navigation does not override an explicit collapsed preference;
- malformed or unavailable browser storage falls back safely;
- the same disclosure behavior works in the mobile navigation drawer.

## Reproduction

```sh
pnpm install --frozen-lockfile
pnpm build
go build -o /tmp/clickclack-sidebar ./apps/api/cmd/clickclack
/tmp/clickclack-sidebar serve --addr :18082 --data /tmp/clickclack-data --dev-bootstrap=true
```

Open `http://localhost:18082/app`, create or receive unread activity in a channel and direct message, then collapse those sections. Verify the active and unread rows remain visible while read inactive rows are hidden. Reload and verify the selected workspace restores each section's state. Switch workspaces to verify the disclosure preferences are independent.

The feature is client-only. It does not change ClickClack APIs, storage schemas, message delivery, or the OpenClaw channel contract.
