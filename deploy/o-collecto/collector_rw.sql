-- collector_rw 桥写权收窄角色（T-0137 安全边界——隧道泄=损失限定行情面；
-- 钱径/账表零写权）。SG psql 一次性执行。
CREATE ROLE collector_rw LOGIN PASSWORD 'CHANGE_ME';
GRANT INSERT,UPDATE,SELECT ON ALL TABLES IN SCHEMA public
  TO collector_rw; -- 收窄：仅 md_* 表面（下列精化）
REVOKE ALL ON fin_cashflow,fin_position,fin_order_leg,fin_execution,
  fin_strategy,fin_strategy_transition,fin_proposal,fin_opp_eval,
  fin_venue_bill,fin_push_ledger,app_config,app_secrets
  FROM collector_rw;
