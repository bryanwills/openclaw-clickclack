# ClickClack Bot Example

Tiny TypeScript bot using `@clickclack/sdk-ts`.

```sh
CLICKCLACK_URL=http://localhost:8080 \
CLICKCLACK_TOKEN=ccb_... \
CLICKCLACK_CHANNEL_ID=chn_... \
CLICKCLACK_TEXT="clack from bot" \
pnpm --filter @clickclack/example-bot start
```

Use `CLICKCLACK_USER_ID=usr_dev` only for local dev-auth servers. Hosted bots
should use a scoped `ccb_...` token from `clickclack admin bot create`.
