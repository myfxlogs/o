-- 0028 事件脊触发器 md 链（T-0048：周期环全拆——行落库→pg_notify 真事件）。
-- T-0167 分链：桥表触发器（ev_ticks/ev_funding/ev_eval）双链同形（T-0168 projector
-- LISTEN PG-md 帧=投影 push 源；SG 侧历史同在），md 域 ev_fwdchk 本链独有；
-- fin 域触发器走 fin 链 fin_0028。
-- 同事务内同 channel+payload 的 NOTIFY 天然去重（批量 flush 一拍一醒）；
-- payload 恒空串=纯唤醒语义（判定数据一律现读，事件不携值防陈拍传染）。
CREATE OR REPLACE FUNCTION ev_notify() RETURNS trigger AS $$
BEGIN PERFORM pg_notify(TG_ARGV[0], ''); RETURN NULL; END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER ev_ticks_trg AFTER INSERT ON md_ticks
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_ticks');
CREATE OR REPLACE TRIGGER ev_funding_trg AFTER INSERT ON md_funding
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_funding');
CREATE OR REPLACE TRIGGER ev_eval_trg AFTER INSERT ON fin_opp_eval
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_eval');
-- review 判决到期域（fin_forward_check 行变=新排期事件）。
CREATE OR REPLACE TRIGGER ev_fwdchk_trg AFTER INSERT OR UPDATE ON fin_forward_check
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_fwdchk');
