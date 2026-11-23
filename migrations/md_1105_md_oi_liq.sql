-- 1105 md_open_interest + md_liquidation（工单O-35587②/#35490 数据面
-- gap 两件——普查发现：OKX OI ctx 臂采集在案但 md 域无表、清算流全缺）：
-- ①md_open_interest=OI 状态表（venue,inst_id 最新态 upsert 同
--   md_ticks 律——300s 快照拍慢变量；ctx md_ctx_fact 日戳 JSON 桶
--   非关系面不动）；②md_liquidation=清算事件流水（OKX 公共 WS
--   liquidation-orders 频道——docs-v5 实证载荷；超表+90d 保留期
--   同 md_funding 先例；天然键去重 DO NOTHING——重连重推/多实例
--   同订幂等）。两表入 fin_data_retention 保留账（清道夫 fail-closed
--   未入账拒动）。

CREATE TABLE IF NOT EXISTS md_open_interest (
    ts       timestamptz NOT NULL DEFAULT now(),
    venue    text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id  text NOT NULL,
    oi       double precision,
    oi_ccy   double precision,
    recv_ts  timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_open_interest.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON TABLE md_open_interest IS 'OKX SWAP OI 最新态（OIPoller 300s 拍 upsert——state 表不删行）';
-- mig-rehearsal-ok: md_open_interest 本迁移内 CREATE——空表建唯一索引零存量违例可能（state upsert 键）
CREATE UNIQUE INDEX IF NOT EXISTS md_open_interest_venue_inst_uq
    ON md_open_interest (venue, inst_id);

CREATE TABLE IF NOT EXISTS md_liquidation (
    ts          timestamptz NOT NULL,
    venue       text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    inst_id     text NOT NULL,
    inst_family text,
    uly         text,
    pos_side    text,
    side        text,
    sz          double precision,
    bk_px       double precision,
    bk_loss     double precision,
    ccy         text,
    recv_ts     timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_liquidation.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON TABLE md_liquidation IS 'OKX SWAP 清算事件流水（liquidation-orders WS——天然键去重）';
SELECT create_hypertable('md_liquidation', 'ts', if_not_exists => TRUE);
-- mig-rehearsal-ok: md_liquidation 本迁移内 CREATE——空表建唯一索引零存量违例可能（天然键去重设计目的）
CREATE UNIQUE INDEX IF NOT EXISTS md_liquidation_dedup_uq
    ON md_liquidation (venue, inst_id, ts, pos_side, side, sz, bk_px);
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM timescaledb_information.jobs WHERE hypertable_name='md_liquidation' AND proc_name='policy_retention') THEN PERFORM add_retention_policy('md_liquidation', interval '90 days'); END IF; END $$;

INSERT INTO fin_data_retention(table_name,class,keep_interval,note) VALUES
  ('md_open_interest','state',NULL,'状态表不删行——upsert 键域封顶（~500 inst）'),
  ('md_liquidation','state',interval '90 days','事件流水超表保留期管删（timescale policy 同 md_funding 口径）')
ON CONFLICT (table_name) DO NOTHING;
