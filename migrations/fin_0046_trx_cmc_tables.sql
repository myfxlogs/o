-- 0046 TRX 自营盘表群（T-0108 nox 移植——fin_trx_* 7 表+fin_cmc_* 4 表+
-- position_daily_income carry 源；id 列 bigserial 主键化裁剪〔nox 裸序列+
-- 后置 pkey 合并等效〕；业主凭据键落 app_config 非 env 裸键）。
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

CREATE TABLE IF NOT EXISTS fin_trx_cycle_vi (
    id BIGSERIAL PRIMARY KEY,
    cycle bigint NOT NULL,
    sr_address character varying(64) NOT NULL,
    vi numeric DEFAULT 0 NOT NULL,
    user_votes bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

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

CREATE TABLE IF NOT EXISTS fin_trx_vote_apr_top (
    id BIGSERIAL PRIMARY KEY,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL,
    sr_name character varying(64) NOT NULL,
    annualized_rate double precision NOT NULL
);

CREATE TABLE IF NOT EXISTS fin_trx_vote_rebalance (
    id integer NOT NULL,
    decided_at timestamp with time zone DEFAULT now() NOT NULL,
    current_yield double precision NOT NULL,
    target_yield double precision NOT NULL,
    improvement_pct double precision NOT NULL,
    need_rebalance boolean NOT NULL,
    reason text NOT NULL,
    target_votes jsonb NOT NULL,
    executed_at timestamp with time zone,
    tx_hash text,
    status text DEFAULT 'pending'::text NOT NULL,
    fail_reason text
);

CREATE TABLE IF NOT EXISTS fin_trx_witness (
    id BIGSERIAL PRIMARY KEY,
    sr_address character varying(64) NOT NULL,
    url character varying(256),
    vote_count bigint,
    fetched_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE IF NOT EXISTS position_daily_income (
    id BIGSERIAL PRIMARY KEY,
    day date NOT NULL,
    energy_trx numeric DEFAULT 0 NOT NULL,
    voting_trx numeric DEFAULT 0 NOT NULL,
    interest_usd numeric DEFAULT 0 NOT NULL,
    price numeric,
    pnl_usd numeric,
    note text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    source character varying(8) DEFAULT 'manual'::character varying NOT NULL
);
-- 1047_cmc_tables.sql —— CMC Markets 采集器表族回归（STATE「CMC 采集器移植」票；
-- M3 切流时 fin_cmc_* 随 D-225 族件剔除不搬（1003 DROP），TRX 页三区业主令 10-19
-- 复工——schema 照 arb-archive 迁移源件 0065+0067+0069 终局形态合并重建，禁删史无涉
-- （NOX 库从未有此族表，纯新建）。
-- 装配键随迁（D-162：装配点 DB 驱动禁 env）——cmc_account_id=业主 CMC 账户号
-- （arb-archive wire_cmc.go 硬编码 38941616 的键化）；cmc_ws_disabled=false=WS 事件
-- 驱动默认开（true=回退全 REST，arb D-225 v2.1 同语义）。

CREATE TABLE IF NOT EXISTS fin_cmc_prices (
    id BIGSERIAL PRIMARY KEY,
    instrument_id VARCHAR(32) NOT NULL,
    bid_price NUMERIC(18,8),
    ask_price NUMERIC(18,8),
    mid_price NUMERIC(18,8),
    market_status VARCHAR(16),
    quote_time TIMESTAMPTZ,
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    src TEXT NOT NULL DEFAULT 'cmc_rest'
);
CREATE INDEX IF NOT EXISTS idx_cmc_prices_time ON fin_cmc_prices(fetched_at);

CREATE TABLE IF NOT EXISTS fin_cmc_account_summary (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL,
    total_account_value NUMERIC(18,2),
    total_cash_balance NUMERIC(18,2),
    total_unrealised_pnl NUMERIC(18,2),
    total_equity_free_for_margin NUMERIC(18,2),
    total_equity_used_in_margin NUMERIC(18,2),
    closeout_price NUMERIC(18,8),
    distance_to_closeout_pct NUMERIC(8,4),
    fetched_at TIMESTAMPTZ DEFAULT NOW(),
    alerts JSONB NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_cmc_summary_time ON fin_cmc_account_summary(fetched_at);

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


-- 装配键（D-162 DB 驱动——cmc_account_id=业主 CMC 账户号；cmc_ws_disabled
-- 缺省 false；TRX 质押地址=业主条件请求键，空值=采集臂让位不误报）。
INSERT INTO app_config(key,value) VALUES
  ('cmc.account_id','""'),
  ('cmc.ws_disabled','false'),
  ('trx.stake_address','""'),
  ('trx.poll_seconds','300')
ON CONFLICT (key) DO NOTHING;
