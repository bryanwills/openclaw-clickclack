# ClickClack

Self-hostable chat with Slack-style threads, Discord-ish warmth, and light crustacean seasoning.

See [SPEC.md](SPEC.md) for the initial product and architecture plan.

## Development

```sh
pnpm install
pnpm build
go run ./apps/api/cmd/clickclack serve
```

Open http://localhost:8080.

Useful commands:

```sh
go test ./...
pnpm -r typecheck
pnpm lint
pnpm coverage
pnpm test:e2e
pnpm build
go run ./apps/api/cmd/clickclack admin bootstrap --name "Peter" --email steipete@gmail.com
go run ./apps/api/cmd/clickclack admin magic-link create --email steipete@gmail.com --name "Peter"
go run ./apps/api/cmd/clickclack backup --out ./data/backup.db
go run ./apps/api/cmd/clickclack export --out ./data/export.json
pnpm --filter @clickclack/example-bot start
```

TypeScript uses `tsgo` from `@typescript/native-preview`; formatting/linting use `oxfmt` and `oxlint`.
Local auth supports dev fallback, `X-ClickClack-User`, bearer session tokens, and CLI-generated magic-link tokens.
Optional GitHub OAuth is enabled with `CLICKCLACK_PUBLIC_URL`, `CLICKCLACK_GITHUB_CLIENT_ID`, and `CLICKCLACK_GITHUB_CLIENT_SECRET`.
The bot example in `examples/bot-ts` uses the framework-neutral SDK and the same auth headers as the web app.
