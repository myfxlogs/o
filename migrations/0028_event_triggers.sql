-- 0025 事件脊触发器（T-0048：周期环全拆——行落库→pg_notify 真事件）。
-- 同事务内同 channel+payload 的 NOTIFY 天然去重（批量 flush 一拍一醒）；
-- payload 恒空串=纯唤醒语义（判定数据一律现读，事件不携值防陈拍传染）。
CREATE OR REPLACE FUNCTION ev_notify() RETURNS trigger AS $$
BEGIN PERFORM pg_notify(TG_ARGV[0], ''); RETURN NULL; END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER ev_ticks_trg AFTER INSERT ON md_ticks
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_ticks');
CREATE OR REPLACE TRIGGER ev_funding_trg AFTER INSERT ON md_funding
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_funding');
CREATE OR REPLACE TRIGGER ev_position_trg AFTER INSERT OR UPDATE ON fin_position
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_position');
CREATE OR REPLACE TRIGGER ev_cashflow_trg AFTER INSERT ON fin_cashflow
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_cashflow');
CREATE OR REPLACE TRIGGER ev_strategy_trg AFTER INSERT OR UPDATE ON fin_strategy
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_strategy');
CREATE OR REPLACE TRIGGER ev_eval_trg AFTER INSERT ON fin_opp_eval
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_eval');
CREATE OR REPLACE TRIGGER ev_account_trg AFTER INSERT ON md_account_events
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_account');
-- review 判决到期域（fin_forward_check 行变=新排期事件）。
CREATE OR REPLACE TRIGGER ev_fwdchk_trg AFTER INSERT OR UPDATE ON fin_forward_check
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_fwdchk');
