-- 0153 T-0499 相位投影键卫生守卫——ops_ticket_phase 视图重建：
-- fin_0147 同形词集/相位映射一字不动，只加 ticket 键卫生 WHERE 守卫——
-- 含空白/逗号族（空格/\n/\t/，/）或多票号形态（T-\d{4}≥2）的拼接伪键
-- 不产生相位行（投影面清尸——写面闸 scripts/meta/ops_event_key_gate.sh
-- 同判据双闸：写面落库前拒+投影面不收）。尸源实证=换行拼接
-- 'T-0270\nT-0277'/'T-0276\nT-0276\nT-0276\nT-0276' 形（chr(10)⊂
-- [[:space:]] 全覆盖；逗号拼接/无空白多票号同收）。存量脏行 append-only
-- 不清洗——读面守卫归位（条目 grep-m1-o-multimatch-newline-tid 同律）。
CREATE OR REPLACE VIEW ops_ticket_phase AS
SELECT l.ticket, e.event AS latest_event, e.ts AS latest_ts,
       e.seat AS latest_seat, e.note AS latest_note,
       CASE e.event
         WHEN '起草'     THEN 'draft'
         WHEN '呈批'     THEN 'approval_pending'
         WHEN '已批'     THEN 'approved'
         WHEN '派单'     THEN 'dispatched'
         WHEN '拉起'     THEN 'pulled'
         WHEN '领单'     THEN 'pulled'
         WHEN '开工'     THEN 'working'
         WHEN '外派'     THEN 'external'
         WHEN '收口'     THEN 'review'
         WHEN '移交'     THEN 'review'
         WHEN '收口待审' THEN 'review'
         WHEN '裁决'     THEN 'verdicted'
         WHEN '完成'     THEN 'done'
         WHEN '驳回'     THEN 'rejected'
         WHEN '挂起'     THEN 'suspended'
         WHEN '作废'     THEN 'void'
         WHEN '异常'     THEN 'error'
       END AS phase
FROM (SELECT ticket, max(ts) AS mt FROM ops_event
      WHERE ticket<>'' AND event IN
        ('起草','呈批','已批','派单','拉起','领单','开工','外派',
         '收口','移交','收口待审','审计中','裁决','完成',
         '驳回','挂起','作废','异常')
        AND ticket !~ '[[:space:],，]'
        AND ticket !~ 'T-[0-9]{4}.*T-[0-9]{4}'
      GROUP BY ticket) l
JOIN ops_event e ON e.ticket=l.ticket AND e.ts=l.mt
  AND e.event IN
    ('起草','呈批','已批','派单','拉起','领单','开工','外派',
     '收口','移交','收口待审','审计中','裁决','完成',
     '驳回','挂起','作废','异常');

-- 授权面同 fin_0147 先例（CREATE OR REPLACE 保权——条件 GRANT 幂等重列，
-- 链内自足防授权面断档）：research_rw 与 o_fin_ro 读册同授——VIEW 只读
-- 投影零写径。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
    EXECUTE 'GRANT SELECT ON ops_ticket_phase TO research_rw';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'o_fin_ro') THEN
    EXECUTE 'GRANT SELECT ON ops_ticket_phase TO o_fin_ro';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'roster_rw') THEN
    EXECUTE 'GRANT SELECT ON ops_ticket_phase TO roster_rw';
  END IF;
END $$;
