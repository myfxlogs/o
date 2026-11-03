-- 0103 T-0314 债清：fin_econ_lesson/app_config 消费角色补权回链
-- （roster_rw=dev 裁定席隧道凭据——裁定三步写面钦定窄白名单：
--   lesson SELECT+UPDATE 裁行；app_config UPDATE 机械回改点 gate.* 键。
--   手工补权 2026-11-03 已生效，本件回 canonical 链防全新部署哑死）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT,UPDATE ON fin_econ_lesson TO roster_rw';
        EXECUTE 'GRANT UPDATE ON app_config TO roster_rw';
    END IF;
END $$;
