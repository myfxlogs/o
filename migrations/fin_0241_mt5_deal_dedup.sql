-- fin_0241 mt5_deal 充提入账幂等硬约束（工单O-35753——MT5 充提采集臂
-- fin_cashflow 落账 source_table='mt5_deal' 新源域）。
--   部分唯一索引钉 (venue,source_table,source_ref) 三元组——spec 钦定
--   去重键；WHERE 限 mt5_deal 域零存量行（新源表——违例集=空集，
--   索引建库期不可能炸），他源表 source_ref 复用形不受约束（ops_event
--   人工簿记先例行共存不冲突）。
--   写径双保险：INSERT...WHERE NOT EXISTS 预检（人工先例冲突闸同语句）
--   +本索引并发/重放兜底硬约束（fin_0233 dcd 凭据索引同构先例）。
-- mig-rehearsal-ok: source_table='mt5_deal' 为新源表存量零行（生产实测
--   fin_cashflow 无 mt5_deal 行——SELECT count(*)=0 取证在案），
--   部分索引违例集=空集不可能部署期炸。
CREATE UNIQUE INDEX IF NOT EXISTS fin_cashflow_mt5_deal_uk
	ON fin_cashflow (venue, source_table, source_ref)
	WHERE source_table = 'mt5_deal';
