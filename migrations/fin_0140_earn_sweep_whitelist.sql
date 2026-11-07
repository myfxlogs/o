-- 0140 T-0439 Earn 扫袋白名单化判据键播种：
--   earn.sweep_whitelist=计价币白名单（JSON 数组键——GetStringList 面，
--   首版钦定只 USDT；USDC 待议不钦定。白名单外扫袋一律禁行）。
--   earn.residual_clear_min_usd=杂币实存即清粉尘阈（USD——阈下=粉尘级
--   残留挂账待归并 fin_residual_custody 如实标；阈上走执行径市价清）。
-- 缺键=fail-closed（白名单外零扫+处置面缺阈让位——生产两键必播）。

INSERT INTO app_config(key,value) VALUES
  ('earn.sweep_whitelist',        '["USDT"]'),
  ('earn.residual_clear_min_usd', '1')
ON CONFLICT (key) DO NOTHING;
