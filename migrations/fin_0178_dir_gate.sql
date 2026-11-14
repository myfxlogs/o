-- 0178 方向书子预算判据键播种（棒3——dir_perp_mom 真户前置钱径件）：
-- risk.dir.max_notional_usd=方向书名义帽（open 持仓+在途 exec 名义聚合,
-- 闸内口径同总敞口读面）；risk.dir.loss_budget_usd=亏完断粮阈（书内
-- fin_position×fin_cashflow 已实现净亏破阈=方向书全拒新开——亏损预算
-- 是账不是建议,close 径不拦退仓权保留）。
-- 种子 0=方向书零预算——真单方向意图全拒（dir_notional_cap 先触）:
-- 毕业=业主供数非系统自批（owner 写域,本金线内子账额度裁量归业主
-- ——与 risk.max_notional_usd 同类）。棒5 小额真户前业主拍数即活。

INSERT INTO app_config(key,value) VALUES
	('risk.dir.max_notional_usd','0'),
	('risk.dir.loss_budget_usd','0')
ON CONFLICT(key) DO NOTHING;
