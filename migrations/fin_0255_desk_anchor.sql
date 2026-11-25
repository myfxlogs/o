-- 0255 工单出生锚契约（工单O-39410——「票不带锚」病灶根修：
-- 立案自由不变，锚=事件行 sem jsonb 结构化键+desk 视图投影列）：
-- ① ops_desk_anchor_derive(note,sem)——出生锚推导纯函数：显式
--   sem 锚优先（写径可钉）→venue= 引用=premise/venue:<v>→升件#<id>
--   =escalation/esc:<id>→D-档=design/<ref>→余=manual/''（判类序=
--   premise 优先——venue 票引升件时前提锚仍须可命中 premise-death）。
-- ② ops_desk_anchor_trg BEFORE INSERT——event='工单' 行落库前
--   NEW.sem 合并锚键（|| 合并不覆写存量键；derive 见 anchor_type
--   已在=返回空=显式锚恒赢）。立案九口零改造全覆盖=出生锚契约
--   机械落实，非流程闸。
-- ③ 存量回填：近30d 工单行（视图窗同界）sem 无锚→derive 补键；
--   推导不出=anchor_type='legacy'（终身标——禁事后改锚含义）。
--   UPDATE 须过 immutable_trg——迁移 tx 内 DISABLE/ENABLE 明示
--   （ops_event_immutable 报错文自载订正径；仅锚键簿记非状态翻）。
-- ④ ops_ticket_desk 视图尾加 anchor_type/anchor_ref 投影列
--   （CREATE OR REPLACE 尾加列律——存量列序不动）。

CREATE OR REPLACE FUNCTION ops_desk_anchor_derive(p_note text, p_sem jsonb)
RETURNS jsonb LANGUAGE plpgsql IMMUTABLE AS $fn$
DECLARE
  v_ref text;
BEGIN
  IF p_sem IS NOT NULL AND p_sem ? 'anchor_type' THEN
    RETURN '{}'::jsonb; -- 显式锚恒赢——derive 不覆写
  END IF;
  v_ref := (regexp_match(p_note, 'venue=''?([a-zA-Z0-9_-]+)''?'))[1];
  IF v_ref IS NOT NULL THEN
    RETURN jsonb_build_object('anchor_type','premise','anchor_ref','venue:'||v_ref);
  END IF;
  v_ref := (regexp_match(p_note, '升件#([0-9]+)'))[1];
  IF v_ref IS NOT NULL THEN
    RETURN jsonb_build_object('anchor_type','escalation','anchor_ref','esc:'||v_ref);
  END IF;
  v_ref := (regexp_match(p_note, '(D-[0-9]{3,5})'))[1];
  IF v_ref IS NOT NULL THEN
    RETURN jsonb_build_object('anchor_type','design','anchor_ref',v_ref);
  END IF;
  RETURN jsonb_build_object('anchor_type','manual','anchor_ref','');
END;
$fn$;

CREATE OR REPLACE FUNCTION ops_desk_anchor_trg() RETURNS trigger AS $fn$
BEGIN
  IF NEW.event='工单' THEN
    NEW.sem := COALESCE(NEW.sem,'{}'::jsonb)
      || ops_desk_anchor_derive(NEW.note, NEW.sem);
  END IF;
  RETURN NEW;
END;
$fn$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ops_desk_anchor_trg ON ops_event;
CREATE TRIGGER ops_desk_anchor_trg BEFORE INSERT ON ops_event
  FOR EACH ROW EXECUTE FUNCTION ops_desk_anchor_trg();

ALTER TABLE ops_event DISABLE TRIGGER ops_event_immutable_trg;
UPDATE ops_event e SET sem = COALESCE(e.sem,'{}'::jsonb) ||
  CASE WHEN ops_desk_anchor_derive(e.note,e.sem)->>'anchor_type'='manual'
       THEN jsonb_build_object('anchor_type','legacy','anchor_ref','')
       ELSE ops_desk_anchor_derive(e.note,e.sem) END
WHERE e.event='工单' AND e.ts > now()-interval '30 days'
  AND NOT (COALESCE(e.sem,'{}'::jsonb) ? 'anchor_type');
ALTER TABLE ops_event ENABLE TRIGGER ops_event_immutable_trg;

