-- 0253 族卡断流探针键+惧值固化（工单O-39330——「族卡→评估臂」无主
-- 转换感知器：candidate 族卡归一名下零产出超龄→breed_stall 升件）：
-- ① dm.hunger.breed_stall_days——族卡断流窗（初值 2d——三张研究员卡
--   11-23 落册现龄恰触发=首个真实信号即实证；探针 fail-closed 键缺
--   不探，落键防哑针是本件验收项）；
-- ② ops_esc_valence upsert——breed_stall 新类 65（管道育种断档——
--   低于 opp_eval_stall 85、高于常规噪音）。

INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.breed_stall_days', '2', 'fin_0253')
ON CONFLICT (key) DO NOTHING;

INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('breed_stall',65,'管道育种断档——注册族卡零产出无主转换（工单O-39330）')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;
