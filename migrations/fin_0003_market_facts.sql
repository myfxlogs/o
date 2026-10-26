-- T-0167 分链：本件在 fin 链=八桥表 SG 镜像落地面（md_ticks/md_funding/md_mark_index
-- 及已废 md_books5 历史），DDL 与 md 链 md_0003 同形（check_migrations 桥表断言钉）。
-- 0003 行情事实表（TimescaleDB hypertable 口径——ts=分区键，事件流行=唯一权威态源）。
-- venue 键域列：COMMENT 归属 + CHECK 裸域禁冒号（D-496 迁移契约）。
CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS md_ticks (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    last     numeric,
    bid      numeric,
    ask      numeric,
    vol_24h  numeric,
    recv_ts  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_ticks.venue IS 'venue 域（裸值禁冒号投影）';
DO $$ BEGIN IF to_regclass('md_ticks') IS NULL THEN PERFORM create_hypertable('md_ticks', 'ts', if_not_exists => TRUE); END IF; END $$;

CREATE TABLE IF NOT EXISTS md_funding (
    ts              timestamptz NOT NULL,
    venue           text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id         text NOT NULL,
    funding_rate    numeric,
    next_funding_ts timestamptz,
    recv_ts         timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_funding.venue IS 'venue 域（裸值禁冒号投影）';
SELECT create_hypertable('md_funding', 'ts', if_not_exists => TRUE);

CREATE TABLE IF NOT EXISTS md_mark_index (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    channel  text NOT NULL,
    px       numeric,
    recv_ts  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_mark_index.venue IS 'venue 域（裸值禁冒号投影）';
DO $$ BEGIN IF to_regclass('md_mark_index') IS NULL THEN PERFORM create_hypertable('md_mark_index', 'ts', if_not_exists => TRUE); END IF; END $$;
