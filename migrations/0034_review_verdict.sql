-- 0034 判决书暂存迁库（T-0066：falsified 判决=策略演化输入——权威落库非容器文件寄生；
-- 待归纳.md 降级为派生镜像 best-effort——写败只 log 一次/进程不刷屏）。
CREATE TABLE IF NOT EXISTS fin_review_verdict (
    id            bigserial PRIMARY KEY,
    position_ref  bigint NOT NULL,
    verdict       text NOT NULL,
    pred_apr      numeric,
    actual_pnl    numeric,
    attribution   text,
    created_at    timestamptz NOT NULL DEFAULT now()
);
-- 同仓一判幂等（判决书=一次判决一存——重拍防重复）。
CREATE UNIQUE INDEX IF NOT EXISTS fin_review_verdict_pos_uq ON fin_review_verdict(position_ref);
