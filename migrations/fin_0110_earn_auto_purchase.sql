-- 0110 T-0342 Earn 自动申购停用开关：earn.auto_purchase_enabled 键播种。
-- JSON bool 原生型（标量契约 T-0018）；播种默认 true=保持现行为；生产缺键
-- =fail-closed 停申购（worker 热读面口径——读败/缺键=停+计数门）。

INSERT INTO app_config(key,value) VALUES
  ('earn.auto_purchase_enabled', 'true')
ON CONFLICT (key) DO NOTHING;
