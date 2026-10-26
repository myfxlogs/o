-- 0020 宏观日历配置键 fin 链（T-0029；T-0167 分链：app_config 归 PG-fin；
-- md_macro_event 表正本走 md 链 md_0020、SG 镜像走 fin_0067——第八桥表）。
INSERT INTO app_config(key,value) VALUES
  ('research.macro.horizon_days','"2"')
ON CONFLICT (key) DO NOTHING;
