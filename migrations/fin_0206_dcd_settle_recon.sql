-- 0206 dcd 结算对账（工单#22919/20904 钦定修法①③）：
-- ①fin_execution.venue_ord_id——venue ordId 持久化确证键（结算对账不靠
--   productId+cTime 模糊配；dcdexec 开仓末拍落值，旧仓由对账拍回填）；
-- ②fin_position_view 重落——dcd_baseline 凭据支加结算退役排除：同 ordId
--   已落 op='dcd_settle' 凭据者不再聚为持仓（结算后 venue LIVE 集已出、
--   账本不退役=幻影持仓+对拍真差永挂；凭据行本身不动=append-only 律）。

ALTER TABLE fin_execution ADD COLUMN IF NOT EXISTS venue_ord_id text;
COMMENT ON COLUMN fin_execution.venue_ord_id IS 'venue 订单号（dcd ordId 族——结算对账确证键；NULL=ordId 持久化前旧仓，靠 productId+cTime 模糊配过渡）';

CREATE OR REPLACE VIEW fin_position_view AS
SELECT 'exec'::text    AS kind,
       p.id            AS position_id,
       p.execution_id  AS execution_id,
       p.strategy_ref  AS strategy_ref,
       p.inst_id       AS inst_id,
       p.notional_usd  AS notional,
       p.delta_neutral AS delta_neutral,
       p.opened_at     AS ts,
       p.closed_at     AS closed_at,
       NULL::text      AS venue,
       NULL::text      AS ccy
FROM fin_position p
UNION ALL
SELECT CASE WHEN d.dom IS NULL THEN 'dcd'::text
            ELSE 'dcd_'||d.dom END AS kind,
       NULL::bigint    AS position_id,
       NULL::bigint    AS execution_id,
       NULL::bigint    AS strategy_ref,
       NULL::text      AS inst_id,
       d.amt           AS notional,
       NULL::boolean   AS delta_neutral,
       d.last_ts       AS ts,
       NULL::timestamptz AS closed_at,
       d.venue         AS venue,
       d.ccy           AS ccy
FROM (
    SELECT r.intent->>'venue' AS venue, r.intent->>'ccy' AS ccy,
           sum((r.intent->>'amt')::numeric) AS amt, max(r.ts) AS last_ts,
           dom.domain AS dom
    FROM fin_risk_decision r
    LEFT JOIN fin_position_domain dom ON dom.decision_id=r.id
    WHERE r.intent->>'op'='dcd_baseline' AND r.decision='allow'
      AND NOT EXISTS (SELECT 1 FROM fin_risk_decision s
                      WHERE s.intent->>'op'='dcd_settle'
                        AND s.intent->>'ordId'=r.intent->>'ordId')
    GROUP BY 1, 2, dom.domain
) d;

COMMENT ON VIEW fin_position_view IS '统一持仓读面（T-0351）——执行仓+产品仓双账合一；kind=exec|dcd|dcd_owner，dcd 支已结算凭据退役（dcd_settle 同 ordId 排除，工单#22919）';
