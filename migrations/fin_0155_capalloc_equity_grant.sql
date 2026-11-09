-- 0155 capalloc 窄门补权：roster_rw SELECT md_equity_curve
-- （唤知会话调 tools/capalloc 判闸值界——净值比例包络锚在
--   md_equity_curve ccy=NAV 末帧；无读权=窄门永远 fail-closed
--   哑死。0151 同例：先手工补权生效，本件回 canonical 链）。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
        EXECUTE 'GRANT SELECT ON md_equity_curve TO roster_rw';
    END IF;
END $$;
