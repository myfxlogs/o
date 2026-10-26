-- 0028 事件脊触发器 fin 链（T-0048：周期环全拆——行落库→pg_notify 真事件）。
-- T-0167 分链：钱径域触发器（ev_position/ev_cashflow/ev_strategy/ev_account）归
-- PG-fin；桥表触发器（ev_ticks/ev_funding/ev_eval）在 md 链 md_0028 同形在册。
-- 同事务内同 channel+payload 的 NOTIFY 天然去重（批量 flush 一拍一醒）；
-- payload 恒空串=纯唤醒语义（判定数据一律现读，事件不携值防陈拍传染）。
CREATE OR REPLACE FUNCTION ev_notify() RETURNS trigger AS $$
BEGIN PERFORM pg_notify(TG_ARGV[0], ''); RETURN NULL; END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER ev_position_trg AFTER INSERT OR UPDATE ON fin_position
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_position');
CREATE OR REPLACE TRIGGER ev_cashflow_trg AFTER INSERT ON fin_cashflow
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_cashflow');
CREATE OR REPLACE TRIGGER ev_strategy_trg AFTER INSERT OR UPDATE ON fin_strategy
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_strategy');
CREATE OR REPLACE TRIGGER ev_account_trg AFTER INSERT ON md_account_events
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_account');
