-- 0162 review_rw 补 ops_headless 授权（升件#9480 伴生病灶——o-research
-- FIN_PG_DSN=review_rw 的 ops_event 升件写径自始 DOA：ops_event_touch_
-- headless 触发器对全部 INSERT 无条件 touch ops_headless，review_rw 持
-- ops_event INSERT 而零 headless 权→触发器炸 42501→dcd 自停升件+伴事务
-- fin_condition_gap 复役行整批滚丢（产线实证：连败 10-15 拍逐拍 WARN
-- auto-retire persistence failed permission denied for table ops_headless，
-- 升件面零行=静默死）。授 SELECT+INSERT+UPDATE 三件套同 roster_rw 形
-- （touch 触发器写心跳/有界态需 UPDATE）——DELETE 不授（有界态簿记面
-- append/state 翻位非删行），同 fin_0160 豁免表坑形（建表忘授权=臂自始死，
-- canon runbook-write-path-needs-grant）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT SELECT, INSERT, UPDATE ON ops_headless TO review_rw';
	END IF;
END $$;
