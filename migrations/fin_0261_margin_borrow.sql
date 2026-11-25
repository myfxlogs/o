-- fin_0261 margin-short 借币证据面（O-39460）。
-- md_margin_borrow：OKX 保证金借币率/额度快照——采集面=okxexec InterestRates+
-- MaxLoan 定拍写入（采集器接线随能力面落地；表先行=评估侧证据契约位）。
-- borrow_apr=年化小数（小时利率×8760 换算在采集侧完成——消费面禁换算歧义）。
CREATE TABLE IF NOT EXISTS md_margin_borrow (
    ccy            text        NOT NULL,
    inst_id        text        NOT NULL,           -- MARGIN/现货对（做空腿标的）
    mgn_mode       text        NOT NULL,           -- cross|isolated
    borrow_apr     double precision NOT NULL,      -- 年化借币成本
    max_loan       double precision NOT NULL,      -- 卖侧可借量（币单位）
    ts             timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (ccy, inst_id, mgn_mode)
);
CREATE INDEX IF NOT EXISTS md_margin_borrow_ts ON md_margin_borrow (ts DESC);

-- 闸键（app_config 热启——缺=fail-closed 不评估不放行）：
-- funding.margin_borrow_stale_hours  证据新鲜度窗（缺=不评估）
-- funding.margin_borrow_max_apr      借币息尖峰闸——超阈=carry 被息吞拒
-- funding.margin_min_loan_usd        可借额度下限（USD 名义——额度尘级=不可执行）
INSERT INTO app_config(key,value) VALUES
  ('funding.margin_borrow_stale_hours','6'),
  ('funding.margin_borrow_max_apr','0.5'),
  ('funding.margin_min_loan_usd','500')
ON CONFLICT (key) DO NOTHING;
