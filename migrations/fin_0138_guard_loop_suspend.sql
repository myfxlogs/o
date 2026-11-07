-- 0138 T-0423 守卫环自残探针——closing 滞留/attempts 阈升件+熔断挂起开仓权。
-- 病灶（pos34 实证 2026-11-07）：delta_drift 幻觉触发平仓循环，fin_guard_close
--   state='closing' attempts=5 连绞烧手续费，中途零告警——既有 close_gave_up
--   仅 attempts+1≥retry_max_attempts 才火，循环前 4 轮纯静默。探针语义钦定：
--   state='closing' 且 (attempts≥N 或 now-last_try_at>Xh) →升件+熔断。
-- 三件：
--   ① loop_alerted_at=探针行锚+节流戳（同仓已报不连拍重复刷——窗内至多一报，
--     窗=guard.loop_alert_hours 同阈复用）；
--   ② fin_guard_suspend=熔断挂起面（业主令 2026-11-07：机器发现不对但不知
--     为何照开照烧案补层——超阈不只升件，同策略开仓权挂起=只出不进；
--     lifted_at NULL=在挂，同策略至多一在挂行（部分唯一索引=幂等键）；
--     解冻唯一径=裁定席判词/业主令（tools/guardlift——lifted_by/lift_reason
--     落审计列）；
--   ③ 阈键种子 guard.loop_alert_attempts=3 / guard.loop_alert_hours=2
--     （dm 类阈键——生产调参走 tools/dmconfig，直写 SQL 禁）。
ALTER TABLE fin_guard_close ADD COLUMN IF NOT EXISTS loop_alerted_at timestamptz;
COMMENT ON COLUMN fin_guard_close.loop_alerted_at IS '守卫环自残探针最近升件时点（T-0423——节流锚：窗内同仓至多一报；NULL=未升过）';

CREATE TABLE IF NOT EXISTS fin_guard_suspend (
    id           bigserial PRIMARY KEY,
    strategy_ref bigint NOT NULL,
    position_id  bigint NOT NULL,
    reason       text NOT NULL,
    suspended_at timestamptz NOT NULL DEFAULT now(),
    lifted_at    timestamptz,
    lifted_by    text,
    lift_reason  text
);
CREATE UNIQUE INDEX IF NOT EXISTS fin_guard_suspend_active
    ON fin_guard_suspend(strategy_ref) WHERE lifted_at IS NULL;
COMMENT ON TABLE fin_guard_suspend IS '守卫熔断挂起面（T-0423）——risk.Gate open 径查 lifted_at IS NULL 行即 deny guard_suspended（平仓径不查=只许出不许进）；解冻=lifted_at/lifted_by/lift_reason 三列落位（裁定席判词或业主令唯一径）';
COMMENT ON COLUMN fin_guard_suspend.position_id IS '触发挂起的仓 id（审计锚——挂起域=strategy_ref 全策略开仓权）';

INSERT INTO app_config(key,value) VALUES
  ('guard.loop_alert_attempts','"3"'),
  ('guard.loop_alert_hours',  '"2"')
ON CONFLICT (key) DO NOTHING;
