-- 0009 双腿原子执行（G7）：fin_execution/fin_order_leg/fin_position+对冲带键。
CREATE TABLE IF NOT EXISTS fin_execution (
    id           bigserial PRIMARY KEY,
    strategy_ref bigint,
    decision_id  bigint,
    token_id     uuid,
    inst_id      text NOT NULL,
    notional_usd numeric NOT NULL,
    state        text NOT NULL CHECK (state IN ('open','legged','rolled_back','closed','failed','rollback_failed')),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS fin_order_leg (
    id           bigserial PRIMARY KEY,
    execution_id bigint NOT NULL REFERENCES fin_execution(id),
    leg_no       smallint NOT NULL,
    cl_ord_id    text NOT NULL,
    venue        text NOT NULL CHECK (venue NOT LIKE '%:%'),
    side         text NOT NULL,
    inst_id      text NOT NULL,
    sz           numeric NOT NULL,
    px           numeric,
    state        text NOT NULL CHECK (state IN ('placed','ack','filled','failed','ambiguous','rolled_back')),
    ts           timestamptz NOT NULL DEFAULT now()
);
COMMENT ON COLUMN fin_order_leg.venue IS 'venue 域（裸值禁冒号投影）';
CREATE TABLE IF NOT EXISTS fin_position (
    id           bigserial PRIMARY KEY,
    execution_id bigint NOT NULL REFERENCES fin_execution(id),
    strategy_ref bigint,
    inst_id      text NOT NULL,
    notional_usd numeric NOT NULL,
    delta_neutral boolean NOT NULL DEFAULT true,
    opened_at    timestamptz NOT NULL DEFAULT now(),
    closed_at    timestamptz
);
INSERT INTO app_config(key,value) VALUES
  ('exec.hedge_band_bps','"20"')
ON CONFLICT (key) DO NOTHING;
