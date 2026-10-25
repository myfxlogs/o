-- 0054 T-0131 事件环自激洪泛根修（生产实证 5k 行/分灌穿提案表）：
--   ① 提案去重触发器入册（决策席临时止血闸迁移固化——同 inst_id+direction
--     存在 fin_proposed/fin_armed 即跳写，幂等形可重放）；
--   ② 洪泛窗死行清史——state=fin_retired 同键只留最新一行（真审计链保尾，
--     机械判定=非最新退役行即洪泛副产物）。
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

-- 洪泛窗清史：fin_retired 同键留最新一行（审计链尾锚），余删。
DELETE FROM fin_proposal p
WHERE p.state='fin_retired' AND EXISTS (
    SELECT 1 FROM fin_proposal k
    WHERE k.inst_id=p.inst_id AND k.direction=p.direction
      AND k.state='fin_retired' AND k.id > p.id);
