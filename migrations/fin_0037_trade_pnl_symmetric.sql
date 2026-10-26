-- 0037 平仓腿本金误记损益矫正（T-0082：source_table='fin_order_leg' 的 trade_pnl
-- 行若为毛额（≈腿名义值）——对称口径修正=卖出所得−开仓成本（notional/2 腿基准）；
-- 本金回笼当利润病灶=幻影盈清场，改后 amount=真价差级数字）。
UPDATE fin_cashflow c SET amount = c.amount - p.notional_usd/2
FROM fin_position p
WHERE c.position_ref = p.id AND c.kind='trade_pnl' AND c.source_table='fin_order_leg'
  AND abs(c.amount) > p.notional_usd*0.1;  -- 毛额级行才矫正（价差级行不动）
