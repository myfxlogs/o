-- 0006 研究域正本面（PG-md——T-0167 分链：fin_opp_template/fin_opp_eval 正表/
-- fin_proposal 正本桥表；判据机读配置种在 fin 链 fin_0006——app_config 归 PG-fin）。
CREATE TABLE IF NOT EXISTS fin_opp_template (
    id       serial PRIMARY KEY,
    template text NOT NULL,
    venue    text NOT NULL CHECK (venue NOT LIKE '%:%'),
    params   jsonb NOT NULL DEFAULT '{}',
    enabled  boolean NOT NULL DEFAULT true
);
COMMENT ON COLUMN fin_opp_template.venue IS 'venue 域（裸值禁冒号投影）';
COMMENT ON COLUMN fin_opp_template.template IS '机会模板型（机会模板书六型——funding 先落）';

CREATE TABLE IF NOT EXISTS fin_opp_eval (
    id         bigserial PRIMARY KEY,
    ts         timestamptz NOT NULL DEFAULT now(),
    inst_id    text NOT NULL,
    template   text NOT NULL,
    direction  text NOT NULL CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot')),
    gross_apr  numeric,
    net_apr    numeric,
    verdict    text NOT NULL CHECK (verdict IN ('proposed','rejected','unevaluable')),
    legs       jsonb NOT NULL,
    evidence   jsonb NOT NULL
);

CREATE TABLE IF NOT EXISTS fin_proposal (
    id         bigserial PRIMARY KEY,
    eval_id    bigint NOT NULL REFERENCES fin_opp_eval(id),
    inst_id    text NOT NULL,
    direction  text NOT NULL,
    net_apr    numeric NOT NULL,
    state      text NOT NULL DEFAULT 'fin_proposed' CHECK (state IN ('fin_proposed','fin_armed','fin_retired','fin_rejected')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS fin_proposal_open ON fin_proposal(inst_id) WHERE state='fin_proposed';

INSERT INTO fin_opp_template(template,venue,params) VALUES
  ('funding-hedge','okx','{"legs":["spot_long","perp_short"],"settle":"8h"}')
ON CONFLICT DO NOTHING;
