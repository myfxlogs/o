-- 0116 T-0376 饥饿响应自参数种子（票面钦定 N=3 天——检测窗非策略阈：
-- 「不替机器拍初值」红线不触——本键=饥饿态机读定义自身窗口，票面明文
-- 「票面定 N，建议 3 天」）。dm 写域分册 dm 类（dm.hunger.*）——机器可
-- 自调，自调必落 ops_event 参数调整行（dmwrite.go 律）。
INSERT INTO app_config(key,value,updated_by) VALUES
  ('dm.hunger.zero_proposal_days', '3', 'fin_0116')
ON CONFLICT (key) DO NOTHING;
