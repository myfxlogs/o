-- 0063 fin_data_retention 表级保留账 md 链（T-0153；T-0167 分链）：A 类热表
-- autovacuum 表级参数=桥表 DDL 与 fin 链 fin_0063 同形（DDL 归一化断言钉）；
-- 保留账本体（fin_data_retention 表+入册行+清道夫）=钱径面归 fin 链——
-- PG-md 可重建面（D-0030 §八：备份降级为 schema+配置，清道夫只在 SG）。
ALTER TABLE fin_opp_eval SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
ALTER TABLE md_mark_index SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
ALTER TABLE md_ticks SET (autovacuum_vacuum_scale_factor=0.02, autovacuum_vacuum_cost_delay=2);
