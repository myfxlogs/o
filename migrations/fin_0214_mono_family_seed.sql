-- 0214 多样性饥荒判据种子（工单#25649——hunger Detect 第三触发形：
-- 窗内判词集中单模板≥阈 pct=评估光谱塌缩；zero_proposal/paper_empty
-- 两旧判只见「有没有」不见「是不是只有一种」——dir_perp_mom 97.9%
-- 实证：369 符 MT5 流三周零 venue=mt5 判词而 hunger 判「饱」）。
-- dm.hunger.* 域键机器自调域在册（dmwrite.go:44 前缀白名单——
-- 阈调走 DMSet 域闸+参数调整行）。
INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.mono_family_days', '7', 'fin_0214'),
  ('dm.hunger.mono_family_pct', '85', 'fin_0214')
ON CONFLICT (key) DO NOTHING;
