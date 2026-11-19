-- fin_0212: dispatchable 证锚硬径（工单#25482——立案闸机检化：
-- 「工单」立案必载证=<可核锚>,无证件驻工单N计数但永不入派发池
-- (看得见派不出,逼分诊补证或销——不静默丢票)。动因=#24798 未核
-- 立案致施工踩空 ~1h/方向旗混桌污染可拣数;软径(纪律自觉)已被证败。
-- 径=w CTE 排位集谓词加「d.note ~ '证='」——排位=施工道唯一入口,
-- 无证即无 rank→dispatchable=false;workable 照真(账面不藏)。
-- 兼容面:存世开工单唯 #24798(挂起中非 workable)零冲击;方向/其他
-- tkind 本不入 w 不受影响;receipt 剔谓词(fin_0208)原样保留。
-- 视图体以 fin_0208 正本逐字为基,唯一 delta=w CTE 一行谓词。

BEGIN;

CREATE OR REPLACE VIEW ops_ticket_desk AS
WITH d AS (
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
  JOIN ops_event ce ON ce.id=a.id
  WHERE a.rel=t.id AND a.ts>t.ts AND a.act='claim'
    AND a.ts>now()-interval '2 hours'
    AND NOT EXISTS (SELECT 1 FROM ops_event sc
      WHERE sc.event='会话收口' AND sc.ts>a.ts
        AND substring(sc.note from 'unit=([A-Za-z0-9_-]+)')
          = substring(ce.note from 'unit=([A-Za-z0-9_-]+)'))) cl ON true
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
WHERE t.event='工单' AND t.ts>now()-interval '30 days'
),
w AS (
  -- 排位集=施工道可拣池(workable∧工单型∧证锚载∧非完工候验末态)——
  -- 方向/无标件不占窗额(方向件workable为真但永不入施工道,排位进来=
  -- 死槽占位);无证工单(fin_0212 工单#25482)=立案闸硬径派不出;
  -- receipt末态=唤知验收件(工单#23931:候验票占窗=空唤出血源)。
  SELECT id, rank() OVER (ORDER BY COALESCE(level,2), deadline, id) AS dispatch_rank
  FROM d WHERE workable AND tkind='工单' AND d.note ~ '证='
    AND COALESCE((SELECT a.act FROM ops_event_act a
      WHERE a.rel=d.id AND a.act IS NOT NULL
      ORDER BY a.ts DESC LIMIT 1),'')<>'receipt'
)
SELECT d.id, d.filed_ts, d.note, d.closed, d.claimed, d.susp_until, d.susp_cls,
  d.workable, d.rulable, d.last_active, d.level, d.susp_since, d.deadline, d.tkind,
  w.dispatch_rank,
  (d.workable AND w.dispatch_rank <= COALESCE(
    (SELECT value::int FROM app_config WHERE key='dm.cog.wip_limit'),20)) AS dispatchable
FROM d LEFT JOIN w ON w.id=d.id;

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT ON ops_ticket_desk TO roster_rw';
    END IF;
END $$;

COMMIT;
