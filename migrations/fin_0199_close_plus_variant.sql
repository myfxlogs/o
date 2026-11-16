-- fin_0199: 销契连写变体补齐（工单#15805 接续——fin_0191 act 表化后实证变体，
-- 处置席接续判词 #20151 修法向=宽连写形）：
-- 批销连写「工单#N 销+升件#M 销」销后随 '+' 非记号契 follower 类
-- （[—：:，,]|$）→ close 谓词落空坠 '回执'/'note'（#19713/#19714 实证
-- act=receipt——票假活至补正销行才闭）。修法=ops_act_parse close 臂
-- follower 类补 '+'（连写分隔符，非销态词——裸引不销契不变）；
-- 返回列承 fin_0193 tkind 六元组签名（DROP 重建同式）。act 为派生表
-- 随迁重建（fin_0191 注「可随时重建」），历史误判行同函数重解析归位。
-- ops_event 事实行零改（append-only）。

-- ①解析函数（fin_0193 签名+tkind 出参原样，唯 close follower 类补 '+'）
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
        AND p_note ~ ('工单#'||r.rel||'[^—#]{0,20}(销[—：:，,+]|销$)') THEN 'close'
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

-- ②act 派生表全量重建（DELETE+同函数重解析含 tkind 列——fin_0191/0193
-- 回填径同式；触发器写入径自此刻起自动走新谓词，历史行归位）。
DELETE FROM ops_event_act;
INSERT INTO ops_event_act(id,ts,seat,event,rel,act,susp_cls,susp_hours,level,tkind)
  SELECT e.id,e.ts,e.seat,e.event,p.rel,p.act,p.susp_cls,p.susp_hours,p.level,p.tkind
  FROM ops_event e, ops_act_parse(e.event,e.note,e.ticket,e.sem) p;
