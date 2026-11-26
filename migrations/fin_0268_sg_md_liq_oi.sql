-- fin_0268 SG md-schema 缺口补齐（工单O-41362）——SG=fin 链域（D-0030§三
-- 钦定），但 SG o-core 内 okxws 采集器写 md_* 表；md_liquidation/
-- md_open_interest 只在 md_1105（PG-md 链）建表→SG 缺表，insertBatch
-- 单事务批插 prepare 败整批丢（liq 行毒掉同批 funding/tick——实证批
-- 30 行 funding28+liq2 全弃，11-23 部署起 3 天间歇失血；外席急诊手灌
-- 同形表止血未入 schema_migrations）。
-- 本件=md_1105 DDL 全等 fin 链双生（与 md_1105 同形对偶——PG-md 侧
-- md_1105 已建同名表时 IF NOT EXISTS 平滑跳过，replay-safe）。
-- 谱系规约（防再犯）：凡 SG 常驻件写 md_* 表，建表件必须走 fin 链——
-- md 链只养 PG-md（md_1103/1104/1108/1110 对偶件与 fin 链已做件撞车，
-- SG 禁跑 -domain=md 整链）。

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
