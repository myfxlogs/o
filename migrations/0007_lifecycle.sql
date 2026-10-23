-- 0007 生命周期状态机+paper 沙箱（G3→G5；判据全 invest.lifecycle.* 热启）。
CREATE TABLE IF NOT EXISTS fin_strategy (
    id           bigserial PRIMARY KEY,
    inst_id      text NOT NULL,
    template_ref text NOT NULL,
    proposal_ref bigint,
    state        text NOT NULL CHECK (state IN ('proposed','paper','probe','live','degraded','retired')),
    notional_usd numeric NOT NULL,
    entered_at   timestamptz NOT NULL DEFAULT now(),
    criteria     jsonb NOT NULL DEFAULT '{}',
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS fin_strategy_state ON fin_strategy(state);

CREATE TABLE IF NOT EXISTS fin_strategy_transition (
    id         bigserial PRIMARY KEY,
    strategy_id bigint NOT NULL REFERENCES fin_strategy(id),
    from_state text NOT NULL,
    to_state   text NOT NULL,
    reason     text NOT NULL,
    ts         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS fin_paper_fill (
    id          bigserial PRIMARY KEY,
    strategy_id bigint NOT NULL REFERENCES fin_strategy(id),
    kind        text NOT NULL CHECK (kind IN ('open','funding','fee','close')),
    inst_id     text NOT NULL,
    amount_usd  numeric NOT NULL,
    detail      jsonb NOT NULL DEFAULT '{}',
    ts          timestamptz NOT NULL DEFAULT now()
);

-- 晋升判据+paper 参数（缺任一键=不晋升 fail-closed；live_switch=投资委员会开关位——
-- G4 第一笔真钱须此键显式 true，缺键/非 true=禁 live 迁移）。
INSERT INTO app_config(key,value) VALUES
  ('invest.paper.notional_usd',      '"100"'),
  ('invest.lifecycle.paper_min_hours','"24"'),
  ('invest.lifecycle.paper_min_net_usd','"0.01"'),
  ('invest.lifecycle.paper_max_dd_usd','"5"'),
  ('invest.lifecycle.live_switch',    '"false"')
ON CONFLICT (key) DO NOTHING;
