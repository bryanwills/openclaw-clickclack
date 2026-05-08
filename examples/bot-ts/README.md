# ClickClack Bot Example

Tiny TypeScript bot using `@clickclack/sdk-ts`.

```sh
CLICKCLACK_URL=http://localhost:8080 \
CLICKCLACK_USER_ID=user_dev \
CLICKCLACK_CHANNEL_ID=chan_... \
CLICKCLACK_TEXT="clack from bot" \
pnpm --filter @clickclack/example-bot start
```

Use `CLICKCLACK_TOKEN` instead of `CLICKCLACK_USER_ID` when running with a bearer session token.
