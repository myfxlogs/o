-- 0049 T-0123 md_* 状态表化（D-0018 业主令：追加式快照全废+库无休止膨胀止血）：
--   ① md_ticks/md_mark_index/md_account_events/md_earn_rate/md_dcd_product
--     超表→普通状态表（唯一键 upsert——读面 ORDER BY ts DESC LIMIT 1 语义天然
--     兼容；键粒度逐写径核对：account_events 含 ccy/inst_id，earn_interest 按
--     ccy 读、account/balance_and_position 按 channel 读，dcd_live ccy=''）。
--   ② md_books5 DROP（SG 实拍 1.8GB/344 万行零 SELECT 消费者——订阅同撤）。
--   ③ md_funding 有窗口消费者（fin_forward_check 结算和 ts<=target）→保留追加
--     +30d 保留期；md_equity_curve 回撤窗历史保留+90d 保留期。
--   ④ 旧追加史随 DROP 清空（审计链在 fin_*+venue 账单——票面钦定）。

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

-- 窗口读面坐标（票面举坐：funding 结算和 30d、equity_curve 回撤窗 90d）。
SELECT add_retention_policy('md_funding', interval '30 days');
SELECT add_retention_policy('md_equity_curve', interval '90 days');
