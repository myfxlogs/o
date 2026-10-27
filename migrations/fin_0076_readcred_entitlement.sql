-- 0076 T-0174 readcred 泛化：entitlement 白名单（venue 集=数据非拓扑——
-- readcred 专口按 app_secrets <venue>_read.* 行族枚举，本行数组控出面）。
-- 新所上架径=app_secrets 加 <venue>_read.* 三行 + 本行数组加元素，dev 零 env 变更。
-- （立项时编号 fin_0071——两度撞区让位：T-0173 host_metrics 占 71、T-0185 cashflow 占 75，终落 76。）
INSERT INTO app_config(key,value,updated_by) VALUES ('readcred.entitlement','["okx"]','fin_0076') ON CONFLICT (key) DO NOTHING;
