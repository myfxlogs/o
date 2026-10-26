-- 0011 复盘判决书 fin 链面（T-0167 分链：fin_review=钱径账链归 PG-fin；
-- fin_forward_check/fin_template_proposal 研究域走 md 链 md_0011）。
CREATE TABLE IF NOT EXISTS fin_review (
    id            bigserial PRIMARY KEY,
    position_ref  bigint NOT NULL,
    verdict       text NOT NULL CHECK (verdict IN ('strong','mild','marginal','falsified')),
    pred_net      numeric,
    actual_net    numeric,
    deviation     numeric,
    attribution   text CHECK (attribution IN ('fee_drift','funding_flip','slippage','model_err','luck','na')),
    detail        jsonb NOT NULL DEFAULT '{}',
    reviewed_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (position_ref)
);
COMMENT ON COLUMN fin_review.verdict IS '四值闭集分级——falsified=经济经验库回改点挂钩';
