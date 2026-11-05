-- 0112 统一持仓读面（T-0351 钱径审计缺陷①——「现在有什么仓」双账合一）：
-- 执行仓（fin_position 对冲对全行）+产品仓（dcd_baseline 凭据聚合）UNION 单读面。
-- 视图=纯读派生对象不落数据表；凭据写径唯一=风险决策件（本视图零 mutating）。
CREATE OR REPLACE VIEW fin_position_view AS
SELECT 'exec'::text    AS kind,       -- 执行仓族=fin_position 全行（对冲对——双腿 venue 各异，按 execution_id 对 fin_order_leg）
       p.id            AS position_id,
       p.execution_id  AS execution_id,
       p.strategy_ref  AS strategy_ref,
       p.inst_id       AS inst_id,
       p.notional_usd  AS notional,   -- exec 口径=对冲对名义 USD
       p.delta_neutral AS delta_neutral,
       p.opened_at     AS ts,         -- exec 口径=仓开 ts
       p.closed_at     AS closed_at,
       NULL::text      AS venue,
       NULL::text      AS ccy
FROM fin_position p
UNION ALL
SELECT 'dcd'::text    AS kind,       -- 产品仓族=dcd_baseline 凭据按 (venue,ccy) 聚合——只认 allow（deny 行=未执行拒单，不入持仓）
       NULL::bigint    AS position_id,
       NULL::bigint    AS execution_id,
       NULL::bigint    AS strategy_ref,
       NULL::text      AS inst_id,
       d.amt           AS notional,   -- dcd 口径=凭据 Σamt 原生币量（≈live 单名义，venue 实持对拍锚）
       NULL::boolean   AS delta_neutral,
       d.last_ts       AS ts,         -- dcd 口径=最近凭据 ts
       NULL::timestamptz AS closed_at,
       d.venue         AS venue,
       d.ccy           AS ccy
FROM (
    SELECT intent->>'venue' AS venue, intent->>'ccy' AS ccy,
           sum((intent->>'amt')::numeric) AS amt, max(ts) AS last_ts
    FROM fin_risk_decision
    WHERE intent->>'op'='dcd_baseline' AND decision='allow'
    GROUP BY 1, 2
) d;

COMMENT ON VIEW fin_position_view IS '统一持仓读面（T-0351）——执行仓+产品仓双账合一；kind=exec|dcd，任何持仓问询单面全口径';
COMMENT ON COLUMN fin_position_view.notional IS '量纲随 kind：exec=对冲对名义 USD；dcd=凭据 Σamt 原生币量';
COMMENT ON COLUMN fin_position_view.ts IS '量纲随 kind：exec=仓开 ts；dcd=最近凭据 ts';
COMMENT ON COLUMN fin_position_view.venue IS 'venue 域（裸值禁冒号投影）；exec 行=NULL（对冲对跨腿 venue 见 fin_order_leg）';
COMMENT ON COLUMN fin_position_view.ccy IS 'dcd 行=凭据币种；exec 行=NULL';

-- 窄读面授权（o_fin_ro 先例同径 fin_0106/0111——持仓问询席=只读）
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='o_fin_ro') THEN
    EXECUTE 'GRANT SELECT ON fin_position_view TO o_fin_ro';
  END IF;
END $$;
