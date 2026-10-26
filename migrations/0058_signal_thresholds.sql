-- 0058 T-0140 信号臂阈值种（app_config 热启——信号驱动假想生成判据机读）：
--   funding_div_ratio：funding_view 现值 vs avg_7d 背离倍数阈（regime 翻转形）；
--   spread_bps：depth5 spread_bps 流动性突变阈。
INSERT INTO app_config(key,value) VALUES
  ('research.signal.funding_div_ratio', '3'),
  ('research.signal.spread_bps',        '50')
ON CONFLICT (key) DO NOTHING;
