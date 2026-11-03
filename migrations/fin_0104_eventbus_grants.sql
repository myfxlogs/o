-- 0104 T-0312 事件总线债：eventbus FIN 侧轮询源对 review_rw 补 SELECT
-- （review_rw=FIN 写入角色原只持 INSERT——max(id) 水印查询哑死=授权坑
--   第四/五犯连发，照 fin_0103 条件 GRANT 形回链；手工补权 2026-11-04
--   已生效，本件 canonical 防全新部署哑死）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
        EXECUTE 'GRANT SELECT ON fin_econ_lesson TO review_rw';
        EXECUTE 'GRANT SELECT ON ops_event TO review_rw';
    END IF;
END $$;
