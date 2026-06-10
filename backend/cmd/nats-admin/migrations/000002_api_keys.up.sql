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
