-- fin_0106 T-0322 件A：o_fin_ro 窄读角色——业主面减负替代包
-- 授予对象=分析/讨论类只读席（COS 亦挂此角色读事实）：钱面/候选面
-- 事实 SELECT-only 白名单制。角色 LOGIN 凭据（口令）产出面手工置——
-- collector_rw 先例（凭据不走迁移）；空口令=fail-closed 不可认证。
-- 黑名单=结构性不授非配置项：app_secrets 及一切凭据物料列永不授——
--   REVOKE 显式钉死（防上游误授 PUBLIC 渗漏）；白名单外表一概不授，
--   写权（INSERT/UPDATE/DELETE/TRUNCATE）一概不授。
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
        CREATE ROLE o_fin_ro LOGIN;
    END IF;
END $$;

GRANT SELECT ON fin_ledger_snapshot, fin_position, fin_strategy,
    fin_opp_eval, fin_opp_hypothesis, fin_opp_template, fin_eval_reject,
    fin_proposal,
    ops_event, ops_headless, ops_ticket_usage,
    ops_seat_latest, ops_ticket_state
TO o_fin_ro;

-- fin_capability_surface=md 域表（PG-md 正本；SG/全链合 schema 面方在）——
-- fin 链独立 schema 测面缺席=条件授权跳过不炸；存在面必授不静默漏。
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
               WHERE c.relname='fin_capability_surface'
                 AND n.oid=current_schema()::regnamespace) THEN
        EXECUTE 'GRANT SELECT ON fin_capability_surface TO o_fin_ro';
    END IF;
END $$;

REVOKE ALL ON app_secrets FROM o_fin_ro;
