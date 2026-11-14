-- fin_0180_dir_pct_gate.sql —— 方向书预算改百分比×venue权益口径（棒5——
-- 业主语义「OKX 余额的百分数」非定值美元,余额动则定值漂移;阀值=系统
-- 自推基线,owner 写域业主可覆）:
--   loss_budget_pct=累计净亏断粮阈(0.10——业主锚:方向能力证伪=永久停);
--   trade_loss_pct=单笔止损亏损帽(0.005——自推:断粮÷20笔样本,业主原
--     议 3% 只够 ~3 笔连亏=实验自产证据前先死,样本不足=验证不能完成);
--   max_notional_pct=书名义帽(0.20——≈6 并发仓,簇止损最坏亏=0.20×
--     0.15=3%<断粮线);
--   stop_frac=lifecycle.dir.stop_frac 闸消费(与 dirpaper 出场同一真源,
--     原未播种——闸读它判单笔最坏亏,显式种=免双侧缺省漂移);
--   research.dirmom.stale_h=末确认棒陈界(56h——1D 棒=确认滞~16h+日
--     cadence 24h→常龄≤40h,原 36h 每日 04:00→16:00 UTC 处死窗零产出;
--     56h 顶=真断供仍捕）。
-- 旧定值键退役(usd 种子 0 从未供数——语义被 _pct 全替)。

INSERT INTO app_config(key,value) VALUES
	('risk.dir.loss_budget_pct','0.10'),
	('risk.dir.trade_loss_pct','0.005'),
	('risk.dir.max_notional_pct','0.20'),
	('lifecycle.dir.stop_frac','0.15'),
	('research.dirmom.stale_h','56')
ON CONFLICT(key) DO NOTHING;
DELETE FROM app_config WHERE key IN ('risk.dir.max_notional_usd','risk.dir.loss_budget_usd');
