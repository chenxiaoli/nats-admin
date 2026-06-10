# API Key Authentication Design

## Goal

Add opaque API keys alongside existing email/password JWT auth, so backend services can call the nats-admin API programmatically without going through the web login flow.

## Context

- Web UI admins use email/password → JWT HS256 session (existing, unchanged)
- Backend services need long-lived, revocable credentials for API access
- API keys inherit the creator's role (super/admin/viewer) — no separate scope system
- System is internal-only, low QPS, so DB lookup per request is acceptable

## Key Format

`nak_live_<32 random bytes, base62>` (~52 chars total). Prefix `nak_` lets middleware distinguish API keys from JWTs without parsing.

## Database

New `api_keys` table:

```sql
CREATE TABLE api_keys (
  id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  admin_id     UUID         NOT NULL REFERENCES admin_users(id) ON DELETE CASCADE,
  name         VARCHAR(100) NOT NULL,
  key_prefix   VARCHAR(12)  NOT NULL,
  key_hash     VARCHAR(64)  NOT NULL,
  last_used_at TIMESTAMPTZ,
  created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  revoked_at   TIMESTAMPTZ,
  UNIQUE (admin_id, name)
);
CREATE INDEX api_keys_hash_idx ON api_keys (key_hash) WHERE revoked_at IS NULL;
```

- `key_hash`: SHA-256 hex of the raw key. Raw key never stored.
- `key_prefix`: First 8 chars of raw key (e.g., `nak_live`) for display in the UI list so admins can identify keys.
- Partial index keeps lookup fast and excludes revoked keys.

## Auth Middleware

`RequireAdmin` in `middleware/auth.go` bifurcates based on token prefix:

**API key path** (`Bearer nak_live_...`):
1. `SHA-256(token)` → look up `api_keys` by `key_hash WHERE revoked_at IS NULL`
2. Join to `admin_users` to get role
3. Set `AdminID` + `AdminRole` in context (same keys as JWT path)
4. Update `last_used_at`

**JWT path** (everything else):
- Existing logic unchanged. Parse JWT, validate signature + expiry, extract claims.

Both paths produce identical context values, so no downstream handler changes needed.

## API Endpoints

All under `/api/v1/settings/api-keys`, protected by existing JWT auth:

| Method | Path | Action |
|--------|------|--------|
| POST | `/api/v1/settings/api-keys` | Create key. Body: `{name}`. Returns `{id, name, key, created_at}`. Key shown once. |
| GET | `/api/v1/settings/api-keys` | List keys for current admin. Returns `{id, name, key_prefix, created_at, last_used_at, revoked_at}[]` |
| DELETE | `/api/v1/settings/api-keys/:id` | Revoke key (sets `revoked_at`). Only creator or super can revoke. |

## Key Lifecycle

1. Admin creates key via UI or API → raw key returned once, DB stores hash only
2. Service uses `Authorization: Bearer nak_live_...` on every request
3. Middleware resolves to admin_id + role, request proceeds as normal
4. Admin revokes key → `revoked_at` set → middleware rejects → 401
5. If admin user is deleted → CASCADE deletes all their API keys

## Audit Integration

Existing `audit_logs` works as-is. The `admin_id` is set from the API key's creator. Add `api_key_id` to the `detail` JSONB field so audit entries can trace back to which key was used.

## Frontend

New page at `/settings/api-keys`:
- Table: name, key prefix, created date, last used, status (active/revoked)
- "Create API Key" dialog with name input → shows key once in copyable field with "This key won't be shown again" warning
- Revoke button with confirmation dialog
- Route protected by existing JWT auth

## What Doesn't Change

- Login flow (email/password → JWT)
- JWT expiry and reauth modal
- Existing route protection (all endpoints accept both JWT and API key)
- Role system (roles exist in DB but no per-endpoint RBAC yet — not in scope)
