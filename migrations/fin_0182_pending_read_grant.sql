-- 0182 议程面读权补授（坑处置:fin_0177 建 ops_esc_pending 视图未授读角色——
--   唤知/独白会话以 roster_rw/o_fin_ro 连库读未决面 permission denied;
--   dispatch.sh psql 2>/dev/null fail-open 吞错,唤知阈/议程单对全体会话
--   静默盲（独白#16130 首拍自报「议程栏读不到」实证）。
-- 口径:SELECT 读面非写面——roster_rw=唤知取证席/独白读账角,
--   o_fin_ro=只读备角;ops_esc_valence 基表同授(权重查询面)。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_esc_pending TO roster_rw';
		EXECUTE 'GRANT SELECT ON ops_esc_valence TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_esc_pending TO o_fin_ro';
		EXECUTE 'GRANT SELECT ON ops_esc_valence TO o_fin_ro';
	END IF;
END $$;
