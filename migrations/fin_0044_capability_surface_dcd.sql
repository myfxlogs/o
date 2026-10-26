-- 0043 T-0102 判据机读配置种 fin 链（T-0167 分链：app_config 归 PG-fin；
-- 能力面/DCD 目录/面差队列 DDL 走 md 链 md_0044；fin_opp_eval direction 扩=
-- 桥表 DDL 与 md 链同形）。
CREATE TABLE IF NOT EXISTS md_dcd_product (
    ts               timestamptz NOT NULL,
    venue            text NOT NULL CHECK (venue NOT LIKE '%:%'),
    surface          text NOT NULL,
    uly              text NOT NULL,
    product_id       text NOT NULL,
    direction        text NOT NULL CHECK (direction IN ('C','P')),
    strike           numeric,
    expiry           timestamptz,
    annualized_yield numeric,
    min_size         numeric,
    max_size         numeric,
    spot             numeric,
    raw              jsonb NOT NULL DEFAULT '{}'
);
DO $$ BEGIN IF to_regclass('md_dcd_product') IS NULL THEN PERFORM create_hypertable('md_dcd_product', 'ts', if_not_exists => TRUE); END IF; END $$;
CREATE INDEX IF NOT EXISTS md_dcd_product_pid ON md_dcd_product(product_id, ts DESC);

ALTER TABLE fin_opp_eval DROP CONSTRAINT IF EXISTS fin_opp_eval_direction_check;
ALTER TABLE fin_opp_eval ADD CONSTRAINT fin_opp_eval_direction_check
  CHECK (direction IN ('short_perp_long_spot','long_perp_short_spot','dcd_call','dcd_put',
                       'gap_carry','gap_hedge'));

INSERT INTO app_config(key,value) VALUES
  ('research.dcd.min_dist_pct',  '8'),
  ('research.dcd.max_term_days', '30'),
  ('research.dcd.min_apr',       '0.05'),
  ('research.dcd.notional_pct',  '20'),
  ('research.gap.scan_interval_min', '360'),
  ('research.gap.gen_cap',       '3')
ON CONFLICT (key) DO NOTHING;
