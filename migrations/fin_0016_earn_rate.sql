-- 0015 Earn 采拍/陈阈配置键 fin 链（T-0023；T-0167 分链：app_config 归 PG-fin，
-- md_earn_rate 表走 md 链 md_0016）。
INSERT INTO app_config(key,value) VALUES
  ('data.earn.poll_sec','"300"'),
  ('research.earn.stale_hours','"24"')
ON CONFLICT (key) DO NOTHING;
