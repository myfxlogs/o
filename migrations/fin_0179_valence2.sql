-- fin_0179_valence2.sql —— 棒4 新升件类惧值种子:
-- 反射/reflex_stall=编译器停(教训物化中断,中位偏上);
-- 懂你/reask_denied=提示级(重复问已否件非死证,压低位防吵)。

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('反射',60,'反射编译器停=教训→闸物化中断,学习中轴件'),
  ('懂你',40,'懂你面提示件——重复问业主已否非死证,压尾防吵')
ON CONFLICT (esc_key) DO NOTHING;
