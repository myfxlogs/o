-- 0192 开仓断流探针窗种子（dm.hunger.open_drought_days——hunger 同族键域：
-- 断流机读定义窗口非策略阈；初值 3d 对齐 zero_proposal_days 票面钦例）。
INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.open_drought_days', '3', 'fin_0192')
ON CONFLICT (key) DO NOTHING;
