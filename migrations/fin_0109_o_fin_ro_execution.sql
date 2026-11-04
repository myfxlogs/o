-- fin_0109 T-0332 返修①：o_fin_ro 白名单补授 fin_execution
-- gauges 量表 arm_zero_output（断供类）取数径 sql_ro=o_fin_ro；0106 白名单
-- 漏 fin_execution 致量表永死（决策席驳回件：两死表取数径零权）。
-- 授权先例同 fin_0108（condition_gap 补授）；执行事实行属只读事实面，
-- 无凭据物料列，白名单补授不扩写面。
GRANT SELECT ON fin_execution TO o_fin_ro;
