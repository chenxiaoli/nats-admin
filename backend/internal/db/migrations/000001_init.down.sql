DROP TRIGGER IF EXISTS audit_logs_no_update ON audit_logs;
DROP FUNCTION IF EXISTS audit_logs_no_modify;

DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS admin_users;
DROP TABLE IF EXISTS user_credentials;
DROP TABLE IF EXISTS tenant_keys;
DROP TABLE IF EXISTS tenants;
