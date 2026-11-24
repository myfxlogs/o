-- fin_0249 venue exec enablement（工单O-37449——CFD 多账户执行面）：
--   ① fin_venue_roster 读面补授：roster_rw（dev/agent 取证角）+o_fin_ro
--     （只读备角）SELECT——fin_0248 漏授实证（roster_rw 全 denied；core=
--     owner 自读不受影响故缺授未爆，窄权角色取证面哑）。collector 桥腿
--     PG-md 本无此表读权不在授列（venue 寻址会话键设计同因）。
--   ② XM 方言硬验行落 fin_symbol_dialect（venue='xmglobal'
--     account='110660304'——BTCUSD#/ETHUSD#）：
--     venue_symbol=XM SymbolList 实读名（'#' 尾方言——O-35250 剥层先例）；
--     canonical=venue_symbol 自指形——xvenueOpen 执行径 Lookup 键=副腿
--     inst_id=cfd venue 符号（xvenuepair.go 建序钉），UNIQUE(venue,
--     account,venue_symbol) 禁双 canonical 变体并存故取执行径正本；
--     全字段=2026-11-24 活会话 SymbolParams/GetQuote 实读（共享会话零
--     新 login——tools/xmprobe 取证）：ContractSize=1（1 BTC/ETH 每手）
--     digits=2 tick_sz=0.01 profit=USD trade_mode=4(FullAccess)；
--     min_lot BTC=0.01/ETH=0.02 lot_step=0.01；点差实测 2.5/11.1bps
--     远低于 verifySymbolMapping 验收带；spec_hash=mt5.SpecHash 同函数
--     实算指纹（漂移探针⑤基线）。quote_snapshot=实读证据快照留痕。
--     verifySymbolMapping 同径判据全过：ctVal 币本位对拍/计价币 USD 锚定/
--     digits>0/FullAccess/活价 bid·ask>0+点差带内——禁猜名判据原点。

DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON fin_venue_roster TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON fin_venue_roster TO o_fin_ro';
	END IF;
END $$;

INSERT INTO fin_symbol_dialect(venue,account,canonical,venue_symbol,ct_val,
	tick_sz,min_sz,lot_step,lot_unit,digits,trade_mode,spec_hash,quote_snapshot,verified_at)
VALUES
('xmglobal','110660304','BTCUSD#','BTCUSD#',1,0.01,0.01,0.01,'lot',2,4,
 '0f91d49f2ab92d0d2ac751a91314d2c5525da16888d0d8397e0e5abbf542044d',
 '{"src":"tools/xmprobe 2026-11-24 活会话实读","bid":60684.75,"ask":60699.75,"spread_bps":2.47,"max_lots":80}'::jsonb,now()),
('xmglobal','110660304','ETHUSD#','ETHUSD#',1,0.01,0.02,0.01,'lot',2,4,
 'fbd43f36e8783214d38c4c04f428bb7227bf82ed6cfdb727c8e36a614952e95f',
 '{"src":"tools/xmprobe 2026-11-24 活会话实读","bid":1707.02,"ask":1708.92,"spread_bps":11.12,"max_lots":1180}'::jsonb,now())
ON CONFLICT (venue,account,canonical) DO NOTHING;
