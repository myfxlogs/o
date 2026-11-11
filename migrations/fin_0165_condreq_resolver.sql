-- 0165 condreq_resolver 结案户（T-0512 飞书钮回调落地前置——「写者
-- ≠结案者」边界保持：roster_rw 只写不结，结案权独立户只结不写。
-- 最小面：SELECT(id,state)+UPDATE(state,resolved_at,resolve_note)
-- ——resolve 谓词只触 pending/id,不触他列不写请求体）。
-- 口令不入迁移——部署后运维 SET PASSWORD 径凭据归 app_secrets
-- feishubridge.resolver_dsn（凭据归 PG 律）。
DO $$ BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='condreq_resolver') THEN
    CREATE ROLE condreq_resolver LOGIN;
  END IF;
END $$;
GRANT SELECT (id,state) ON fin_condition_gap TO condreq_resolver;
GRANT UPDATE (state,resolved_at,resolve_note) ON fin_condition_gap TO condreq_resolver;
