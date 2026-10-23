-- 0008 风控单入口（G6 钱径咽喉）：决策留痕表+放行令牌表；判据全 risk.* 热启。
CREATE TABLE IF NOT EXISTS fin_risk_decision (
    id          bigserial PRIMARY KEY,
    ts          timestamptz NOT NULL DEFAULT now(),
    intent      jsonb NOT NULL,
    decision    text NOT NULL CHECK (decision IN ('allow','deny')),
    reason      text NOT NULL CHECK (reason IN
      ('allow','kill_switch','missing_param','intent_incomplete','notional_cap','venue_concentration','exposure_cap','token_missing','token_expired','token_used','stale_state')),
    config_gen  bigint NOT NULL,
    token_id    uuid,
    expires_at  timestamptz
);

-- 放行令牌：allow 决策产 uuid 令牌（单次消费+时效——执行件机械防绕过）。
CREATE TABLE IF NOT EXISTS fin_risk_token (
    token_id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id bigint NOT NULL REFERENCES fin_risk_decision(id),
    intent_hash text NOT NULL,
    issued_at   timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz
);

INSERT INTO app_config(key,value) VALUES
  ('risk.kill_switch',           '"false"'),
  ('risk.max_notional_usd',      '"500"'),
  ('risk.max_venue_pct',         '"0.35"'),
  ('risk.max_exposure_usd',      '"1000"'),
  ('risk.token_ttl_seconds',     '"30"')
ON CONFLICT (key) DO NOTHING;

-- fin_strategy 补 venue 列（venue 集中度判据须按 venue 分桶；键域 CHECK+COMMENT）。
ALTER TABLE fin_strategy ADD COLUMN IF NOT EXISTS venue text NOT NULL DEFAULT 'okx';
DO $$ BEGIN
  ALTER TABLE fin_strategy ADD CONSTRAINT fin_strategy_venue_chk CHECK (venue NOT LIKE '%:%');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
COMMENT ON COLUMN fin_strategy.venue IS 'venue 域（裸值禁冒号投影）';
