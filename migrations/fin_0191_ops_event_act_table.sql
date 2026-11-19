-- fin_0191: ops_event_act 视图→触发器维护表(fin_0190续刀——审计回扫:视图每查重解析19k行正则,桌面10s+)
-- 解析函数 ops_act_parse 单点化:触发器写入径+回填径共用同一表达式——「解析一次」做实
-- 级标绑 rel(首引证票):工单#A…工单#B 级=P1 不再误绑A
-- ops_event 事实行零改(append-only);act为派生表可随时重建——DROP重建径见本文件尾部注释
BEGIN;

-- ①解析函数(单点)——返回 rel/act/susp_cls/susp_hours/level 五元组
CREATE OR REPLACE FUNCTION ops_act_parse(p_event text, p_note text, p_ticket text, p_sem jsonb)
RETURNS TABLE(rel bigint, act text, susp_cls text, susp_hours int, level int)
LANGUAGE sql IMMUTABLE AS $$
  SELECT r.rel,
    COALESCE(p_sem->>'act', CASE
      WHEN p_event='认领' AND r.rel IS NOT NULL THEN 'claim'
      WHEN p_event='在办' AND r.rel IS NOT NULL THEN 'doing'
      WHEN p_event='处置' AND r.rel IS NOT NULL AND p_note ~ '挂起条件=' THEN 'suspend'
      WHEN p_event='处置' AND r.rel IS NOT NULL AND p_note ~ '挂起解=' THEN 'resume'
      WHEN p_event='处置' AND r.rel IS NOT NULL
        AND p_note ~ ('工单#'||r.rel||' ?接续') THEN 'continue'
      WHEN p_event='处置' AND r.rel IS NOT NULL
        AND p_note ~ ('工单#'||r.rel||'[^—#]{0,20}(销[—：:，,]|销$)') THEN 'close'
      WHEN p_event='处置' AND r.rel IS NOT NULL AND p_note ~ '回执' THEN 'receipt'
      WHEN p_event='处置' AND r.rel IS NOT NULL THEN 'note'
    END) AS act,
    CASE WHEN p_event='处置' AND p_note ~ '挂起条件=' THEN
      COALESCE(p_sem->>'cls', CASE
        WHEN substring(p_note from '挂起条件=([^—]*)') ~ '认领窗|施工窗|上下文|预算耗尽' THEN 'capacity'
        WHEN substring(p_note from '挂起条件=([^—]*)') ~ '裁定|细裁|DM|授权|口径|规格' THEN 'adjudication'
        ELSE 'other' END)
    END AS susp_cls,
    (regexp_match(p_note,'挂起期限=([0-9]+)h'))[1]::int AS susp_hours,
    COALESCE((p_sem->>'level')::int,
      (regexp_match(p_note,'工单#'||r.rel||'[^#]{0,30}级[ ]*[升降=]*[ ]*P([123])'))[1]::int) AS level
  FROM (SELECT COALESCE((p_sem->>'rel')::bigint,
    (regexp_match(p_note,'工单#([0-9]+)'))[1]::bigint,
    (regexp_match(COALESCE(p_ticket,''),'^(?:工单#)?([0-9]+)$'))[1]::bigint) AS rel) r
$$;

-- ②派生表(同名列替视图——desk视图定义零改)
DROP VIEW IF EXISTS ops_ticket_desk;
DROP VIEW IF EXISTS ops_event_act;
CREATE TABLE IF NOT EXISTS ops_event_act(
  id bigint PRIMARY KEY, ts timestamptz, seat text, event text,
  rel bigint, act text, susp_cls text, susp_hours int, level int);
CREATE INDEX IF NOT EXISTS ops_event_act_rel_i ON ops_event_act(rel);
CREATE INDEX IF NOT EXISTS ops_event_act_ts_i ON ops_event_act(ts);

