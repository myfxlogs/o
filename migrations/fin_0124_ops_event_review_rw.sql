-- 0124 T-0402 OCI 臂 ops_event 写拒权——review_rw 补最小写权。
-- 病灶：o-research finPool=review_rw（FIN_PG_DSN 隧道 SG fin 正本）写
--   ops_event 三径（onboard「venue接入」/dcd 升件/grammar volsell 升件）
--   被 42501 拒——fin_0104 只授 SELECT（eventbus 水印读面），写面从未授；
--   onboard 自报行静默丢=业主可见性面漏。
-- 授权最小化红线：INSERT+序列 USAGE（bigserial nextval 必需）+SELECT
--   （链写触发器 INVOKER 读链头所需——fin_0104 已授，幂等重授保全新部署）；
--   UD/TRUNCATE 不授（fin_0121 ops_event_lockdown 册内已收编，同口径守）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT INSERT ON ops_event TO review_rw';
		EXECUTE 'GRANT USAGE ON SEQUENCE ops_event_id_seq TO review_rw';
		EXECUTE 'GRANT SELECT ON ops_event TO review_rw';
	END IF;
END $$;
