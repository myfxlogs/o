-- 0159 已知残差豁免表处置席写授权（升件#9255/#9283/#9288/#9291/#9294 五连
-- 复燃根因——runbook 对账档钦定「口径残差→fin_known_residual 豁免行」径，
-- 但 fin_0036 建表未授任何角色，roster_rw 持 ops_event/lesson 写径独缺
-- 豁免表=豁免臂自始机械不可达，探针同值每拍暴报噪声）。授 SELECT（读
-- 既有豁免行查重/算豁免后净差）+INSERT（写豁免行）+序列 USAGE——
-- UPDATE/DELETE 不授：豁免行 append-only 同 ops_event 律，到期自动复报
-- 无需改行。表为豁免簿记低敏面（不含金额敏感位——amount_usd 为差值非
-- 仓值），同 fin_0139 verdict 闸箱授 roster_rw 先例。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT, INSERT ON fin_known_residual TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE fin_known_residual_id_seq TO roster_rw';
	END IF;
END $$;
