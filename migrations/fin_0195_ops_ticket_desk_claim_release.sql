-- 0195 认领租约收口释租（工单#19677 claim-lease-hostage-v2，fin_0190
-- 语义层根修之续案——0190 注「认领=2h 在办租约(不动,#19677 另案)」；
-- fin_0191 已把 ops_event_act 视图表化=触发器维护派生表，fin_0194 在
-- 其上叠 WIP 服务窗，本迁移只动 ops_ticket_desk 视图一处——cl lateral
-- 认领谓词加释租臂，不碰 act 表/解析函数/WIP 排位层）：
-- 2h 认领租约>会话寿命——会话收口后认领行仍占租约焊死票（实证#19506
-- 21:40认领21:41收口锁至23:41；#19645误挂起自焊死）。修法=票面裁定径②
-- 「收口行释租」：认领被同 unit 会话收口行作废（unit=为会话唯一锚——
-- o-worker-<epoch>/o-self-<epoch> 不复用；无 unit= 的旧席认领行回落
-- 2h 原租约不失效）。径①「绑 ops_headless last_activity」实测不可行——
-- headless 按日×席粒度（WRK-20261116 单行覆全日施工席）非会话粒度，
-- 异会话存活即续命=租约永不释，弃。
-- desk 视图体以 fin_0194（=fin_0193 tkind+0194 WIP窗 逐字同形）正本为基，
-- 唯一 delta=cl lateral 认领谓词：同 unit 会话收口行(ts>认领行)作废该认领。

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
  -- 排位集=施工道可拣池(workable∧工单型)——方向/无标件不占窗额
  -- (方向件workable为真但永不入施工道,排位进来=死槽占位)
  SELECT id, rank() OVER (ORDER BY COALESCE(level,2), deadline, id) AS dispatch_rank
  FROM d WHERE workable AND tkind='工单'
)
SELECT d.id, d.filed_ts, d.note, d.closed, d.claimed, d.susp_until, d.susp_cls,
  d.workable, d.rulable, d.last_active, d.level, d.susp_since, d.deadline, d.tkind,
  w.dispatch_rank,
  (d.workable AND w.dispatch_rank <= COALESCE(
    (SELECT value::int FROM app_config WHERE key='dm.cog.wip_limit'),20)) AS dispatchable
FROM d LEFT JOIN w ON w.id=d.id;

GRANT SELECT ON ops_ticket_desk TO roster_rw;

COMMIT;
