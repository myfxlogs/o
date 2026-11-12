-- fin_0169 账链豁免册（工单#11419——升件#11399 定性处置落件）：
-- #9442 孤行伪戳系写入面越权改写/绕写孤案（存戳与行内容重算不等、
-- 链拓扑完好、邻行全绿），PG 无行级写史不可回溯归因；append-only
-- 禁回填→检出永久在账每拍假警。立 known-residual 册：
--   ①册行仅抑「伪戳」一档检出——断链/裸行/创世锚三检对册内行仍判
--     （豁免=「该行内容重算不等已知且立案」非「该行免一切检」）；
--   ②expires_at NULL=永久（不可愈残类），有期=限窗豁免期到复检自警；
--   ③册 append-only——应用角色只 INSERT/SELECT，UD/TRUNCATE 默认拒。
CREATE TABLE IF NOT EXISTS ops_chain_residual (
	id bigint PRIMARY KEY,
	reason text NOT NULL,
	ticket_ref text NOT NULL,
	created_by text NOT NULL DEFAULT current_user,
	created_at timestamptz NOT NULL DEFAULT now(),
	expires_at timestamptz
);
COMMENT ON TABLE ops_chain_residual IS 'ops_event 账链检出豁免册（fin_0169/工单#11419——伪戳已知立案行抑重报，拓扑三检不豁免；append-only）';

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
		AND NOT EXISTS (SELECT 1 FROM ops_chain_residual r WHERE r.id=x.id
			AND (r.expires_at IS NULL OR r.expires_at>now()))
	UNION ALL
	SELECT id,'断链','prev_hash 悬指——指涉 hash 不在表内' FROM x
		WHERE hash IS NOT NULL AND prev_hash IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM x p WHERE p.hash=x.prev_hash)
	UNION ALL
	SELECT id,'裸行','链起点后无 hash 行——链写触发器失效/绕写' FROM x
		WHERE hash IS NULL AND id>g0;
$$ LANGUAGE sql STABLE;
COMMENT ON FUNCTION ops_chain_verify() IS 'ops_event SHA 链全链重算 DAG 口径（fin_0169 豁免册接入——ops_chain_residual 册内行抑伪戳重报一档，断链/裸行/创世锚照判；fin_0158 DAG 口径不变）';

-- 种子行：#9442 孤行伪戳定案豁免（升件#11399 处置行全案）。
INSERT INTO ops_chain_residual(id,reason,ticket_ref) VALUES
	(9442,'11-10 孤行伪戳——存戳 f05cded5 与行内容重算不等，链拓扑完好无断链/裸行；写入面越权改写/绕写孤案不可回溯归因，append-only 禁回填','升件#11399')
ON CONFLICT (id) DO NOTHING;

-- 授权：册读面开应用读册，写面仅处置/复审两席写册（roster_rw/review_rw）；
-- UD/TRUNCATE 默认无授——册 append-only。
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT ON ops_chain_residual TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='review_rw') THEN
		EXECUTE 'GRANT SELECT,INSERT ON ops_chain_residual TO review_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_chain_residual TO o_fin_ro';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='research_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_chain_residual TO research_ro';
	END IF;
END $$;
