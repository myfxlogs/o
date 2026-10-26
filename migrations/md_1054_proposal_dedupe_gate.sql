-- 0054 T-0131 事件环自激洪泛根修 md 链（T-0167 分链：提案去重闸=桥表机制
-- 与 fin 链 fin_0054 同形（DDL 归一化断言钉）；洪泛窗死行清史=历史数据修复
-- 不落 PG-md 新库——研究臂建库即走同键在途跳写 dedupe）。
CREATE OR REPLACE FUNCTION fin_proposal_dedupe() RETURNS trigger AS $$
BEGIN
    IF EXISTS (SELECT 1 FROM fin_proposal
        WHERE inst_id=NEW.inst_id AND direction=NEW.direction
          AND state IN ('fin_proposed','fin_armed')) THEN
        RETURN NULL; -- 同键在途提案/武装行——跳写不报错（幂等闸）
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS fin_proposal_dedupe_trg ON fin_proposal;
CREATE TRIGGER fin_proposal_dedupe_trg BEFORE INSERT ON fin_proposal
  FOR EACH ROW EXECUTE FUNCTION fin_proposal_dedupe();
