# Database Schema

## tenants（Account 元数据）

```sql
CREATE TABLE tenants (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name                  VARCHAR(100) UNIQUE NOT NULL,
  slug                  VARCHAR(50)  UNIQUE NOT NULL,
  account_public_key    VARCHAR(60)  NOT NULL,
  account_jwt           TEXT         NOT NULL,   -- 当前有效 JWT（明文，随 revocation 更新）
  js_max_memory_storage BIGINT       DEFAULT -1,
  js_max_disk_storage   BIGINT       DEFAULT -1,
  js_max_streams        INT          DEFAULT -1,
  js_max_consumers      INT          DEFAULT -1,
  max_connections       INT          DEFAULT -1,
  max_subscriptions     INT          DEFAULT -1,
  status                VARCHAR(20)  DEFAULT 'active',  -- active | suspended | deleted
  created_at            TIMESTAMPTZ  DEFAULT NOW(),
  updated_at            TIMESTAMPTZ  DEFAULT NOW()
);
```

## tenant_keys（Account NKey seed，加密存储）

```sql
-- Account seed 用于签发 User JWT；不存 Operator seed（只在 env）
CREATE TABLE tenant_keys (
  tenant_id       UUID   PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
  encrypted_seed  BYTEA  NOT NULL,
  nonce           BYTEA  NOT NULL,
  key_version     INT    DEFAULT 1
);
```

## user_credentials（User JWT + 加密 seed）

```sql
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
  created_at       TIMESTAMPTZ  DEFAULT NOW(),
  UNIQUE(tenant_id, name)
);
CREATE INDEX ON user_credentials(tenant_id);
```

## admin_users（管理后台用户）

```sql
CREATE TABLE admin_users (
  id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
  email         VARCHAR(200) UNIQUE NOT NULL,
  password_hash VARCHAR(100) NOT NULL,   -- bcrypt cost=12
  role          VARCHAR(20)  DEFAULT 'admin',  -- super | admin | viewer
  created_at    TIMESTAMPTZ  DEFAULT NOW()
);
```

## audit_logs（append-only，不允许 UPDATE/DELETE）

```sql
CREATE TABLE audit_logs (
  id          BIGSERIAL    PRIMARY KEY,
  admin_id    UUID,
  tenant_id   UUID,
  action      VARCHAR(100) NOT NULL,  -- tenant.create | credential.revoke | ...
  resource    VARCHAR(100),
  resource_id TEXT,
  detail      JSONB,
  ip_addr     INET,
  created_at  TIMESTAMPTZ  DEFAULT NOW()
);
CREATE INDEX ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX ON audit_logs(created_at DESC);
```

## 设计约定

- 所有 `id` 列用 UUID，不用自增整数
- 金额/容量用 BIGINT（字节数），-1 表示不限制（与 NATS JWT 语义对齐）
- `account_jwt` 列随每次 revocation/limits 变更同步更新
- `encrypted_seed` + `nonce` 永远成对存取，AES-256-GCM 解密
- audit_logs 不设外键约束（`admin_id`/`tenant_id` 可能已被软删除）
