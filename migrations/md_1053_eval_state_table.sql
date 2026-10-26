-- 0051 T-0126 fin_opp_eval 状态表化 md 链（T-0167 分链：fin_opp_eval UNIQUE 键+
-- ev_eval 触发器=桥表 DDL，与 fin 链 fin_0053 同形（DDL 归一化断言钉）；
-- 数据面收敛（eval_id 重指/非最新行删）=历史数据修复不落 PG-md 新库——
-- 正本建库即状态表形，无追加史可收敛）。
ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_key;
ALTER TABLE fin_opp_eval
    ADD CONSTRAINT fin_opp_eval_key UNIQUE (inst_id, template, direction);

DROP TRIGGER IF EXISTS ev_eval_trg ON fin_opp_eval;
CREATE TRIGGER ev_eval_trg AFTER INSERT OR UPDATE ON fin_opp_eval
  FOR EACH STATEMENT EXECUTE FUNCTION ev_notify('ev_eval');
