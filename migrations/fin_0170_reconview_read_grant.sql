-- fin_0170: reconview 对账取证读面补授（工单#11438 附）
-- 病灶：tools/reconview 以 roster_rw 连库，三段印（账本分量/venue 现读/diff）
--   的账本侧两读面被拒——fin_risk_decision 零授权（ReconSets 停泊/dcd 台账源）、
--   fin_cashflow 未授 roster_rw（净额印源）；查询 _= 吞错致印面全空无声。
-- 口径：补授 SELECT（读面非写面）——roster_rw 唤知取证席、o_fin_ro 只读备角；
--   与 fin_known_residual/ops_chain_residual 既有授式同形（角色存在才授）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON fin_risk_decision TO roster_rw';
		EXECUTE 'GRANT SELECT ON fin_cashflow TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON fin_risk_decision TO o_fin_ro';
		EXECUTE 'GRANT SELECT ON fin_cashflow TO o_fin_ro';
	END IF;
END $$;
