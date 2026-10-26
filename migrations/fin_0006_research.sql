-- 0006 研究域 fin 链面（T-0167 分链：fin_opp_eval/fin_proposal 桥表在 SG 落镜像——
-- DDL 归一化断言与 md 链 md_0006 同形；fin_opp_template 正本归 PG-md 不在 fin 链；
-- 判据机读配置种=app_config 归 PG-fin）。
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

INSERT INTO app_config(key,value) VALUES
  ('research.funding.taker_fee_bps',  '5.0'),
  ('research.funding.slippage_bps',   '3.0'),
  ('research.funding.fail_cost_bps',  '2.0'),
  ('research.funding.window_loss_bps','4.0'),
  ('research.funding.min_net_apr',    '0.05'),
  ('research.funding.settles_per_day','3')
ON CONFLICT (key) DO NOTHING;
