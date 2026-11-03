-- 1081 T-0300 venue scout 公开面采集表（PG-md 事实行族 append-only）——
-- 每 venue 每结算周期一拍全 USDT 永续 funding/标记/指数（度量用不着逐 tick）。
CREATE TABLE IF NOT EXISTS md_venue_funding (
    ts           timestamptz NOT NULL,
    venue        text NOT NULL CHECK (venue NOT LIKE '%:%'),
    inst         text NOT NULL,
    funding_rate numeric,
    next_funding numeric,
    interval_h   numeric,
    mark         numeric,
    index        numeric
);
SELECT create_hypertable('md_venue_funding', 'ts', if_not_exists => TRUE);
COMMENT ON TABLE md_venue_funding IS
  'venue scout 公开 funding 事实行（T-0300）——零凭据公开端点采集；inst 键=venue 原生符号';
CREATE INDEX IF NOT EXISTS md_venue_funding_vi ON md_venue_funding(venue,inst,ts DESC);
SELECT add_retention_policy('md_venue_funding', interval '90 days');
INSERT INTO fin_data_retention(table_name,class,keep_interval,note) VALUES
  ('md_venue_funding','audit_keep','90 days','T-0300 venue scout 采集——30d 滚动度量+余量')
ON CONFLICT (table_name) DO NOTHING;
