-- 0064 T-0162 ev_* 通知洪泛源治——评估写拍逐行 NOTIFY 塌陷（供方半根修，
-- 消费侧读饥饿已由 T-0154 修）。病灶生产亲证：fin_opp_eval 全键每拍 autocommit
-- upsert（INSERT..ON CONFLICT DO UPDATE SET ts=now() 无条件覆写）→每行独立
-- 事务各发一 NOTIFY（~500 行/min，同事务去重永不生效），而两拍间值变率 0.5%
-- （600 行采样仅 3 行 apr/evidence 动，verdict 变=0）——99.5% 通知是纯噪声，
-- lifecycle 一醒一拍重扫同值=空转。治法=仅收通知面（评估写拍本体不动）：
-- 实质变才发——INSERT=新键恒实质变；UPDATE（含 upsert DO UPDATE 臂，PG18
-- 实证 row-level 触发器该臂 TG_OP='UPDATE' 且 OLD/NEW 可比）verdict/评分/
-- legs/evidence 任一 IS DISTINCT 才 pg_notify（ts 是写拍元数据不计；DO
-- NOTHING 臂零行零fire）。同事务空 payload 同值去重=批内合流天然在位
-- （0028 律不变）；生产写拍为逐行 autocommit，静默面即降率主力。
-- 形状律实证（PG 18.6，t0162 探针）：statement 级 transition table 触发器
-- 禁多事件（"transition tables cannot be specified for triggers with more
-- than one event"）——INSERT OR UPDATE 合一只能 row-level，故本触发器
-- row-level（逐行判定）+同事务去重（批合流）组合达成同效。
-- ev_strategy 同查结论（票面施工项②）：写方全是单 statement 批量 UPDATE
-- （0028 statement 级触发器天然批合流，一拍一 NOTIFY）+状态迁移皆实质变
-- （机迁/退役谓词只命中真变行）——无需动，ev_notify 原样保留。
CREATE OR REPLACE FUNCTION ev_notify_eval() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'INSERT'
     OR NEW.verdict IS DISTINCT FROM OLD.verdict
     OR NEW.gross_apr IS DISTINCT FROM OLD.gross_apr
     OR NEW.net_apr IS DISTINCT FROM OLD.net_apr
     OR NEW.legs IS DISTINCT FROM OLD.legs
     OR NEW.evidence IS DISTINCT FROM OLD.evidence THEN
    PERFORM pg_notify(TG_ARGV[0], '');
  END IF;
  RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ev_eval_trg ON fin_opp_eval;
CREATE TRIGGER ev_eval_trg AFTER INSERT OR UPDATE ON fin_opp_eval
  FOR EACH ROW EXECUTE FUNCTION ev_notify_eval('ev_eval');
