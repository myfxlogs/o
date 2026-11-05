-- 0115 T-0364：dcd 遗留仓收归业主域（业主令 2026-11-05——ordId 11448262
--   BTC-USDT-261225-95000-C 0.0207 BTC 收归独立仓，策略敞口账剔出；纯账面
--   语义 venue 仓零动、risk.* 键不动、其余 dcd 凭据零动）。
-- 三件：①凭据归属域册 fin_position_domain——fin_risk_decision 行→持仓域
--   （decision_id 软锚无 FK，同 fin_cashflow.source_ref 惯例：凭据行不可变、
--   测试径清场不炸），业主行按 ordId 单定位断言落册；②fin_position_view
--   dcd 支按域分流 kind='dcd_owner'——cockpit 敞口 SQL 与闸面 dcdExposureUSD
--   kind='dcd' 谓词同源剔出（显示面最简隐：敞口分子剔除，在账可查靠视图
--   kind 证据行）；③cashflow kind 闭集扩 owner_in/owner_out——12-25 结算
--   回款归因=业主资产收/支非策略线（class=transfer 同「内部移动非盈利」
--   口径；venue 共户托管，net 口径照计入=venue 总权益对拍不破）。
CREATE TABLE IF NOT EXISTS fin_position_domain (
    decision_id bigint PRIMARY KEY,   -- fin_risk_decision 凭据软锚（无 FK——source_ref 同径惯例）
    domain      text NOT NULL CHECK (domain IN ('owner')), -- 归属域闭集（无行=strategy 缺省；新域扩 CHECK 走迁移）
    inst_id     text NOT NULL DEFAULT '', -- venue 产品单号（结算归因主链键）
    ord_id      text NOT NULL DEFAULT '', -- venue 订单号（结算归因备链键——账单 instId 缺位方言兜底）
    note        text NOT NULL DEFAULT '',
    marked_at   timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_position_domain IS '凭据归属域册（T-0364）——fin_risk_decision 凭据行→持仓域；无行=strategy 缺省，domain=owner=业主独立仓（策略账剔出，资产托管不变）';
COMMENT ON COLUMN fin_position_domain.decision_id IS 'fin_risk_decision.id 软锚（凭据行不可变——无 FK 同 source_ref 惯例）';

DO $$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n FROM fin_risk_decision
     WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='11448262';
    IF n <> 1 THEN
        RAISE EXCEPTION 'T-0364 expects exactly one dcd_baseline credential ordId=11448262; got %', n;
    END IF;
END $$;

INSERT INTO fin_position_domain(decision_id,domain,inst_id,ord_id,note)
SELECT id,'owner','BTC-USDT-261225-95000-C','11448262',
       'T-0364 业主令 2026-11-05：dcd 遗留仓收归业主独立仓——策略账剔出纯账面语义，venue 仓零动'
FROM fin_risk_decision
WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'='11448262'
ON CONFLICT (decision_id) DO NOTHING;

-- 结算归因闭集扩：owner_in=业主资产流入（回款/收益/退款），owner_out=流出
-- （申购扣款）——transfer 口径非策略线；earn_* 排除面不含此二键=venue 侧
-- 共户托管现金照入 net（挪出=对账真差）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check CHECK (kind IN
  ('fee','funding_settle','trade_pnl','rollback_cost','deposit','withdraw',
   'earn_in','earn_out','interest','recon_adjust','spot_buy','spot_sell',
   'owner_in','owner_out'));

INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
  ('owner_in','transfer','业主资产流入——业主独立仓结算回款非策略线（T-0364）'),
  ('owner_out','transfer','业主资产流出——业主域申购扣款同内部移动（T-0364）')
ON CONFLICT (kind) DO NOTHING;

-- 视图域分流重落（列单同形——kind 值域扩 exec|dcd|dcd_owner，谓词消费面
-- kind='dcd' 天然只取策略域）。
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
SELECT CASE WHEN d.dom IS NULL THEN 'dcd'::text
            ELSE 'dcd_'||d.dom END AS kind, -- 产品仓族=dcd_baseline 凭据按 (venue,ccy,域) 聚合——T-0364 域册分流（owner→dcd_owner 业主独立仓；只认 allow）
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
    SELECT r.intent->>'venue' AS venue, r.intent->>'ccy' AS ccy,
           sum((r.intent->>'amt')::numeric) AS amt, max(r.ts) AS last_ts,
           dom.domain AS dom
    FROM fin_risk_decision r
    LEFT JOIN fin_position_domain dom ON dom.decision_id=r.id
    WHERE r.intent->>'op'='dcd_baseline' AND r.decision='allow'
    GROUP BY 1, 2, dom.domain
) d;

COMMENT ON VIEW fin_position_view IS '统一持仓读面（T-0351）——执行仓+产品仓双账合一；kind=exec|dcd|dcd_owner（T-0364 业主域凭据分流——策略敞口两消费面谓词 kind=''dcd'' 同源剔出），任何持仓问询单面全口径';
COMMENT ON COLUMN fin_position_view.notional IS '量纲随 kind：exec=对冲对名义 USD；dcd/dcd_owner=凭据 Σamt 原生币量';
COMMENT ON COLUMN fin_position_view.ts IS '量纲随 kind：exec=仓开 ts；dcd/dcd_owner=最近凭据 ts';
COMMENT ON COLUMN fin_position_view.venue IS 'venue 域（裸值禁冒号投影）；exec 行=NULL（对冲对跨腿 venue 见 fin_order_leg）';
COMMENT ON COLUMN fin_position_view.ccy IS 'dcd/dcd_owner 行=凭据币种；exec 行=NULL';

INSERT INTO fin_data_retention(table_name,class,note)
VALUES('fin_position_domain','permanent','凭据归属域册——账面语义永久')
ON CONFLICT (table_name) DO UPDATE SET class='permanent';

-- 窄读面授权（o_fin_ro 先例同径 fin_0106/0112——持仓问询席=只读）
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='o_fin_ro') THEN
    EXECUTE 'GRANT SELECT ON fin_position_domain TO o_fin_ro';
  END IF;
END $$;
