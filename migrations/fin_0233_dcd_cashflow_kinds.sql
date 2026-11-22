-- fin_0233 dcd 簿记机械环 kind 扩集（工单O-33362——T-0101 凭据配对模型
-- 根修：dcd 申购/结算现金腿入 counted 域）。
--   dcd_out=双币赢申购扣款（负——自有资金流入结构产品=内部移动 transfer）；
--   dcd_in =双币赢结算回款本金腿（正——产品到期回款=内部移动 transfer）。
--   收益腿不另设 kind——yieldSz 可分拆时按既有 'interest'(income) 记，
--   不可分拆并入 dcd_in 实收额（本金+息一体合法——现金域实收为准）。
--   两键皆 counted（earn_in/out 排除面外）——对账公式
--   net=cashflow+parked+dcdBase+mtm 下与 dcd_baseline 凭据配对互抵：
--   申购=-名义(dcd_out)+基线(dcdBase+N) 净零变；结算=+实收(dcd_in/interest)
--   -基线退役(dcdBase-N) 净=venue 真实现金变动。
--   recon_adjust 不取——其契约=基线校准一次性行「禁日常使用」，dcd 流
--   为常态 recurring，语义域不混（#33341 手工补丁同型仅历史留痕）。
-- 幂等机械闸：dcd_baseline/dcd_settle 凭据 (op,ordId) 唯一——凭据已在
-- 不重复落腿由写径检查承接，本索引=并发/重放兜底硬约束（存量三单无重）。
-- mig-rehearsal-ok: 生产真库实测 GROUP BY (op,ordId) 仅
--   dcd_baseline|11448262 / dcd_baseline|11594773 / dcd_settle|11401353
--   三对互异零重——存量违例不存在，索引建库期不可能炸（O-33362 取证）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check
	CHECK (kind = ANY (ARRAY['fee','funding_settle','trade_pnl','rollback_cost',
		'deposit','withdraw','earn_in','earn_out','interest','recon_adjust',
		'spot_buy','spot_sell','owner_in','owner_out',
		'dust_in','dust_out','supply_income','airdrop','unmapped_bill',
		'dcd_out','dcd_in'])) NOT VALID;
ALTER TABLE fin_cashflow VALIDATE CONSTRAINT fin_cashflow_kind_check;

INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
	('dcd_out','transfer','双币赢申购扣款——自有资金入结构产品内部移动（O-33362）'),
	('dcd_in','transfer','双币赢结算回款本金腿——产品到期回款内部移动（O-33362）')
ON CONFLICT (kind) DO NOTHING;

CREATE UNIQUE INDEX IF NOT EXISTS fin_risk_decision_dcd_cred_uk
	ON fin_risk_decision ((intent->>'op'), (intent->>'ordId'))
	WHERE intent->>'op' IN ('dcd_baseline','dcd_settle');