CREATE OR REPLACE VIEW ops_ticket_desk AS
WITH d AS (
         SELECT t.id,
            t.ts AS filed_ts,
            t.note,
            co.closed,
            cl.claimed,
            sp.susp_until,
            sp.susp_cls,
            ((co.closed IS NOT TRUE) AND (cl.claimed IS NOT TRUE) AND ((sp.susp_until IS NULL) OR (sp.susp_cls = 'capacity'::text))) AS workable,
            ((co.closed IS NOT TRUE) AND (sp.susp_until IS NOT NULL) AND (sp.susp_cls = 'adjudication'::text)) AS rulable,
            COALESCE(la.last_active, t.ts) AS last_active,
            COALESCE(lvl.level, ((regexp_match(t.note, '级[ ]*[升降=]*[ ]*P([123])'::text))[1])::integer) AS level,
            sp.susp_since,
            (t.ts + ((COALESCE(((regexp_match(t.note, '期限=([0-9]+)h'::text))[1])::integer, 24))::double precision * '01:00:00'::interval)) AS deadline,
            tk.tkind,
            COALESCE(t.sem->>'anchor_type','legacy') AS anchor_type,
            COALESCE(t.sem->>'anchor_ref','') AS anchor_ref
           FROM ((((((ops_event t
             LEFT JOIN LATERAL ( SELECT bool_or((a.act = 'close'::text)) AS closed
                   FROM ops_event_act a
                  WHERE ((a.rel = t.id) AND (a.ts > t.ts))) co ON (true))
             LEFT JOIN LATERAL ( SELECT (count(*) > 0) AS claimed
                   FROM (ops_event_act a
                     JOIN ops_event ce ON ((ce.id = a.id)))
                  WHERE ((a.rel = t.id) AND (a.ts > t.ts) AND (a.act = 'claim'::text) AND (a.ts > (now() - '02:00:00'::interval)) AND (NOT (EXISTS ( SELECT 1
                           FROM ops_event sc
                          WHERE ((sc.event = '会话收口'::text) AND (sc.ts > a.ts) AND ("substring"(sc.note, 'unit=([A-Za-z0-9_-]+)'::text) = "substring"(ce.note, 'unit=([A-Za-z0-9_-]+)'::text)))))))) cl ON (true))
             LEFT JOIN LATERAL ( SELECT (s.ts + ((COALESCE(s.susp_hours, ((regexp_match(t.note, '期限=([0-9]+)h'::text))[1])::integer, 72))::double precision * '01:00:00'::interval)) AS susp_until,
                    s.susp_cls,
                    s.ts AS susp_since
                   FROM ops_event_act s
                  WHERE ((s.rel = t.id) AND (s.ts > t.ts) AND (s.act = 'suspend'::text) AND ((s.ts + ((COALESCE(s.susp_hours, ((regexp_match(t.note, '期限=([0-9]+)h'::text))[1])::integer, 72))::double precision * '01:00:00'::interval)) > now()) AND (NOT (EXISTS ( SELECT 1
                           FROM ops_event_act r
                          WHERE ((r.rel = t.id) AND (r.ts > s.ts) AND (r.act = ANY (ARRAY['resume'::text, 'close'::text])))))))
                  ORDER BY s.ts DESC
                 LIMIT 1) sp ON (true))
             LEFT JOIN LATERAL ( SELECT max(a.ts) AS last_active
                   FROM ops_event_act a
                  WHERE ((a.rel = t.id) AND (a.ts > t.ts) AND (a.act IS NOT NULL))) la ON (true))
             LEFT JOIN LATERAL ( SELECT a.level
                   FROM ops_event_act a
                  WHERE ((a.rel = t.id) AND (a.level IS NOT NULL))
                  ORDER BY a.ts DESC
                 LIMIT 1) lvl ON (true))
             LEFT JOIN ops_event_act tk ON ((tk.id = t.id)))
          WHERE ((t.event = '工单'::text) AND (t.ts > (now() - '30 days'::interval)))
        ), w AS (
         SELECT d_1.id,
            rank() OVER (ORDER BY COALESCE(d_1.level, 2), d_1.deadline, d_1.id) AS dispatch_rank
           FROM d d_1
          WHERE (d_1.workable AND (d_1.tkind = '工单'::text) AND (d_1.note ~ '证='::text) AND (COALESCE(( SELECT a.act
                   FROM ops_event_act a
                  WHERE ((a.rel = d_1.id) AND (a.act IS NOT NULL))
                  ORDER BY a.ts DESC
                 LIMIT 1), ''::text) <> 'receipt'::text))
        )
 SELECT d.id,
    d.filed_ts,
    d.note,
    d.closed,
    d.claimed,
    d.susp_until,
    d.susp_cls,
    d.workable,
    d.rulable,
    d.last_active,
    d.level,
    d.susp_since,
    d.deadline,
    d.tkind,
    w.dispatch_rank,
    (d.workable AND (w.dispatch_rank <= COALESCE(( SELECT (app_config.value)::integer AS value
           FROM app_config
          WHERE (app_config.key = 'dm.cog.wip_limit'::text)), 20))) AS dispatchable,
    d.anchor_type,
    d.anchor_ref
   FROM (d
     LEFT JOIN w ON ((w.id = d.id)));
