-- 0044 T-0106 判据机读 config 种 fin 链（T-0167 分链：app_config 归 PG-fin；
-- 假想卡/面差链 DDL 走 md 链 md_0045）。
INSERT INTO app_config(key,value) VALUES
  ('research.hypothesis.min_products', '1'),
  ('research.hypothesis.families',     '"carry,hedge"')
ON CONFLICT (key) DO NOTHING;
