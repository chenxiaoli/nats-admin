# Merge cmd Binaries into Single Binary

## Goal

Merge `cmd/server`, `cmd/migrate`, `cmd/bootstrap` into a single binary `nats-admin` with subcommands.

## Usage

```bash
nats-admin server              # Start HTTP server (default, backward compatible)
nats-admin migrate up <dsn>    # Run database migrations
nats-admin migrate down <dsn>  # Rollback database migrations
nats-admin bootstrap           # One-time operator initialization
```

Running `nats-admin` with no arguments defaults to `server` (backward compatible with current ENTRYPOINT).

## Structure

```
cmd/
└── nats-admin/
    ├── main.go           # Entry point: flag.Parse, subcommand dispatch
    ├── frontend.go       # //go:embed all:frontend_dist (moved from cmd/server/)
    ├── frontend_dist/    # (moved from cmd/server/)
    ├── server.go         # runServer() — extracted from cmd/server/main.go
    ├── migrate.go        # runMigrate() — extracted from cmd/migrate/main.go
    ├── bootstrap.go      # runBootstrap() — extracted from cmd/bootstrap/main.go
```

Old directories `cmd/server/`, `cmd/migrate/`, `cmd/bootstrap/` are deleted.

## Implementation Details

**`main.go`**: parse `os.Args[1]` as subcommand. If empty or `server`, run `runServer()`. If `migrate`, forward remaining args to `runMigrate()`. If `bootstrap`, run `runBootstrap()`.

**Each subcommand file**: contains one exported function (`runServer`, `runMigrate`, `runBootstrap`) with the exact logic from the current `main.go`'s `run()` function. No logic changes.

## Files Changed

| File | Action |
|------|--------|
| `cmd/nats-admin/main.go` | Create — subcommand dispatch |
| `cmd/nats-admin/server.go` | Create — extract from `cmd/server/main.go` |
| `cmd/nats-admin/migrate.go` | Create — extract from `cmd/migrate/main.go` |
| `cmd/nats-admin/bootstrap.go` | Create — extract from `cmd/bootstrap/main.go` |
| `cmd/nats-admin/frontend.go` | Move from `cmd/server/frontend.go` |
| `cmd/nats-admin/frontend_dist/` | Move from `cmd/server/frontend_dist/` |
| `cmd/server/`, `cmd/migrate/`, `cmd/bootstrap/` | Delete |
| `backend/Dockerfile` | Update build target to `./cmd/nats-admin` |
| `backend/CLAUDE.md` | Update dev commands |
| `.github/workflows/release.yml` | Update binary name and build path |
| `docker-compose.yml` | Update migrate entrypoint |

## No Behavior Changes

- `server` subcommand reads the same env vars, same config
- `migrate` subcommand takes the same positional args
- `bootstrap` subcommand outputs the same format
- Docker image still exposes port 8080, runs `server` by default
