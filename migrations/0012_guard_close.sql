-- 0012 看护平仓（G8）：risk 判径 kind 列 + 平仓态子面表 + guard 热启键。
-- 平仓态机落自有 fin_guard_close（position_id 逻辑关联不设 FK——fin_position 归
-- T-0014/0009 所有，本票零 schema 依赖、合并序无关）；position 侧只写契约内
-- closed_at 列。close intent 与 open 同落 fin_risk_decision 留痕，kind 区分判径。
ALTER TABLE fin_risk_decision ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'open';
DO $$ BEGIN
  ALTER TABLE fin_risk_decision ADD CONSTRAINT fin_risk_decision_kind_chk
    CHECK (kind IN ('open','close'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_risk_decision.kind IS '判径 kind：open=开仓径全闸；close=平仓径（kill_switch/上限不拦——敞口只减不增，拦平仓=存风险）';

-- 平仓态子面（行缺省=在仓 open；closing=intent 在途；closed=已平；close_failed=败待重试）。
CREATE TABLE IF NOT EXISTS fin_guard_close (
    position_id  bigint PRIMARY KEY,
    state        text NOT NULL CHECK (state IN ('closing','closed','close_failed')),
    trigger_kind text NOT NULL,
    attempts     int NOT NULL DEFAULT 0,
    last_try_at  timestamptz,
    closed_at    timestamptz,
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- 判据/节拍键（缺任一键=对应触发面 fail-closed 跳过+计数，哨戒循环不炸）。
-- exec.hedge_band_bps 与 T-0014/0009 双落保险：先到者种键，后到 DO NOTHING。
INSERT INTO app_config(key,value) VALUES
  ('guard.flip_bps',             '"5"'),
  ('guard.max_hold_hours',       '"72"'),
  ('guard.retry_max_attempts',   '"5"'),
  ('guard.retry_backoff_seconds','"60"'),
  ('exec.hedge_band_bps',        '"20"')
ON CONFLICT (key) DO NOTHING;
