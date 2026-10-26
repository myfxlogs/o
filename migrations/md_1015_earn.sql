-- 0015 Earn 停泊态 md 链（T-0167 分链：md_earn_rate 现率事实表归 PG-md——
-- D-0030 §五：采样迁 research 节点（公开读零凭据），SG earn 停泊径不读此表零跨机；
-- cashflow/reason 闭集扩+earn 配置键=钱径面走 fin 链 fin_0015）。
-- md_earn_rate=事实行族（append-only，非态表）：公开面 lending-rate-summary 采样行，
-- 读径带新鲜度门（research.earn.stale_hours）；采不到=不写行（禁假值）。
CREATE TABLE IF NOT EXISTS md_earn_rate (
    ts      timestamptz NOT NULL DEFAULT now(),
    venue   text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy     text NOT NULL,
    rate    numeric NOT NULL,
    recv_ts timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN md_earn_rate.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN md_earn_rate.rate IS '年化 APR 小数（0.035=3.5%）——源=GET /finance/savings/lending-rate-summary avgRate（24h 平均年化出借率）';
SELECT create_hypertable('md_earn_rate', 'ts', if_not_exists => TRUE);
