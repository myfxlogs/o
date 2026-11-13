-- fin_0174: bill 记账 kind 闭集收口（升件#14671 审计二巡 P1——
-- dust_in/dust_out/supply_income/airdrop 册载而 DB 闭集缺位=零写者病灶
-- 本体；补 unmapped_bill 残差 kind=未映射子型 balChg≠0 守恒落账位）。
-- class 册：dust/supply/airdrop 四 kind fin_0075 已在册，仅补 unmapped_bill
-- （adjustment——未归因移动非赚非转，留待人工归因不改 income/transfer 计）。
ALTER TABLE fin_cashflow DROP CONSTRAINT fin_cashflow_kind_check;
ALTER TABLE fin_cashflow ADD CONSTRAINT fin_cashflow_kind_check
	CHECK (kind = ANY (ARRAY['fee','funding_settle','trade_pnl','rollback_cost',
		'deposit','withdraw','earn_in','earn_out','interest','recon_adjust',
		'spot_buy','spot_sell','owner_in','owner_out',
		'dust_in','dust_out','supply_income','airdrop','unmapped_bill']));

INSERT INTO fin_cashflow_kind_class(kind,class,note) VALUES
	('unmapped_bill','adjustment','未映射账单现金移动——残差落账待归因（升件#14671）')
ON CONFLICT (kind) DO NOTHING;
