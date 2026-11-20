-- 0220 saga 流程级补偿账（工单#25807——#25771 裁定⑤ adopt 方法/reject Temporal
-- 框架；腿级 rolled_back 已有，缺流程编排账）。fin_saga=每步记账+反向补偿序；
-- steps=[{name,status(pending|done|failed|skipped|undone|undo_failed),done_at,
-- undo_fn_ref}] 最小集；ref_id=execution_id（断点恢复重建 undo 上下文用）。
-- state 词表同 fin_execution 域对齐形（active|compensating|done|failed——
-- 流程态非腿态，两层不互替）。
CREATE TABLE IF NOT EXISTS fin_saga (
    id         bigserial PRIMARY KEY,
    flow_type  text NOT NULL CHECK (flow_type IN ('open','close','rebalance')),
    state      text NOT NULL CHECK (state IN ('active','compensating','done','failed')),
    ref_id     bigint,
    steps      jsonb NOT NULL DEFAULT '[]',
    created_at timestamptz NOT NULL DEFAULT now(),
    done_at    timestamptz
);
COMMENT ON COLUMN fin_saga.ref_id IS '流程宿主行 id（execution=fin_execution.id——resume 重建 undo 上下文）';
COMMENT ON COLUMN fin_saga.steps IS 'jsonb 数组 [{name,status,done_at,undo_fn_ref}]——undo_fn_ref=代码内注册回调名（saga 只按序调，不做自动推导）';
CREATE INDEX IF NOT EXISTS fin_saga_active_idx ON fin_saga(state, created_at) WHERE state='active';
