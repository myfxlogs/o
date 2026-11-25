-- md_1109_mt5_venue_rename.sql — 工单O-39440① CFD 类目根修 md 链对偶件
-- （fin_0257 同改——双链各落；OCI PG-md 主表 venue 'mt5'→'exness'）
-- 免疫件: 全 UPDATE 带 venue='mt5' 谓词——重放零写。
UPDATE md_klines SET venue='exness' WHERE venue='mt5';
UPDATE md_ticks SET venue='exness' WHERE venue='mt5';
UPDATE md_funding SET venue='exness' WHERE venue='mt5';
