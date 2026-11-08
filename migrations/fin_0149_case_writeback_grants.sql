-- 0149 GW-001 段E1 案卷回写授权——roster_rw 补 fin_econ_lesson
-- INSERT(status,verdict_ref) 柱级（复发=新案落位写径：case_writeback.sh
-- 复发分支 INSERT 新 rootcause_diag 案 entity_ref=<base>#r<n>
-- status='pending' verdict_ref='' 重入诊断队；kind/entity_ref/numbers
-- 柱级 INSERT 已由 fin_0146 在案、verdict_ref UPDATE/全列 SELECT 由
-- fin_0103 在案——本迁移（原0148让位eval_pulse改0149）只补缺两柱，窄授权不授全表 INSERT）。
-- ops_event INSERT/SELECT live 实测已在（旁路授权先例 fin_0147 同型）
-- ——正本补齐防新部署断链（案卷回写三处归流/面戳行皆写 ops_event）。
-- 序列 USAGE 幂等重申——live 实测已用，正本补齐防新部署断链。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT INSERT (status,verdict_ref) ON fin_econ_lesson TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE fin_econ_lesson_id_seq TO roster_rw';
		EXECUTE 'GRANT INSERT ON ops_event TO roster_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_event_id_seq TO roster_rw';
	END IF;
END $$;
