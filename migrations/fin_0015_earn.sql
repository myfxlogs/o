-- 0015 Earn 停泊态 fin 链（T-0167 分链）：cashflow kind 闭集扩+risk reason 闭集扩
-- +earn 配置键=钱径面归 PG-fin；md_earn_rate 表走 md 链 md_0015。

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
