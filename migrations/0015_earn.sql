-- 0015 Earn 停泊态（D-0006 业主令——金钱不眠）：现率事实行+cashflow/reason 闭集扩+earn 配置键。
-- md_earn_rate=事实行族（append-only，非态表）：公开面 lending-rate-summary 采样行，
-- 读径带新鲜度门（research.earn.stale_hours）；采不到=不写行（禁假值）。
CREATE TABLE IF NOT EXISTS md_earn_rate (
    ts      timestamptz NOT NULL DEFAULT now(),
    venue   text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy     text NOT NULL,
    rate    numeric NOT NULL,
    recv_ts timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_earn_rate.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN md_earn_rate.rate IS '年化 APR 小数（0.035=3.5%）——源=GET /finance/savings/lending-rate-summary avgRate（24h 平均年化出借率）';
SELECT create_hypertable('md_earn_rate', 'ts', if_not_exists => TRUE);

-- cashflow kind 闭集扩：earn_out=闲资停泊出交易户（amount 负）；earn_in=赎回回拢入交易户（amount 正）。
-- 金额符号=对交易户账本净额的边际（停泊款不入交易户净额，与 unified 快照对账自洽）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check CHECK (kind IN
  ('fee','funding_settle','trade_pnl','rollback_cost','deposit','withdraw','earn_in','earn_out'));

-- risk reason 闭集扩 'transfer'：内部资金移动留痕（不走红控敞口判——不增敞口，钱动必有记录）。
ALTER TABLE fin_risk_decision DROP CONSTRAINT fin_risk_decision_reason_check;
ALTER TABLE fin_risk_decision ADD CONSTRAINT fin_risk_decision_reason_check CHECK (reason IN
  ('allow','kill_switch','missing_param','intent_incomplete','notional_cap','venue_concentration',
   'exposure_cap','token_missing','token_expired','token_used','stale_state','transfer'));

INSERT INTO app_config(key,value) VALUES
  ('research.earn.stale_hours', '"2"'),
  ('earn.sweep_min_usd',        '"100"'),
  ('earn.redeem_buffer_bps',    '"50"'),
  ('data.earn.poll_sec',        '"60"')
ON CONFLICT (key) DO NOTHING;
