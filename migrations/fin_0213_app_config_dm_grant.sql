-- 0213 DM 写径授权债：app_config 建表时只给 roster_rw SELECT+UPDATE，
-- DMSet 的 INSERT..ON CONFLICT 无 INSERT 权——dmconfig 从 dev 侧
-- permission denied 哑死（grant-not-in-migration-silent-dead 第四犯，
-- 2026-11-19 earn.auto_purchase_enabled 回开被迫绕径 SG 属主户实证）。
-- 手工补权当日已生效，本件 canonical 防全新部署重演。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT INSERT ON app_config TO roster_rw';
    END IF;
END $$;
