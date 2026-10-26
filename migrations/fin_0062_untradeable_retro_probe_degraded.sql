-- 0062 T-0151 黑名单回扫覆面补两态（T-0147 验收衍生——策略 286181
-- SHEIN-USDT-SWAP probe 态漏扫滞留实案）：0061 trigger 谓词只覆
-- live/paper/proposed 三态，黑名单 inst 上的 probe/degraded 策略滞留
-- ——probe→live 晋升后首开仓才被 opener 闸退役，窗口期在册活着=状态机
-- 谎言。本迁移把谓词集扩为六态闭集减 retired（'live','paper','proposed',
-- 'probe','degraded'）——retired 外全覆，闭集拒自挡已退役行（幂等二重
-- 同 0061：inst_id PK 重复入册走 UPDATE 不触发 AFTER INSERT+状态谓词）。
-- 另带存量清尾一次（trigger 只打新入册行——已入册黑名单 inst 上的漏网
-- 非 retired 策略由本迁移一次性回扫，语义同 trigger 体：transition 行
-- 溯源+armed 提案 fin_retired）。
CREATE OR REPLACE FUNCTION fn_untradeable_retro() RETURNS trigger AS $$
BEGIN
  INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
  SELECT id, state, 'retired', 'venue_untradeable_retroactive:'||NEW.code
  FROM fin_strategy WHERE inst_id=NEW.inst_id
    AND state IN ('live','paper','proposed','probe','degraded');
  UPDATE fin_strategy SET state='retired'
  WHERE inst_id=NEW.inst_id
    AND state IN ('live','paper','proposed','probe','degraded');
  UPDATE fin_proposal SET state='fin_retired'
  WHERE inst_id=NEW.inst_id AND state='fin_armed';
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_untradeable_retro ON fin_venue_untradeable;
CREATE TRIGGER trg_untradeable_retro AFTER INSERT ON fin_venue_untradeable
  FOR EACH ROW EXECUTE FUNCTION fn_untradeable_retro();

-- 存量清尾（一次）：0061 生效期入册的黑名单 inst 上漏网 probe/degraded
-- （及他非 retired 态）策略，此处回扫退役+溯源行——票面验收锚「存量黑
-- 名单 inst 上任意非 retired 策略=零」。已 retired 行被谓词自挡=重跑幂等。
INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
SELECT s.id, s.state, 'retired', 'venue_untradeable_retroactive:0062_backfill'
FROM fin_strategy s JOIN fin_venue_untradeable u ON u.inst_id=s.inst_id
WHERE s.state IN ('live','paper','proposed','probe','degraded');

UPDATE fin_strategy s SET state='retired'
FROM fin_venue_untradeable u
WHERE u.inst_id=s.inst_id
  AND s.state IN ('live','paper','proposed','probe','degraded');

UPDATE fin_proposal p SET state='fin_retired'
FROM fin_venue_untradeable u
WHERE u.inst_id=p.inst_id AND p.state='fin_armed';
