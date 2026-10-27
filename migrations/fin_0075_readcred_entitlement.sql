-- 0075 T-0174 readcred 泛化：entitlement 白名单（venue 集=数据非拓扑——
-- readcred 专口按 app_secrets <venue>_read.* 行族枚举，本行数组控出面）。
-- 新所上架径=app_secrets 加 <venue>_read.* 三行 + 本行数组加元素，dev 零 env 变更。
-- （立项时编号 fin_0071，与 T-0173 host_metrics 撞区——master 跳 71 让位实证沿例让至 75。）
INSERT INTO app_config(key,value,updated_by) VALUES ('readcred.entitlement','["okx"]','fin_0075') ON CONFLICT (key) DO NOTHING;
