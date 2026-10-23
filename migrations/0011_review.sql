-- 0011 复盘判决书+远期验证（G10 闭环收口）：fin_review/fin_forward_check/模板修正提案行。
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

-- 远期验证排期（DSHQ 五池模式：决策点后多窗复测；否决提案同排——拒对/拒错考据）。
CREATE TABLE IF NOT EXISTS fin_forward_check (
    id         bigserial PRIMARY KEY,
    obj_kind   text NOT NULL CHECK (obj_kind IN ('position','rejected_proposal')),
    obj_ref    bigint NOT NULL,
    window_h   smallint NOT NULL,
    target_ts  timestamptz NOT NULL,
    anchor_val numeric,
    actual_val numeric,
    judged     boolean NOT NULL DEFAULT false,
    judged_ok  boolean,
    judged_at  timestamptz
);

-- 模板判据修正提案行（复盘写回=提案态，投资决策面裁决后落 app_config——不自动改）。
CREATE TABLE IF NOT EXISTS fin_template_proposal (
    id        bigserial PRIMARY KEY,
    template  text NOT NULL,
    param_key text NOT NULL,
    cur_val   numeric,
    sug_val   numeric,
    evidence  jsonb NOT NULL DEFAULT '{}',
    state     text NOT NULL DEFAULT 'proposed' CHECK (state IN ('proposed','adopted','dismissed')),
    ts        timestamptz NOT NULL DEFAULT now()
);
