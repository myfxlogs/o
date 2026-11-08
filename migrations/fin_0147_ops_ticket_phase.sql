-- 0147 GW-001 段B1 相位投影层——ops_ticket_phase VIEW。
-- 生命周期词集过滤的每票最新事件投影：说明类事件（升件/候水/送达/
-- 退班/等锁/先例适用/部署/诊断草案/议题等旁证行）一律不进相位——
-- 旧 ops_ticket_state 全事件 DISTINCT ON 被尾随说明类事件踢出队列的
-- 病灶根修面（被踢票 0443/0444/0446/0449/0452 在案）。旧 VIEW 一字
-- 不动，存量消费者（cockpit 等）零误伤。
-- 词集：正词 起草→已批→派单→拉起/领单→收口/移交/收口待审→裁决→完成；
-- 侧态 呈批/开工/外派/审计中（队列判据依赖的票史态，不进正词链
-- 但进相位——审计中/外派不载反会把在审/外派票误留队列）；
-- 负词 驳回/挂起/作废/异常。
-- 处置=说明类不进相位（实证：被踢票 T-0443/0444/0446/0449/0452 尾行
-- 均=CEO「断链处置」注记——处置的是升件告警非票本体，票仍候派；
-- 处置入相位=派单未拉起票被踢出补拉队=段B判据①钦定形「已批→派单→
-- 处置」须列 dispatched）。升件/候水/送达/退班/等锁/先例适用/部署/
-- 诊断草案/议题/领号等旁证行同例不进相位。
-- 同刻并列（ts 相同多生命周期事件）join 出多行——与 dm_queue 旧推导
-- 同 tie 语义，消费面 awk 去重兜底；phase=词→相位 token 机读面。
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
      GROUP BY ticket) l
JOIN ops_event e ON e.ticket=l.ticket AND e.ts=l.mt
  AND e.event IN
    ('起草','呈批','已批','派单','拉起','领单','开工','外派',
     '收口','移交','收口待审','审计中','裁决','完成',
     '驳回','挂起','作废','异常');

-- 授权面同 ops_ticket_state 先例（fin_0086/fin_0106）：research_rw 与
-- o_fin_ro 读册同授——VIEW 只读投影零写径。
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'research_rw') THEN
    EXECUTE 'GRANT SELECT ON ops_ticket_phase TO research_rw';
  END IF;
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'o_fin_ro') THEN
    EXECUTE 'GRANT SELECT ON ops_ticket_phase TO o_fin_ro';
  END IF;
END $$;
