-- 1087 T-0326 返修① stakeCarryScan 质押腿断供根修——五 fin 表升
-- bridge-reverse 副本入 PG-md（升件1593 补裁采纳修法①，md_1070 先例同形）：
--   病灶=hypothesis 臂持 md 池读 fin 域表三连 to_regclass NULL→SpotYield/
--   shortCarryAnn 42P01→「该对不评如实」静默（concrete cmc:TRX 零产=结构性
--   断供）；rental 腿 fin_trx_account/fin_trx_txs err==nil 吞 42P01→rental 恒
--   0 退化（spotyield.go 同病灶一并根修）。五表体量全小（vote_apr 2409/
--   trx_account 3008/trx_txs 813/cmc_history 39/cmc_positions 8390 行）——
--   全表 diff 廉价，勿造 CDC/触发器新范式；无 SG 触发帧者随 Reconcile 周期
--   对账。DDL 与 fin 链正表归一化同形（check_migrations 断言⑥钉）。
-- SG 正本→projector 反向镜像（全表 diff 同源）；研究面只读，写权唯一=
-- projector。触发器/回填 UPDATE 不在副本。

CREATE TABLE IF NOT EXISTS fin_trx_account (
    id BIGSERIAL PRIMARY KEY,
    address character varying(64) NOT NULL,
    votes_total bigint NOT NULL,
    sr_address character varying(64),
    balance_sun bigint NOT NULL,
    allowance_sun bigint NOT NULL,
    energy_delegated bigint NOT NULL,
    bandwidth_staked bigint NOT NULL,
    net_limit bigint NOT NULL,
    unfrozen jsonb,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    tron_power bigint DEFAULT 0 NOT NULL,
    net_used bigint DEFAULT 0 NOT NULL,
    free_net_used bigint DEFAULT 0 NOT NULL,
    free_net_limit bigint DEFAULT 0 NOT NULL,
    energy_limit bigint DEFAULT 0 NOT NULL,
    energy_used bigint DEFAULT 0 NOT NULL,
    reward_num_sun bigint DEFAULT 0 NOT NULL
);
COMMENT ON TABLE fin_trx_account IS
  'fin_trx_account PG-md 只读副本（T-0326 反向桥）——SG 正本 projector 全表 diff 维护，质押腿 rental 分母读面';

CREATE TABLE IF NOT EXISTS fin_trx_txs (
    id BIGSERIAL PRIMARY KEY,
    tx_id character varying(64) NOT NULL,
    ts timestamp with time zone NOT NULL,
    kind character varying(32) NOT NULL,
    amount_sun bigint NOT NULL,
    counterparty character varying(64),
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    cheat_status boolean DEFAULT false NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS fin_trx_txs_tx_id_uidx ON fin_trx_txs (tx_id);
COMMENT ON TABLE fin_trx_txs IS
  'fin_trx_txs PG-md 只读副本（T-0326 反向桥）——SG 正本 projector 全表 diff 维护，rental_income 流水读面';

CREATE TABLE IF NOT EXISTS fin_trx_vote_apr (
    id BIGSERIAL PRIMARY KEY,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    current_rate double precision DEFAULT 0 NOT NULL,
    current_sr character varying(64),
    max_rate double precision DEFAULT 0 NOT NULL,
    max_sr character varying(64),
    spread double precision DEFAULT 0 NOT NULL,
    min_rate double precision DEFAULT 0 NOT NULL,
    min_sr character varying(64)
);
COMMENT ON TABLE fin_trx_vote_apr IS
  'fin_trx_vote_apr PG-md 只读副本（T-0326 反向桥）——SG 正本 projector 全表 diff 维护，stakeCarryScan SpotYield vote 年化读面';

CREATE TABLE IF NOT EXISTS fin_cmc_positions (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL,
    instrument_id VARCHAR(32),
    direction VARCHAR(8),
    quantity NUMERIC(18,2),
    entry_price NUMERIC(18,8),
    unrealised_pnl NUMERIC(18,2),
    margin_amount NUMERIC(18,2),
    margin_percentage NUMERIC(8,4),
    evaluation_bid_price NUMERIC(18,8),
    evaluation_ask_price NUMERIC(18,8),
    opened_at TIMESTAMPTZ,
    order_id VARCHAR(64),
    fetched_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cmc_positions_time ON fin_cmc_positions(fetched_at);
COMMENT ON TABLE fin_cmc_positions IS
  'fin_cmc_positions PG-md 只读副本（T-0326 反向桥）——SG 正本 projector 全表 diff 维护，shortCarryAnn 空腿名义读面';

CREATE TABLE IF NOT EXISTS fin_cmc_history (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL,
    transaction_id VARCHAR(64) UNIQUE,
    transaction_type VARCHAR(32),
    amount NUMERIC(18,2),
    balance NUMERIC(18,2),
    currency VARCHAR(8),
    order_id VARCHAR(32),
    transaction_time TIMESTAMPTZ,
    fetched_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cmc_history_time ON fin_cmc_history(transaction_time);
CREATE INDEX IF NOT EXISTS idx_cmc_history_txid ON fin_cmc_history(transaction_id);
COMMENT ON TABLE fin_cmc_history IS
  'fin_cmc_history PG-md 只读副本（T-0326 反向桥）——SG 正本 projector 全表 diff 维护，cmc_realized 隔夜结算读面';

-- 保留账入册（清道夫-md fail-closed 未入账报缺——副本=镜像不删行，同
-- md_1070 口径：sweeper 删行与全表 diff 互搏=churn 环，故 class=state）。
INSERT INTO fin_data_retention(table_name,class,keep_interval,note) VALUES
  ('fin_trx_account','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0326）'),
  ('fin_trx_txs','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0326）'),
  ('fin_trx_vote_apr','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0326）'),
  ('fin_cmc_positions','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0326）'),
  ('fin_cmc_history','state',NULL,'SG 正本反向镜像只读副本——projector diff 维护（T-0326）')
ON CONFLICT (table_name) DO NOTHING;
