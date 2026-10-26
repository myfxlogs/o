-- 0005 私有面事实+快照行（okx-private-ws 契约：事件落事实行+快照行活性心跳刷源）。
CREATE TABLE IF NOT EXISTS md_account_events (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    channel  text NOT NULL,
    ccy      text,
    inst_id  text,
    data     jsonb NOT NULL,
    recv_ts  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_account_events.venue IS 'venue 域（裸值禁冒号投影）';
SELECT create_hypertable('md_account_events', 'ts', if_not_exists => TRUE);

-- 快照行（unified 归一键域：armed 腿键=okx:perp/spot 但账户行只落 unified）；
-- 活性判据=行新鲜度非事件到达率（静户零事件≠WS 死——3min 心跳刷源兜底）。
CREATE TABLE IF NOT EXISTS fin_venue_account_snapshot (
    venue       text NOT NULL CHECK (venue NOT LIKE '%:%'),
    venue_type  text NOT NULL CHECK (venue_type IN ('unified','spot','perp','funding','earn','futures','mt4','mt5')),
    ccy         text NOT NULL,
    equity      numeric,
    avail       numeric,
    src         text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, venue_type, ccy)
);
COMMENT ON COLUMN fin_venue_account_snapshot.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN fin_venue_account_snapshot.venue_type IS 'venue_type 枚举闭集';
