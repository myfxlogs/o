-- fin_0209: close-finality 引证收窄（工单#15805——处置行裸写「工单#id」
-- 20 字窗内见「销」字即把 OPEN 票压出面；三连实证#14140/#15636/#15664
-- 工件未竟靠继票抢救，#21745「勘误续不销/续验待验不销」否义句被销$误吞）。
-- 修法=close 臂销词紧邻化：票号后空白≤2 即须显式销态词（销案/已销/销票/
-- 销+记号契 follower；范围销「#N-M 销」与「裁定落+销」变体保留），
-- 票号与销词间任意文字=引证不算销；否义面（不销/勿销/暂销/缓销/误销）
-- 20 字窗内显否优先。销契键（sem act）径不变。
-- 随迁重建（fin_0199 同式），历史误判行同函数重解析归位——误销票回 OPEN，
-- ops_event 事实行零改（append-only）。

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
        AND p_note ~ ('工单#'||r.rel||'([ ]{0,2}(销案|已销|销票|销[——：:，,+$]|销$|裁定落\+销)|[-—][0-9]+[ ]*销)')
        AND p_note !~ ('工单#'||r.rel||'[^—#]{0,20}(不销|勿销|暂销|缓销|误销)') THEN 'close'
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

-- act 派生表全量重建（fin_0199 同式——含 tkind 列；触发器写入径自此刻
-- 走新谓词，历史行归位：误销票「续不销/引证中缝」回 note 票复 OPEN）。
DELETE FROM ops_event_act;
INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level,tkind)
  SELECT e.id,e.ts,e.seat,e.event,p.rel,p.act,p.susp_cls,p.susp_hours,p.level,p.tkind
  FROM ops_event e, ops_act_parse(e.event,e.note,e.ticket,e.sem) p;
