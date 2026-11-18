-- 0210 工单#24412 研究席唤起径——roster_rw 补 fin_venue_candidate SELECT
-- （dispatch_research.sh 触发③ T-0300 达阈扫 state='recommended'——
--   roster_rw=派发拍会话角色,fin_0098 只授 o+projector_rw;
--   只读窄权同 0103 先例——无行依赖纯权面,幂等 IF EXISTS 门）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_venue_candidate TO roster_rw';
    END IF;
END $$;
