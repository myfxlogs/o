-- 0156 capalloc 窄门补权②：roster_rw USAGE config_generation_seq
-- （app_config UPDATE 触发 generation=nextval 默认——0155 补了
--   SELECT 面,序列 USAGE 漏网=窄门写键首实测即撞权限
--   (SQLSTATE 42501)。产线手工补权已生效,本件回 canonical 链）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT USAGE ON SEQUENCE config_generation_seq TO roster_rw';
    END IF;
END $$;
