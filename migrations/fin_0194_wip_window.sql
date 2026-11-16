-- fin_0194: 工单面在制品限界(WIP window)——队列开环根修
-- 病灶实证(2026-11-17 工单面动线审计):立案35/天>施工产能19/天净涨14/天;
--   滞留探把每张晾票变门口件→处置会话顺手又立案57件/48h→桌面更深→更多
--   晾票→更多升件=正反馈自喂环(滞留探注释自招「会话只办门口件作业本不拽活」)。
--   根=工单队列是全身唯一没闭环的器官:钱/新鲜/健康各有探针闭环,队列没有。
-- 修法=在制品限界(看板第一性):
--   dispatchable=workable中按(级,期限,id)排前N张=服务窗,窗内才可拣;
--   窗外=backlog在账不可拣——λ>μ时饿死按级排不随机,饿死可见性本身成信号。
--   滞留探收窗内(dispatchable)——backlog不逐票升件,改一张聚合压力卡;
--   窗阈=app_config cog.wip_limit DM可调(默认20),backlog告警阈env件。
BEGIN;

-- ①desk重定义:内层承fin_0193逐字,外包一层仅对workable集排位露两列
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
    (SELECT value::int FROM app_config WHERE key='cog.wip_limit'),20)) AS dispatchable
FROM d LEFT JOIN w ON w.id=d.id;

-- ②窗阈种子(DM可调——o-core热读app_config,改窗宽零部署)
INSERT INTO app_config(key,value,updated_by) VALUES
  ('cog.wip_limit','20','fin_0194')
ON CONFLICT (key) DO NOTHING;

COMMIT;
