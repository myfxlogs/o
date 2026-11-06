-- fin_0121 T-0386 ops_event 防篡改账——hash 链化+禁 UD（自我迭代加固族①，
--   grounder 判例移植：append-only 脊+SHA 链=自治可信根）。
-- 四件：
--   ① prev_hash/hash 两列——从迁移后首个新行起链（老行不回填——历史不可变
--     已是事实；创世行=迁移后首个链行 prev_hash=NULL 锚注明）；
--   ② 链写=BEFORE INSERT 触发器（票面许「app 层统一链函数 或 DB trigger 兜，
--     择一注释引证」——取 trigger：写面异构多径（report.sh psql/Go 服务族/
--     meta 壳脚本/部署尾拍）无单一 app 层收口点，库内闸才全径必经；且写者
--     显式供 hash/prev_hash 被 trigger 覆写——伪戳入库即不可能）；
--   ③ 禁 UD=REVOKE UPDATE/DELETE/TRUNCATE 应用角色（写角色册查：prod fin
--     2026-11-06 实测现役写角色=roster_rw 持 INSERT+SELECT——写面照旧；
--     UD/TRUNCATE 全应用角色收编，owner 面不触及；TRUNCATE 同族收——
--     毁行同义于改账）。锁权成函数 ops_event_lockdown()——迁移自调+
--     授权漂移重锁径复用（桩测断言同调一式不抄词）；
--   ④ 验链函数 ops_chain_verify() 库内单源——验链拍（lifecycle 饥饿同槽）
--     与 scripts/check_ops_chain.sh 同调一式，口径不裂。

ALTER TABLE ops_event ADD COLUMN IF NOT EXISTS prev_hash text;
ALTER TABLE ops_event ADD COLUMN IF NOT EXISTS hash text;
COMMENT ON COLUMN ops_event.prev_hash IS 'SHA-256 链上前驱链行 hash（T-0386——迁移后首链行=创世锚 NULL；老行不回填恒 NULL）';
COMMENT ON COLUMN ops_event.hash IS 'SHA-256(id|ts|seat|event|ticket|note|prev_hash)（T-0386 链写触发器落——伪戳/断链=ops_chain_verify 检出）';

-- 链写函数 BEFORE INSERT：链头=最新已链行 hash，创世=NULL。
-- pg_advisory_xact_lock 串行链写——并发插入同读旧链头会产出断链
-- （append-only 低频账串行本可忽略；RC 快照下持锁后 SELECT 见最新已提交链头）。
CREATE OR REPLACE FUNCTION ops_event_chain_link() RETURNS trigger AS $$
DECLARE
	prev text;
BEGIN
	PERFORM pg_advisory_xact_lock(hashtext('ops_event_chain'));
	SELECT o.hash INTO prev FROM ops_event o
		WHERE o.hash IS NOT NULL ORDER BY o.id DESC LIMIT 1;
	NEW.prev_hash := prev;
	NEW.hash := encode(sha256(
		(NEW.id||'|'||to_char(NEW.ts AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
		 ||'|'||NEW.seat||'|'||NEW.event||'|'||NEW.ticket||'|'||NEW.note
		 ||'|'||COALESCE(prev,''))::bytea),'hex');
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER ops_event_chain_trg BEFORE INSERT ON ops_event
	FOR EACH ROW EXECUTE FUNCTION ops_event_chain_link();

-- 链头读依赖：链写触发器 INVOKER 语义——INSERT 持权角色须持 SELECT
-- （roster_rw/research_rw=ops_event 写册，缺 SELECT=插行炸——「不破坏写面」
--  红线同迁移守；角色缺席环境零炸同族规）。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_event TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_event TO research_rw';
	END IF;
END $$;

-- 锁权函数（幂等）：应用角色 UD/TRUNCATE 收编——迁移自调一次，授权漂移后
-- 可重锁（check 面只断言不改权）。角色缺席件逐跳——全新部署/测试实例零炸。
CREATE OR REPLACE FUNCTION ops_event_lockdown() RETURNS void AS $$
DECLARE
	r text;
BEGIN
	FOREACH r IN ARRAY ARRAY['roster_rw','research_rw','review_rw','o_fin_ro',
		'analyst_rw','collector_rw','projector_rw','research_ro'] LOOP
		IF EXISTS (SELECT FROM pg_roles WHERE rolname=r) THEN
			EXECUTE format('REVOKE UPDATE,DELETE,TRUNCATE ON ops_event FROM %I', r);
		END IF;
	END LOOP;
	REVOKE UPDATE,DELETE,TRUNCATE ON ops_event FROM PUBLIC;
END;
$$ LANGUAGE plpgsql;
COMMENT ON FUNCTION ops_event_lockdown() IS 'ops_event 禁 UD 锁权（T-0386——应用角色 UPDATE/DELETE/TRUNCATE 收编；幂等可重锁）';

SELECT ops_event_lockdown();

-- 验链函数（验链拍/check_ops_chain 单源）——四类断：
--   创世锚（首链行 prev_hash 非 NULL）/伪戳（hash 重算不等）/
--   断链（prev_hash 与前链行 hash 不等——前链行=最近非空 hash 行，
--   跨裸行不冤枉）/裸行（链起点后无 hash 行——链写失效/绕写）。
CREATE OR REPLACE FUNCTION ops_chain_verify()
RETURNS TABLE(id bigint, kind text, detail text) AS $$
	WITH c AS (
		SELECT id,prev_hash,hash,
			encode(sha256(
				(id||'|'||to_char(ts AT TIME ZONE 'UTC','YYYY-MM-DD"T"HH24:MI:SS.US"Z"')
				 ||'|'||seat||'|'||event||'|'||ticket||'|'||note
				 ||'|'||COALESCE(prev_hash,''))::bytea),'hex') AS want
		FROM ops_event
	),
	x AS (
		SELECT c.*, min(id) FILTER (WHERE hash IS NOT NULL) OVER () AS g0
		FROM c
	)
	SELECT id,'创世锚','首个链行 prev_hash 非 NULL' FROM x WHERE id=g0 AND prev_hash IS NOT NULL
	UNION ALL
	SELECT id,'伪戳','hash 重算不等' FROM x WHERE hash IS NOT NULL AND hash<>want
	UNION ALL
	SELECT id,'断链','prev_hash 与前链行 hash 不等' FROM x
		WHERE hash IS NOT NULL AND id<>g0 AND prev_hash IS DISTINCT FROM (
			SELECT p.hash FROM x p WHERE p.hash IS NOT NULL AND p.id<x.id
			ORDER BY p.id DESC LIMIT 1)
	UNION ALL
	SELECT id,'裸行','链起点后无 hash 行——链写触发器失效/绕写' FROM x
		WHERE hash IS NULL AND id>g0;
$$ LANGUAGE sql STABLE;
COMMENT ON FUNCTION ops_chain_verify() IS 'ops_event SHA 链全链重算（T-0386——验链拍/check_ops_chain 单源；零行=链完）';
