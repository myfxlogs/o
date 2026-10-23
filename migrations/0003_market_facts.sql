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
SELECT create_hypertable('md_ticks', 'ts', if_not_exists => TRUE);

CREATE TABLE IF NOT EXISTS md_books5 (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    bids     jsonb NOT NULL,
    asks     jsonb NOT NULL,
    seq_id   bigint,
    checksum bigint,
    recv_ts  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_books5.venue IS 'venue 域（裸值禁冒号投影）';
SELECT create_hypertable('md_books5', 'ts', if_not_exists => TRUE);

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
SELECT create_hypertable('md_mark_index', 'ts', if_not_exists => TRUE);
