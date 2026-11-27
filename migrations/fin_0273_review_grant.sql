-- 0273 review_rw 补 onboard 臂双表授权（升件#43077——fin_0162 同型病灶：
-- 建表忘授权=臂自始死。实证：o-research venue=xmglobal+exness 推进败
-- （ops_event_act 42501——ops_event INSERT 触发器链件）+漂移探针拍败
-- （fin_symbol_dialect 42501——探针⑤读面）连环 WARN）。
-- 授权面=onboard 臂（r.fin=FIN_PG_DSN=review_rw）最小实证面：
--   ops_event_act 授 SELECT+INSERT 同 roster_rw 形（fin_0191 授权口径——
--     推进路径 ops_event INSERT 触发器链读/写件）；
--   fin_symbol_dialect 授 SELECT（漂移探针⑤读面 mt5/dialect.go Rows——
--     写面在 cmd/core/dialect_submit SG 超户域，review_rw 无需写）。
-- SG 侧 2026-11-27 已先活授同口径治臂（臂 WARN 止）——本迁移为簿记对齐，
-- GRANT 幂等重放无害。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT SELECT, INSERT ON ops_event_act TO review_rw';
		EXECUTE 'GRANT SELECT ON fin_symbol_dialect TO review_rw';
	END IF;
END $$;
