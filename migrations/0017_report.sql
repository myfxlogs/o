-- 0017 经营报表（单源机读交卷件——数字全出 fin_cashflow/position/strategy/md_earn_rate 在册行；
-- period+period_start 唯一=幂等重跑 UPDATE 非插重行）。
CREATE TABLE IF NOT EXISTS fin_report (
    id           bigserial PRIMARY KEY,
    period       text NOT NULL CHECK (period IN ('daily','weekly','monthly')),
    period_start timestamptz NOT NULL,
    period_end   timestamptz NOT NULL,
    metrics      jsonb NOT NULL DEFAULT '{}',  -- 指标体：net/cum_nav/opened/closed/evals/alerts/earn…
    note         text NOT NULL DEFAULT '',     -- 缺数据诚实注记+月度三行交代
    generated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (period, period_start)
);
COMMENT ON TABLE fin_report IS '经营报表单源件——数字禁账本外口径（司账同铁律）';
