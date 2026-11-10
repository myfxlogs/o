-- 0157 T-0521 滑点采样表——FinRL 镜照③摩擦回灌件②：成交腿落价当时锁
-- md_ticks mark 快照（bid/ask/last）→侧向化滑点 bps+盘口点差落账。
-- md_ticks 状态表无史可回溯——历史滑点自此采样起自产累积；mark 鲜度
-- 携 mark_ts（校准面按鲜度过滤——陈 mark 不充数）。
CREATE TABLE IF NOT EXISTS fin_slip_sample (
    id         bigserial PRIMARY KEY,
    leg_id     bigint NOT NULL,
    ts         timestamptz NOT NULL DEFAULT now(),
    venue      text NOT NULL CHECK (venue NOT LIKE '%:%'),
    inst_id    text NOT NULL,
    side       text NOT NULL,
    px         numeric NOT NULL,
    mark_px    numeric,
    mark_ts    timestamptz,
    slip_bps   numeric,
    spread_bps numeric
);
COMMENT ON TABLE fin_slip_sample IS 'T-0521 滑点实绩采样——成交腿当时 mark 锁存（自产历史起点）';
COMMENT ON COLUMN fin_slip_sample.slip_bps IS '侧向化滑点：buy=(px-mark)/mark·1e4，sell 反向——正=成本';
CREATE INDEX IF NOT EXISTS fin_slip_sample_ts_idx ON fin_slip_sample(ts);
