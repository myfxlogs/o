-- 0151 票号原子发号序列（提案-票号原子发号-DB序列消撞号，业主令
-- 2026-11-08 直裁已批外派施工）——fin_ticket_seq 消双席并发取号撞
-- 号类（2026-11-08 单日三连撞实证）。
-- 种子=现仓最大票号+1（author 时 docs/tasks/+done/ 两面扫 max=0495
-- →START 496）；迁移为纯 DB 面读不到 git 票面——种子仅初值，
-- next_ticket.sh 消费径每次先 setval 地板(GREATEST(last_value,
-- repo_max),true) 再 nextval，自校正漂移（半成品窗：迁移未应用/
-- 扫号径并存期撞号仍由 merge 期重编让号流程兜底，对账桩
-- check_ticket_numbers.sh --seq 盯 last_value≥max票号）。
-- 授权 roster_rw：next_ticket.sh 经 FIN_PG_DSN 消费——USAGE
-- (nextval)+SELECT(last_value 读)+UPDATE(setval 地板) 三件套，
-- 序列为专用计数器低敏面。
CREATE SEQUENCE IF NOT EXISTS fin_ticket_seq START WITH 496;

DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT USAGE, SELECT, UPDATE ON SEQUENCE fin_ticket_seq TO roster_rw';
	END IF;
END $$;
