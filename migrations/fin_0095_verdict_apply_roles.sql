-- 0095 T-0232 Phase-2 权限收窄（乙案判决/状态应用分离——D-0036 §12.2 裁4）：
--
-- 一、fin_verdict_cursor：SG 侧判决消费者持久化游标（过渡件——outbox 判决道
--     切换后随消费者同废；幂等键=fin_review.id 单调水位）。
-- 二、四角色收窄（T-0223 授权册为审计输入）：
--     research_ro  SELECT fin_position/fin_strategy/fin_cashflow+研究面按需读
--     review_rw    INSERT fin_review/fin_review_verdict+判决取证读面
--     roster_rw    席位台账自报族（registry/ops/hostmetrics/meta 读+secrets 读）
--     projector_rw （已建——八桥镜像 upsert 面不扩不收）
--     collector_rw SG 面全撤（只留 md 面——OCI PG-md 侧角色不在本链）
-- 三、research_rw 全撤+NOLOGIN（36 表宽写权归零——T-0169 直写钱径态病灶根修；
--     其自报/读取径全由 roster_rw 承接，查询读径由 research_ro 承接）。
-- 四、口令零落库零入仓：新角色口令=SCRAM 哈希自 research_rw 同值复制
--     （pg_authid 运行时取——密文不字面入册）；env 侧只换用户名密文不动。
--
-- 施工切序（实机手落与本迁移等幂）：①本文件角色+授权段先落 → ②OCI/dev env
--   换用户名+unit 换 → ③本文件收窄段（REVOKE+NOLOGIN）落 = 判决链零断窗。
CREATE TABLE IF NOT EXISTS fin_verdict_cursor (
    k            smallint PRIMARY KEY DEFAULT 1 CHECK (k = 1),
    applied_upto bigint NOT NULL DEFAULT 0,
    updated_at   timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE fin_verdict_cursor IS '判决消费持久化游标（T-0232 过渡件——fin_review.id 单调水位；outbox 判决道切换同废）';
COMMENT ON COLUMN fin_verdict_cursor.applied_upto IS '已应用 fin_review.id 水位（幂等键——重放=谓词天然无操作）';
INSERT INTO fin_verdict_cursor(k) VALUES (1) ON CONFLICT (k) DO NOTHING;
-- 投产种子：游标立时已判行=旧径已应用面——只消费立标后新增判决（防历史重放
-- 误迁非 live 策略/误退后 armed 提案）。守卫：fin_review 缺席域（空库/无 fin
-- 链面）静默零行——正规径 fin_0011 在前恒在。
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.tables
		WHERE table_schema=current_schema() AND table_name='fin_review') THEN
		EXECUTE 'UPDATE fin_verdict_cursor SET applied_upto=
			(SELECT COALESCE(max(id),0) FROM fin_review) WHERE applied_upto=0';
	END IF;
END $$;

-- ── 角色创建（幂等——缺席才建）──────────────────────────
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='research_ro') THEN
		CREATE ROLE research_ro LOGIN;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='review_rw') THEN
		CREATE ROLE review_rw LOGIN;
	END IF;
	IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='roster_rw') THEN
		CREATE ROLE roster_rw LOGIN;
	END IF;
END $$;

-- ── 授权面 ────────────────────────────────────────────
-- research_ro：研究面 fin 读（钦定三表+T-0223 反桥读径 continuity——
-- fin_venue_untradeable/md_earn_rate 反桥源表按需读+app_config 配置读）。
GRANT SELECT ON fin_position,fin_strategy,fin_cashflow,
	fin_venue_untradeable,app_config TO research_ro;
DO $$ -- md_earn_rate 正本 md 链（SG 双链并集在，fin-only schema 缺席守卫授）。
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.tables
		WHERE table_schema=current_schema() AND table_name='md_earn_rate') THEN
		EXECUTE 'GRANT SELECT ON md_earn_rate TO research_ro';
	END IF;
END $$;

-- review_rw：判决事实追加+判决取证读面（乙案——OCI→SG 仅剩写面）。
GRANT INSERT,SELECT ON fin_review,fin_review_verdict TO review_rw;
GRANT USAGE ON SEQUENCE fin_review_id_seq,fin_review_verdict_id_seq TO review_rw;
GRANT SELECT ON fin_position,fin_strategy,fin_proposal,fin_opp_eval,
	fin_execution,fin_cashflow TO review_rw;

-- roster_rw：席位台账自报族（registry beat/ops 事件簿/hostmetrics/meta 任务面
-- 读+secrets 读——research_rw 自报径全职能承接；app_secrets SELECT=feishu-notify
-- 解密径与收窄前同曝面零扩）。
GRANT SELECT,INSERT,UPDATE ON fin_employee_registry,ops_headless,fin_host_metrics TO roster_rw;
GRANT INSERT,SELECT ON ops_event,ops_ticket_usage TO roster_rw;
GRANT SELECT ON ops_seat_latest,ops_ticket_state,app_config,app_secrets TO roster_rw;
GRANT USAGE ON SEQUENCE ops_event_id_seq,ops_headless_seq,ops_ticket_usage_id_seq TO roster_rw;
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name='meta') THEN
		EXECUTE 'GRANT USAGE ON SCHEMA meta TO roster_rw';
		EXECUTE 'GRANT SELECT ON meta.meta_task,meta.task_phase_edges,meta.task_phase_words TO roster_rw';
		EXECUTE 'GRANT EXECUTE ON FUNCTION meta.task_transition(text,text,text,text,text,text,text,text,text,text,text,integer,boolean) TO roster_rw';
	END IF;
END $$;

-- ── 口令哈希同值复制（密文不字面入册——env 只换用户名不动密文）──
DO $$
DECLARE h text;
BEGIN
	SELECT rolpassword INTO h FROM pg_authid WHERE rolname='research_rw';
	IF h IS NOT NULL THEN
		EXECUTE format('ALTER ROLE research_ro PASSWORD %L', h);
		EXECUTE format('ALTER ROLE review_rw PASSWORD %L', h);
		EXECUTE format('ALTER ROLE roster_rw PASSWORD %L', h);
	END IF;
END $$;

-- ── 收窄段（切次序③——revoke+NOLOGIN；幂等可重放）────────
DO $$
BEGIN
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='research_rw') THEN
		REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM research_rw;
		REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM research_rw;
		REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM research_rw;
		IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name='meta') THEN
			REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA meta FROM research_rw;
			REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA meta FROM research_rw;
			REVOKE USAGE ON SCHEMA meta FROM research_rw;
		END IF;
		IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name='_timescaledb_internal') THEN
			REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA _timescaledb_internal FROM research_rw;
		END IF;
		ALTER ROLE research_rw NOLOGIN;
	END IF;
	IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname='collector_rw') THEN
		REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM collector_rw;
		REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM collector_rw;
		REVOKE ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public FROM collector_rw;
		IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name='_timescaledb_internal') THEN
			REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA _timescaledb_internal FROM collector_rw;
		END IF;
		ALTER ROLE collector_rw NOLOGIN;
	END IF;
END $$;