-- ③写入即解析触发器
CREATE OR REPLACE FUNCTION ops_event_act_trg() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level)
    SELECT NEW.id,NEW.ts,NEW.seat,NEW.event,p.rel,p.act,p.susp_cls,p.susp_hours,p.level
    FROM ops_act_parse(NEW.event,NEW.note,NEW.ticket,NEW.sem) p;
  RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS ops_event_act_ai ON ops_event;
CREATE TRIGGER ops_event_act_ai AFTER INSERT ON ops_event
  FOR EACH ROW EXECUTE FUNCTION ops_event_act_trg();

-- ④历史回填(同函数——新旧解析同源)
INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level)
  SELECT e.id,e.ts,e.seat,e.event,p.rel,p.act,p.susp_cls,p.susp_hours,p.level
  FROM ops_event e, ops_act_parse(e.event,e.note,e.ticket,e.sem) p
  ON CONFLICT (id) DO NOTHING;

-- ⑤desk视图原样重建(定义与fin_0190逐字一致——读面零漂)
CREATE OR REPLACE VIEW ops_ticket_desk AS
SELECT t.id, t.ts filed_ts, t.note,
  co.closed, cl.claimed, sp.susp_until, sp.susp_cls,
  (co.closed IS NOT TRUE AND cl.claimed IS NOT TRUE
    AND (sp.susp_until IS NULL OR sp.susp_cls='capacity')) AS workable,
  (co.closed IS NOT TRUE AND sp.susp_until IS NOT NULL
    AND sp.susp_cls='adjudication') AS rulable,
  COALESCE(la.last_active, t.ts) AS last_active,
  COALESCE(lvl.level, (regexp_match(t.note,'级[ ]*[升降=]*[ ]*P([123])'))[1]::int) AS level,
  sp.susp_since,
  t.ts + COALESCE((regexp_match(t.note,'期限=([0-9]+)h'))[1]::int,24)*interval '1 hour' AS deadline
FROM ops_event t
LEFT JOIN LATERAL (SELECT bool_or(a.act='close') AS closed
  FROM ops_event_act a WHERE a.rel=t.id AND a.ts>t.ts) co ON true
LEFT JOIN LATERAL (SELECT count(*)>0 AS claimed FROM ops_event_act a
  WHERE a.rel=t.id AND a.ts>t.ts AND a.act='claim'
    AND a.ts>now()-interval '2 hours') cl ON true
LEFT JOIN LATERAL (
  SELECT s.ts + COALESCE(s.susp_hours,
           (regexp_match(t.note,'期限=([0-9]+)h'))[1]::int,72)*interval '1 hour' AS susp_until,
         s.susp_cls, s.ts AS susp_since
  FROM ops_event_act s
  WHERE s.rel=t.id AND s.ts>t.ts AND s.act='suspend'
    AND s.ts + COALESCE(s.susp_hours,
          (regexp_match(t.note,'期限=([0-9]+)h'))[1]::int,72)*interval '1 hour' > now()
    AND NOT EXISTS (SELECT 1 FROM ops_event_act r
      WHERE r.rel=t.id AND r.ts>s.ts AND r.act IN ('resume','close'))
  ORDER BY s.ts DESC LIMIT 1) sp ON true
LEFT JOIN LATERAL (SELECT max(a.ts) AS last_active FROM ops_event_act a
  WHERE a.rel=t.id AND a.ts>t.ts AND a.act IS NOT NULL) la ON true
LEFT JOIN LATERAL (SELECT a.level FROM ops_event_act a
  WHERE a.rel=t.id AND a.level IS NOT NULL ORDER BY a.ts DESC LIMIT 1) lvl ON true
WHERE t.event='工单' AND t.ts>now()-interval '30 days';

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT ON ops_event_act, ops_ticket_desk TO roster_rw';
        EXECUTE 'GRANT INSERT ON ops_event_act TO roster_rw';
    END IF;
END $$;
COMMIT;

