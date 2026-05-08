---
read_when:
  - adding a CLI subcommand or flag
  - changing bootstrap, migrations, backup, or export behavior
---

# CLI

The single binary is `clickclack`. Source: `apps/api/cmd/clickclack/main.go`.

```text
clickclack <command> [flags]

Commands:
  serve      run the HTTP/WebSocket server (default if no command given)
  migrate    apply embedded SQL migrations
  admin      bootstrap, user create, invite create, magic-link create
  backup     write a SQLite backup file
  export     write a JSON dump to a file or stdout
```

## `serve`

```sh
clickclack serve \
  --addr :8080 \
  --data ./data \
  --db sqlite://./data/clickclack.db \
  --config ./clickclack.json \
  --dev-bootstrap=true
```

- Loads config from `--config` (JSON) layered on top of env vars (`CLICKCLACK_*`).
  CLI flags win when explicitly set. See [configuration.md](configuration.md).
- Creates `<data>`, `<data>/uploads`, `<data>/logs`.
- Opens SQLite (modernc) with WAL, foreign keys, and `busy_timeout=5000`,
  then runs migrations.
- When `--dev-bootstrap=true` (the default), creates a `Local Captain` user
  and a `ClickClack` workspace if the DB is empty. Disable in production.
- Logs the resolved listen URL and the dev-auth user ID.

## `migrate`

```sh
clickclack migrate --data ./data --db sqlite://./data/clickclack.db
```

Idempotent: each migration in
`apps/api/internal/store/sqlite/migrations/` is recorded in
`schema_migrations` and skipped on subsequent runs. Use this in
deployments before flipping traffic to a new build.

## `admin`

### `admin bootstrap`

```sh
clickclack admin bootstrap --name "Peter" --email steipete@gmail.com
```

Creates the first user, workspace, and `general` channel if none exist.
Idempotent — re-running prints the existing user's ID. Output is the
`usr_...` ID on stdout, ready to capture in a shell script.

### `admin user create`

```sh
clickclack admin user create --name "Ari" --email ari@example.com [--workspace wsp_...]
```

Creates a user. With `--workspace`, also adds them to that workspace as a
`member`. Prints the new user ID.

### `admin invite create`

```sh
clickclack admin invite create --workspace wsp_...
```

Mints an invite token (created by the first user in the DB, which is the
owner in single-tenant deployments). Prints the token. There is no consume
endpoint over HTTP yet — invite tokens are reserved for V1 work.

### `admin magic-link create`

```sh
clickclack admin magic-link create --email steipete@gmail.com --name "Peter"
```

Mints a magic-link token. Hand it to the user; they POST it to
`/api/auth/magic/consume` to get a session. See
[features/auth.md](features/auth.md).

## `backup`

```sh
clickclack backup --data ./data --out ./data/backup.db
```

Uses SQLite's online backup API to write a hot copy of the database. Safe to
run while `serve` is up. The destination must be on the same filesystem as
`<data>` if you want a fast atomic move afterwards.

## `export`

```sh
clickclack export --data ./data --out ./data/export.json
clickclack export --out -                # stdout
```

Writes a JSON dump of users, workspaces, channels, messages, threads,
reactions, uploads metadata, and DMs. Useful for migrations between SQLite
files or for one-off audits.

## Exit codes

`0` on success, non-zero on any error (a bare `log.Fatal` in `main`). Scripts
should rely on the exit code, not the log line format.
