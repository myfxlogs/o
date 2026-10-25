-- 0048 T-0110 TRX 整品迁移补件：
--   ① position_daily_income.day 唯一约束（nox 1000 基线同款——日终 rollup
--     ON CONFLICT (day) 幂等锚；缺=auto 重复落行假增）。
--   ② trxvote 旗歇/判据 config 种（引擎停驶默认；翻 true=业主显式授权点）。
ALTER TABLE position_daily_income DROP CONSTRAINT IF EXISTS position_daily_income_day_key;
CREATE UNIQUE INDEX IF NOT EXISTS position_daily_income_day_key
  ON position_daily_income (day);

INSERT INTO app_config(key,value) VALUES
  ('trx.vote_enabled',          'false'),
  ('trx.vote_min_srs',          '3'),
  ('trx.vote_max_srs',          '3'),
  ('trx.vote_max_brokerage',    '20'),
  ('trx.vote_yield_drop_pct',   '15.0'),
  ('trx.vote_better_opp_pct',   '0.0'),
  ('trx.vote_max_rebal_per_day','1'),
  ('trx.vote_min_interval_min', '120'),
  ('trx.vote_spread_threshold', '0.001'),
  ('trx.floor_pct',             '8.0'),
  ('trx.grpc_endpoint',         '"grpc.trongrid.io:50051"')
ON CONFLICT (key) DO NOTHING;
