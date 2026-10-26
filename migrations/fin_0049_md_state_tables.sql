-- 0049 T-0123 md_* 状态表化 fin 链（T-0167 分链）：桥表状态化 DDL 与 md 链
-- md_0049 同形（DDL 归一化断言钉——SG 镜像表与 PG-md 正表回滚同形前提）；
-- md_account_events（账户事件=钱径事件非行情，D-0030 §五归 fin）状态化+
-- md_equity_curve 回撤窗 90d 保留期=fin 域本链面。

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

DROP TABLE md_account_events;
CREATE TABLE md_account_events (
    ts       timestamptz NOT NULL,
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    channel  text NOT NULL,
    ccy      text NOT NULL DEFAULT '',
    inst_id  text NOT NULL DEFAULT '',
    data     jsonb NOT NULL,
    recv_ts  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (venue, channel, ccy, inst_id)
);
COMMENT ON COLUMN md_account_events.venue IS 'venue 域（裸值禁冒号投影）';

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

-- 窗口读面坐标（票面举坐：funding 结算和 30d、equity_curve 回撤窗 90d）。
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM timescaledb_information.jobs WHERE hypertable_name='md_funding' AND proc_name='policy_retention') THEN PERFORM add_retention_policy('md_funding', interval '30 days'); END IF; END $$;
SELECT add_retention_policy('md_equity_curve', interval '90 days');
