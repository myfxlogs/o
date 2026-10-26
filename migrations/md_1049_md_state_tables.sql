-- 0049 T-0123 md_* 状态表化 md 链（D-0018 业主令：追加式快照全废+库无休止膨胀止血）：
--   T-0167 分链：桥表状态化 DDL（md_ticks/md_mark_index/md_dcd_product/md_funding
--   保留期）与 fin 链 fin_0049 同形（DDL 归一化断言钉）；md_earn_rate=PG-md 本域；
--   md_books5 DROP=md 历史清面；md_account_events 状态化+md_equity_curve 保留期
--   =fin 域走 fin_0049。
--   桥表超表→普通状态表（唯一键 upsert——读面 ORDER BY ts DESC LIMIT 1 语义天然
--   兼容）；旧追加史随 DROP 清空（审计链在 fin_*+venue 账单——票面钦定）。

DROP TABLE IF EXISTS md_books5;

DROP TABLE md_ticks;
CREATE TABLE md_ticks (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    last     numeric,
    bid      numeric,
    ask      numeric,
    vol_24h  numeric,
    recv_ts  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, inst_id)
);
COMMENT ON COLUMN md_ticks.venue IS 'venue 域（裸值禁冒号投影）';

DROP TABLE md_mark_index;
CREATE TABLE md_mark_index (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    channel  text NOT NULL,
    px       numeric,
    recv_ts  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, inst_id, channel)
);
COMMENT ON COLUMN md_mark_index.venue IS 'venue 域（裸值禁冒号投影）';

DROP TABLE md_earn_rate;
CREATE TABLE md_earn_rate (
    ts    timestamptz NOT NULL DEFAULT now(),
    venue text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy   text NOT NULL,
    rate  numeric NOT NULL,
    PRIMARY KEY (venue, ccy)
);
COMMENT ON COLUMN md_earn_rate.venue IS 'venue 域（裸值禁冒号投影）';

DROP TABLE md_dcd_product;
CREATE TABLE md_dcd_product (
    ts               timestamptz NOT NULL,
    venue            text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface          text NOT NULL,
    uly              text NOT NULL,
    product_id       text NOT NULL,
    direction        text NOT NULL CHECK (direction IN ('C','P')),
    strike           numeric,
    expiry           timestamptz,
    annualized_yield numeric,
    min_size         numeric,
    max_size         numeric,
    spot             numeric,
    raw              jsonb NOT NULL DEFAULT '{}',
    PRIMARY KEY (venue, product_id)
);
CREATE INDEX md_dcd_product_uly ON md_dcd_product(uly);

-- 窗口读面坐标（票面举坐：funding 结算和 30d——md_equity_curve 保留期在 fin_0049）。
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM timescaledb_information.jobs WHERE hypertable_name='md_funding' AND proc_name='policy_retention') THEN PERFORM add_retention_policy('md_funding', interval '30 days'); END IF; END $$;
