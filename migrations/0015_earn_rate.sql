-- 0015 Earn 现率事实表+采拍/陈阈键（T-0023 动态比较底——零资金移动读面半）。
CREATE TABLE IF NOT EXISTS md_earn_rate (
    ts    timestamptz NOT NULL DEFAULT now(),
    venue text NOT NULL DEFAULT 'okx' CHECK (venue NOT LIKE '%:%'),
    ccy   text NOT NULL,
    rate  numeric NOT NULL
);
COMMENT ON COLUMN md_earn_rate.venue IS 'venue 域（裸值禁冒号投影）';
SELECT create_hypertable('md_earn_rate', 'ts', if_not_exists => TRUE);
INSERT INTO app_config(key,value) VALUES
  ('data.earn.poll_sec','"300"'),
  ('research.earn.stale_hours','"24"')
ON CONFLICT (key) DO NOTHING;
