-- fin_0200: simdrill 演习腿现金流归位 sim 域（工单#22576——D-0048 裁定①施工件，
-- 前案 机制#20819 接续）：
-- 病灶=simdrill 全产线同码只替 venue IO，execution.legCash 以 leg.Venue
-- （'okx'/'mt5' 路由真相）落 fin_cashflow.venue——演习现货腿 spot_buy/
-- spot_sell 以真 venue 名义入账，机器口径净产出五处 venue<>'sim' 谓词
-- （hunger/commercial_stall/owner_beat/selfaware/death_grad）全吞演习流水
-- （7/14/30d=-7.15/-5.71/-9.05，剔演习=-0.21/+1.56/-1.78——仓#88/#89 交易所
-- 账单 0 条被判词#22350 当真亏实证）。
-- 写侧根修=execution.Runner.WithBookVenue('sim')（simdrill 三径注入——
-- fin_cashflow/fin_slip_sample venue 列归 sim 域；fin_order_leg.venue 保腿
-- Venue 路由真相：guard 平仓腿经 facts.legs 回读须路由得回执行件）。
-- 存量归位=重标非冲销（venue 列本义=账本归属域——演习成交从未触 okx，
-- 'sim' 更贴真相；勘误冲销留双倍历史毒）。关联键=票面验收同式：
-- source_table='fin_order_leg'→fin_execution→fin_strategy.template_ref
-- LIKE 'simdrill%'（含 simdrill:cell:* 矩阵格）。幂等：venue<>'sim' 谓词
-- 重跑零行。
UPDATE fin_cashflow c SET venue='sim'
FROM fin_order_leg l
JOIN fin_execution e ON e.id=l.execution_id
JOIN fin_strategy s ON s.id=e.strategy_ref
WHERE c.source_table='fin_order_leg' AND c.source_ref=l.id
  AND s.template_ref LIKE 'simdrill%' AND c.venue<>'sim';
