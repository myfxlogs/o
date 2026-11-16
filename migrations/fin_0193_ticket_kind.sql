-- fin_0193: 票型列 tkind——施工桌面滤方向件根修
-- 病灶实证:方向/票被施工误拣→认领→核边界→挂起还票弹跳烧会话(#19797在案;
--   无标方向类件#18314/#18757同款雷)。票型前缀立案契已钦定(工单//方向/),
--   桌面谓词没消费——语义层写径解析一次落结构列,谓词读列不读词面。
-- 立案行自载tkind(event='工单'且note携「<型>/<slug>」前缀);desk露列;
-- 滤口径=tkind='工单'白名单(与立案契同——非契约型不上施工桌,唤知桌面全见不遮)。
BEGIN;

-- ①解析函数加tkind出参(返回列变=DROP重建,事务内同窗零缝;
--   ops_event_act_trg为plpgsql体对ops_act_parse无硬依赖,安全)
DROP FUNCTION IF EXISTS ops_act_parse(text,text,text,jsonb);
CREATE FUNCTION ops_act_parse(p_event text, p_note text, p_ticket text, p_sem jsonb)
RETURNS TABLE(rel bigint, act text, susp_cls text, susp_hours int, level int, tkind text)
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
      (regexp_match(p_note,'工单#'||r.rel||'[^#]{0,30}级[ ]*[升降=]*[ ]*P([123])'))[1]::int) AS level,
    CASE WHEN p_event='工单' THEN substring(p_note from '^([^\s/]{1,12})/') END AS tkind
  FROM (SELECT COALESCE((p_sem->>'rel')::bigint,
    (regexp_match(p_note,'工单#([0-9]+)'))[1]::bigint,
    (regexp_match(COALESCE(p_ticket,''),'^(?:工单#)?([0-9]+)$'))[1]::bigint) AS rel) r
$$;

-- ②act表加列+触发器插列(写径解析一次——立案行不可变,正合写时语义)
ALTER TABLE ops_event_act ADD COLUMN IF NOT EXISTS tkind text;
CREATE OR REPLACE FUNCTION ops_event_act_trg() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level,tkind)
    SELECT NEW.id,NEW.ts,NEW.seat,NEW.event,p.rel,p.act,p.susp_cls,p.susp_hours,p.level,p.tkind
    FROM ops_act_parse(NEW.event,NEW.note,NEW.ticket,NEW.sem) p;
  RETURN NEW;
END $$;

-- ③立案行历史回填(同表达式——写径/回填同源铁律)
UPDATE ops_event_act a SET tkind=substring(e.note from '^([^\s/]{1,12})/')
FROM ops_event e
WHERE e.id=a.id AND e.event='工单' AND e.note ~ '^[^\s/]{1,12}/';

-- ④desk视图露tkind列(定义承fin_0191逐字+尾列追加)
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
  t.ts + COALESCE((regexp_match(t.note,'期限=([0-9]+)h'))[1]::int,24)*interval '1 hour' AS deadline,
  tk.tkind
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
LEFT JOIN ops_event_act tk ON tk.id=t.id
WHERE t.event='工单' AND t.ts>now()-interval '30 days';

COMMIT;
