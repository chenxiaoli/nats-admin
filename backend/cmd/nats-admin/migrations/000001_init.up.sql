-- 001_init.sql
-- Creates the five core tables for the NATS admin backend.
-- Money/byte columns use BIGINT; -1 means "no limit" (NATS JWT semantics).
-- audit_logs intentionally has no FKs: admin_id / tenant_id may be soft-deleted.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------- tenants ----------
CREATE TABLE tenants (
  id                    UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  name                  VARCHAR(100) UNIQUE NOT NULL,
  slug                  VARCHAR(50)  UNIQUE NOT NULL,
  account_public_key    VARCHAR(60)  NOT NULL,
  account_jwt           TEXT         NOT NULL,
  js_max_memory_storage BIGINT       NOT NULL DEFAULT -1,
  js_max_disk_storage   BIGINT       NOT NULL DEFAULT -1,
  js_max_streams        INT          NOT NULL DEFAULT -1,
  js_max_consumers      INT          NOT NULL DEFAULT -1,
  max_connections       INT          NOT NULL DEFAULT -1,
  max_subscriptions     INT          NOT NULL DEFAULT -1,
  status                VARCHAR(20)  NOT NULL DEFAULT 'active'
                         CHECK (status IN ('active', 'suspended', 'deleted')),
  created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX tenants_status_idx     ON tenants (status);
CREATE INDEX tenants_account_pub_idx ON tenants (account_public_key);

-- ---------- tenant_keys (encrypted Account seed) ----------
CREATE TABLE tenant_keys (
  tenant_id       UUID   PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  encrypted_seed  BYTEA  NOT NULL,
  nonce           BYTEA  NOT NULL,
  key_version     INT    NOT NULL DEFAULT 1
);

-- ---------- user_credentials (User JWT + encrypted seed) ----------
CREATE TABLE user_credentials (
  id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id        UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  name             VARCHAR(100) NOT NULL,
  user_public_key  VARCHAR(60)  NOT NULL,
  user_jwt         TEXT         NOT NULL,
  encrypted_seed   BYTEA        NOT NULL,
  nonce            BYTEA        NOT NULL,
  pub_allow        TEXT[],
  sub_allow        TEXT[],
  revoked_at       TIMESTAMPTZ,
  expires_at       TIMESTAMPTZ,
  created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, name)
);
CREATE INDEX user_credentials_tenant_idx    ON user_credentials (tenant_id);
CREATE INDEX user_credentials_pubkey_idx    ON user_credentials (user_public_key);
CREATE INDEX user_credentials_revoked_idx   ON user_credentials (tenant_id) WHERE revoked_at IS NOT NULL;

-- ---------- admin_users (web console) ----------
CREATE TABLE admin_users (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  email         VARCHAR(200) UNIQUE NOT NULL,
  password_hash VARCHAR(100) NOT NULL,
  role          VARCHAR(20)  NOT NULL DEFAULT 'admin'
                 CHECK (role IN ('super', 'admin', 'viewer')),
  created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ---------- audit_logs (append-only) ----------
CREATE TABLE audit_logs (
  id          BIGSERIAL    PRIMARY KEY,
  admin_id    UUID,
  tenant_id   UUID,
  action      VARCHAR(100) NOT NULL,
  resource    VARCHAR(100),
  resource_id TEXT,
  detail      JSONB,
  ip_addr     INET,
  created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX audit_logs_tenant_time_idx ON audit_logs (tenant_id, created_at DESC);
CREATE INDEX audit_logs_time_idx        ON audit_logs (created_at DESC);

-- Enforce append-only: trigger blocks UPDATE/DELETE.
CREATE OR REPLACE FUNCTION audit_logs_no_modify() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'audit_logs is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update
BEFORE UPDATE OR DELETE ON audit_logs
FOR EACH ROW EXECUTE FUNCTION audit_logs_no_modify();
