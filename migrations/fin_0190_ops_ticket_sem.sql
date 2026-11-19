-- 0190 工单态语义层根修——「note→语义」解析从N处谓词收敛为单点视图层。
-- 病灶族实证(2026-11-16 业主令「直接根修不走止血」):
--   引证误销#19499 / 容量自锁#19506 / 回执无销#19669 / 级降P1措辞不生效 / 等裁定面饥饿
--   ——母病=「工单#id 引证+销/挂起/接续/级」同义正则散落在 dispatch.sh×4 /
--   dispatch_worker / 三桌面 runbook,各写一份各修一遍,修一漏一。
-- 修法=单点翻译层:
--   ops_event.sem JSONB —— writer 可选显式语义槽(act/cls/rel/until/level),
--                          不解析散文;NULL=走下方解析(旧行全 NULL 无侵)。
--   ops_event_act 视图 —— 全库唯一 note→(rel,act,susp_cls,susp_hours,level)
--                          解析处。act∈close/suspend/resume/continue/receipt/claim/doing。
--   ops_ticket_desk 视图 —— 每票一行派生态:closed/claimed/susp_until/susp_cls/
--                          workable(可拣)/rulable(可裁)/last_active/level。
--   (名撞避让:ops_ticket_state 为 fin_0086 驾驶舱「每票最新事件」视图,在役)
-- 语义与原谓词逐点对齐:
--   销=显式销标处置行(销——|销:|销,|工单#N 销|行尾销)——「销票归唤知」
--   等名词性引用不再误销(#15866实证:回执载「销票归唤知」被旧谓词字面
--   销);无标记处置=act'note'在案活动非销票(隐式销=引证误销同族绝根,
--   未结票滞留探兜底复验);
--   挂起=处置载「挂起条件=」,窗=s.ts+挂起期限h∥票期限h∥72h;
--   挂起解=<裁定结果>=resume(新钦定复办记号——解除挂起不销票);
--   认领=2h 在办租约(不动,#19677 另案);级=级P/级降P/级升P 三写法同解。
--   引证收窄=act只绑首引「工单#N」一张票——同处置行后引证=散文引用
--   不产生动作(引证误销绝根)。
-- ops_event 事实行零改零 UPDATE——append-only 铁律无损,翻译层可随时重建。

ALTER TABLE ops_event ADD COLUMN IF NOT EXISTS sem JSONB;

CREATE OR REPLACE VIEW ops_event_act AS
SELECT e.id, e.ts, e.seat, e.event, r.rel,
  COALESCE(e.sem->>'act', CASE
    WHEN e.event='认领' AND r.rel IS NOT NULL THEN 'claim'
    WHEN e.event='在办' AND r.rel IS NOT NULL THEN 'doing'
    WHEN e.event='处置' AND r.rel IS NOT NULL AND e.note ~ '挂起条件=' THEN 'suspend'
    WHEN e.event='处置' AND r.rel IS NOT NULL AND e.note ~ '挂起解=' THEN 'resume'
    WHEN e.event='处置' AND r.rel IS NOT NULL
      AND e.note ~ ('工单#'||r.rel||' ?接续') THEN 'continue'
    WHEN e.event='处置' AND r.rel IS NOT NULL
      AND e.note ~ ('工单#'||r.rel||'[^—#]{0,20}(销[—：:，,]|销$)') THEN 'close'
    WHEN e.event='处置' AND r.rel IS NOT NULL AND e.note ~ '回执' THEN 'receipt'
    WHEN e.event='处置' AND r.rel IS NOT NULL THEN 'note'
  END) AS act,
  CASE WHEN e.event='处置' AND e.note ~ '挂起条件=' THEN
    COALESCE(e.sem->>'cls', CASE
      WHEN substring(e.note from '挂起条件=([^—]*)') ~ '认领窗|施工窗|上下文|预算耗尽' THEN 'capacity'
      WHEN substring(e.note from '挂起条件=([^—]*)') ~ '裁定|细裁|DM|授权|口径|规格' THEN 'adjudication'
      ELSE 'other' END)
  END AS susp_cls,
  (regexp_match(e.note,'挂起期限=([0-9]+)h'))[1]::int AS susp_hours,
  COALESCE((e.sem->>'level')::int,
    (regexp_match(e.note,'工单#[0-9]+[^#]{0,30}级[ ]*[升降=]*[ ]*P([123])'))[1]::int) AS level
FROM ops_event e,
LATERAL (SELECT COALESCE((e.sem->>'rel')::bigint,
    (regexp_match(e.note,'工单#([0-9]+)'))[1]::bigint,
    (regexp_match(COALESCE(e.ticket,''),'^(?:工单#)?([0-9]+)$'))[1]::bigint) AS rel) r;

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
    END IF;
END $$;
