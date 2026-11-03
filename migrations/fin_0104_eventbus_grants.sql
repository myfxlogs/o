-- 0104 T-0312 事件总线债：fin_econ_lesson 对 review_rw 补 SELECT
-- （FIN 侧 eventbus 轮询源——review_rw 原持 INSERT 写锚无 SELECT，
--   max(id) 水印查询哑死=授权坑第四犯，照 fin_0103 条件 GRANT 形回链；
--   手工补权 2026-11-04 已生效，本件 canonical 防全新部署哑死）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_econ_lesson TO review_rw';
    END IF;
END $$;
