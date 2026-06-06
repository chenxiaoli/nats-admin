# NATS Multi-Tenant Admin

A web-based management platform for [NATS](https://nats.io/) that handles the full lifecycle of multi-tenant deployments: account provisioning, user credential issuance/revocation, JetStream resource management, and real-time cluster monitoring.

Built on NATS decentralized JWT authentication model (Operator → Account → User trust chain) with AES-256-GCM encrypted key storage and per-tenant subject isolation.

## Features

- **Tenant Management** — Create, suspend, activate, and delete NATS accounts with configurable limits (connections, subscriptions, JetStream storage/streams/consumers).
- **Credential Issuance** — Generate scoped `.creds` files with custom pub/sub subject permissions. Revocation re-signs the Account JWT and pushes to the resolver immediately.
- **JetStream Admin** — Manage Streams and KV Buckets per tenant with create, list, purge, and delete operations through per-tenant connections.
- **Real-time Monitoring** — Server stats (connections, messages, bytes, uptime) and per-account connection counts via System Account subscriptions, with WebSocket push to the dashboard.
- **Audit Logging** — Append-only log of all admin actions (tenant CRUD, credential operations) with IP tracking.
- **Resolver Reconciliation** — On startup, all active tenant JWTs are re-pushed to the NATS resolver for self-healing after disk loss or fresh containers.

## Architecture

```
Operator NKey (SO...)  ──signs──▶  Account JWT  (per tenant)
                                    │
Account NKey  (SA...)  ──signs──▶  User JWT     (per credential)
                                    │
User NKey     (SU...)  ─embedded─▶ .creds file  (one-time download)
```

- Each tenant gets an isolated NATS Account with its own subject namespace.
- User credentials are scoped by pub/sub permissions within the account.
- Account seeds are AES-256-GCM encrypted at rest; the master key lives in an environment variable.
- The backend connects to NATS as a System Account user for resolver management and monitoring, and as per-tenant users for JetStream operations.

## Tech Stack

| Layer | Choice | Purpose |
|-------|--------|---------|
| Backend | Go 1.22+ | NATS ecosystem has no non-Go alternatives for nkeys/jwt |
| HTTP Router | chi v5 | Lightweight middleware composition |
| Database | PostgreSQL 16 | Tenant metadata, encrypted keys, audit logs |
| DB Access | pgx/v5 | Direct SQL, no ORM |
| Frontend | React 18 + Vite + TypeScript | SPA |
| UI | TailwindCSS | Utility-first styling |
| Data Fetching | TanStack Query v5 | Server state management |
| Admin Auth | JWT HS256 | Separate from NATS JWT chain |
| Seed Encryption | AES-256-GCM | Master key from env |

## Prerequisites

- Go 1.22+
- Node.js 18+ with pnpm
- Docker & Docker Compose (for PostgreSQL and NATS)
- A running NATS 2.10+ server with operator mode enabled

## Quick Start

### 1. Start infrastructure

```bash
docker compose up -d
```

This starts PostgreSQL (port 5432) and NATS (port 4222, monitor port 8222).

### 2. Bootstrap the Operator

First-time only. Generates the Operator NKey, System Account, and writes the initial `.env`:

```bash
cd backend
go run ./cmd/bootstrap/...
```

Follow the output instructions to set the required environment variables.

### 3. Run database migrations

```bash
go run ./cmd/migrate/... up
```

### 4. Start the backend

```bash
go run ./cmd/server/...
```

Server listens on `:8080` by default.

### 5. Start the frontend

```bash
cd ../frontend
pnpm install
pnpm dev
```

Frontend dev server runs on `:5173` with API proxy to `:8080`.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP server port |
| `ENV` | No | `development` | Environment name |
| `DATABASE_URL` | **Yes** | — | PostgreSQL connection string |
| `NATS_URL` | **Yes** | — | NATS server URL (e.g. `nats://localhost:4222`) |
| `RESOLVER_DIR` | No | `/data/nats/resolver` | NATS account resolver directory |
| `OPERATOR_SEED` | **Yes** | — | Operator NKey seed (`SO...`), never stored in DB |
| `SYSTEM_ACCOUNT_SEED` | **Yes** | — | System Account NKey seed (`SA...`) |
| `MASTER_KEY` | **Yes** | — | 64-char hex string (32 bytes) for AES-256-GCM seed encryption |
| `JWT_SECRET` | **Yes** | — | 64+ char hex string for admin JWT signing |
| `JWT_EXPIRY` | No | `24h` | Admin JWT token expiry |
| `BOOTSTRAP_ADMIN_EMAIL` | No | `admin@example.com` | Initial admin user email |
| `BOOTSTRAP_ADMIN_PASSWORD` | No | `changeme` | Initial admin user password |

## API Reference

### Auth

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Admin login, returns JWT |
| POST | `/api/v1/auth/refresh` | Refresh JWT token |

### Tenants

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenants` | List all tenants |
| POST | `/api/v1/tenants` | Create tenant (generates NKey → signs JWT → pushes to resolver → saves to DB) |
| GET | `/api/v1/tenants/:id` | Get tenant details |
| PUT | `/api/v1/tenants/:id` | Update limits (re-signs Account JWT → pushes to resolver) |
| DELETE | `/api/v1/tenants/:id` | Soft-delete tenant (removes from resolver) |
| POST | `/api/v1/tenants/:id/suspend` | Suspend tenant (removes Account JWT from resolver) |
| POST | `/api/v1/tenants/:id/activate` | Reactivate tenant (re-pushes Account JWT) |

### Credentials

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenants/:id/credentials` | List credentials for tenant |
| POST | `/api/v1/tenants/:id/credentials` | Issue credential (returns `.creds` file, one-time) |
| DELETE | `/api/v1/tenants/:id/credentials/:cid` | Revoke credential (re-signs Account JWT with revocation) |
| POST | `/api/v1/tenants/:id/credentials/:cid/rotate` | Rotate credential seed |

### JetStream

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/tenants/:id/jetstream/streams` | List streams |
| POST | `/api/v1/tenants/:id/jetstream/streams` | Create stream |
| DELETE | `/api/v1/tenants/:id/jetstream/streams/:name` | Delete stream |
| POST | `/api/v1/tenants/:id/jetstream/streams/:name/purge` | Purge stream messages |
| GET | `/api/v1/tenants/:id/jetstream/kv` | List KV buckets |
| POST | `/api/v1/tenants/:id/jetstream/kv` | Create KV bucket |
| DELETE | `/api/v1/tenants/:id/jetstream/kv/:bucket` | Delete KV bucket |

### Monitoring

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/monitor/server` | Server stats snapshot |
| GET | `/api/v1/monitor/tenants` | All account stats |
| GET | `/api/v1/monitor/tenants/:id` | Single account stats |
| WS | `/api/v1/ws/monitor` | Real-time stats push (WebSocket) |

## Deployment

### Production Checklist

- **Operator Seed**: Store in a secrets manager (Vault, AWS Secrets Manager), never in container images or compose files.
- **MASTER_KEY**: Rotate by re-encrypting all tenant seeds with the new key.
- **TLS**: Terminate TLS at a reverse proxy (nginx, Caddy) in front of the backend.
- **NATS Server**: Use the provided `nats/server.conf` as a base. In production, enable TLS on client and leafnode ports, and restrict the monitoring endpoint to internal networks.
- **Database**: Use managed PostgreSQL with SSL. Run migrations as part of the deploy pipeline.
- **Resolver Directory**: Mount as a persistent volume. The backend re-seeds on startup, but a warm resolver reduces initial latency.
- **Frontend**: Build with `pnpm build`, serve the `dist/` directory as static files.

### Docker Compose (Production-like)

The included `docker-compose.yml` provides PostgreSQL and NATS for development. For production:

1. Replace `postgres:16-alpine` with a managed database or a hardened Postgres container with SSL and backup.
2. Mount a production `nats/server.conf` with TLS and appropriate `operator` JWT path.
3. Add the backend and frontend as services with proper health checks.

## Project Structure

```
.
├── backend/
│   ├── cmd/
│   │   ├── server/main.go          # HTTP server entrypoint
│   │   ├── migrate/main.go         # DB migration runner
│   │   └── bootstrap/main.go       # One-time Operator initialization
│   ├── internal/
│   │   ├── config/                 # Environment config (viper)
│   │   ├── operator/               # Operator NKey loading, Account JWT signing
│   │   ├── tenant/                 # Account lifecycle, resolver push/delete
│   │   ├── credential/             # User JWT issuance, revocation, rotation
│   │   ├── jetstream/              # Per-tenant NATS connection pool, Stream/KV CRUD
│   │   ├── monitor/                # System Account subscriptions, WebSocket hub
│   │   ├── api/                    # Router, middleware, handlers
│   │   └── db/                     # Migrations, sqlc generated code
│   ├── go.mod
│   └── .env                        # Local env (gitignored)
├── frontend/
│   ├── src/
│   │   ├── api/                    # TanStack Query hooks
│   │   ├── pages/                  # Route components
│   │   ├── components/             # Shared UI components
│   │   └── lib/                    # Utilities
│   ├── package.json
│   └── vite.config.ts
├── nats/
│   └── server.conf                 # NATS server configuration
├── docs/
│   ├── nats-auth.md                # NATS JWT auth model reference
│   └── db-schema.md                # Database schema documentation
├── docker-compose.yml
└── .gitignore
```

## License

[MIT](LICENSE)
