-- 0061 T-0147 终局分类补面——fin_venue_untradeable 入册钩回扫存量（trigger）：
-- venue 第一次拒=学费一次且当场清尾——同 inst live/paper/proposed 策略→
-- retired（transition 行 reason=venue_untradeable_retroactive 溯源）+armed
-- 提案→fin_retired。幂等二重：inst_id PK（重复入册走 UPDATE 不触发 AFTER
-- INSERT）+状态谓词（已 retired/fin_retired 不再触）。ROW 级（NEW 行语义——
-- STATEMENT 级 NEW=NULL 实证零匹配，batch 入册逐行等价）。
-- 覆盖手工入册径；服务侧自动径（hedgeOpener.untradeable）同 trigger 双保险
-- （谓词幂等零冲突）。
CREATE OR REPLACE FUNCTION fn_untradeable_retro() RETURNS trigger AS $$
BEGIN
  INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
  SELECT id, state, 'retired', 'venue_untradeable_retroactive:'||NEW.code
  FROM fin_strategy WHERE inst_id=NEW.inst_id AND state IN ('live','paper','proposed');
  UPDATE fin_strategy SET state='retired'
  WHERE inst_id=NEW.inst_id AND state IN ('live','paper','proposed');
  UPDATE fin_proposal SET state='fin_retired'
  WHERE inst_id=NEW.inst_id AND state='fin_armed';
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_untradeable_retro ON fin_venue_untradeable;
CREATE TRIGGER trg_untradeable_retro AFTER INSERT ON fin_venue_untradeable
  FOR EACH ROW EXECUTE FUNCTION fn_untradeable_retro();
