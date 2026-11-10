-- fin_0158_ops_chain_verify_dag.sql —— T-0386 验链函数 DAG 口径校准（升件 #9139 根修）。
--
-- 病灶：原「断链」检按 id 序判（prev_hash 与前一非空 hash 行不等即断）——
--   但链头读规则是 ORDER BY id DESC（fin_0121 链写触发器同口径），
--   而 id 由 nextval 在 advisory_xact 锁【之前】分配：并发写手下
--   低 id 迟提行必然指向前方高 id 链头，且其后继仍指 id 最大链头
--   →每次穿插产两件假阳（迟提行自身 + 其 id 序后继）+ 一叉枝
--   （同父双子）。11-09 02:37/02:58 两窗层扫期 CTO/决策席 批量
--   与 起草拍/MECH 单行穿插实证 4 断全为此形——内容 hash 全数
--   重算有效（零伪戳），指涉行俱在，非篡改非断供。
--
-- 校准：链=防篡改证非全序史——安全性要件是「内容 hash 承诺前驱」
--   非「id 邻接线性」。重定义四类检：
--   创世锚/伪戳/裸行 三检原义不动；
--   断链=prev_hash 悬指（指涉 hash 不在表内——删行/越域链头/手写
--   伪针方为真断）；id 乱序/同父叉枝=并发结构性形态不报。
--   消费面 opschain.Verify/ReportBreak 与 check_ops_chain.sh LIVE
--   段同函数单源自动继承，零改面。
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
	SELECT id,'断链','prev_hash 悬指——指涉 hash 不在表内' FROM x
		WHERE hash IS NOT NULL AND prev_hash IS NOT NULL
			AND NOT EXISTS (SELECT 1 FROM x p WHERE p.hash=x.prev_hash)
	UNION ALL
	SELECT id,'裸行','链起点后无 hash 行——链写触发器失效/绕写' FROM x
		WHERE hash IS NULL AND id>g0;
$$ LANGUAGE sql STABLE;
COMMENT ON FUNCTION ops_chain_verify() IS 'ops_event SHA 链全链重算 DAG 口径（T-0386/fin_0158 校准——断链=prev_hash 悬指非 id 邻序；并发迟提叉枝=结构性形态不报；验链拍/check_ops_chain 单源；零行=链完）';
