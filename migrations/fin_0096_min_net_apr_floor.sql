-- 0096 T-0297 min_net_apr 阈归位 0.20→0.06（D-0043/CTO 会签：评估器费用
-- 年化摊销根修后，0.20 是「评估器不摊销时的手工折算阈」——口径归位非
-- 放松；Earn 动态底 floor=max(静态,Earn) 不变）。DO UPDATE 显式径：
-- 旧 seed DO NOTHING 不会覆盖 SG 现值 0.20，须 UPDATE 语义。
INSERT INTO app_config(key,value) VALUES
  ('research.funding.min_net_apr','0.06')
ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value;
