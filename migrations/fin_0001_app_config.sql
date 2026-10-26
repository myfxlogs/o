-- 0001 底表：业务配置单源（D-0005 业主令——生产 env 只留 DB_DSN，余全量入 DB）。
-- app_config=标量配置；app_secrets=密文（解密钥=主机密钥文件，不入库不入 env）。
-- 代次列 generation + pg_notify config_changed：写方同事务触发，消费方 LISTEN
-- +断线快照重载（严格热启契约）。
CREATE SEQUENCE IF NOT EXISTS config_generation_seq;

CREATE TABLE IF NOT EXISTS app_config (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    generation bigint NOT NULL DEFAULT nextval('config_generation_seq'),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_secrets (
    key        text PRIMARY KEY,
    ciphertext bytea NOT NULL,
    generation bigint NOT NULL DEFAULT nextval('config_generation_seq'),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION notify_config_changed() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('config_changed', TG_TABLE_NAME);
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION bump_config_generation() RETURNS trigger AS $$
BEGIN
    NEW.generation := nextval('config_generation_seq');
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER app_config_gen
    BEFORE INSERT OR UPDATE ON app_config
    FOR EACH ROW EXECUTE FUNCTION bump_config_generation();
CREATE OR REPLACE TRIGGER app_config_notify
    AFTER INSERT OR UPDATE OR DELETE ON app_config
    FOR EACH ROW EXECUTE FUNCTION notify_config_changed();

CREATE OR REPLACE TRIGGER app_secrets_gen
    BEFORE INSERT OR UPDATE ON app_secrets
    FOR EACH ROW EXECUTE FUNCTION bump_config_generation();
CREATE OR REPLACE TRIGGER app_secrets_notify
    AFTER INSERT OR UPDATE OR DELETE ON app_secrets
    FOR EACH ROW EXECUTE FUNCTION notify_config_changed();
