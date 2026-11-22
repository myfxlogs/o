-- fin_0228_ops_esc.sql —— 升件写径归一函件（工单O-30514 形散神不散
-- 根修①）：escalate.sh QC() 转义正解与 deploy_watch.sh esc() 手搓
-- 无转义两份发散实现→引号断径致升件写败被 tolerate 吞（#29764/
-- #29766 实证）。本函件=escalate.sh 现件语义全量复刻的库内单写径：
--   note=prefix||'——'||detail；24h note 前缀同窗去重（窗内有=不写
--   返回 NULL；去重查询自身败=fail-open 照常写——「报侧不吞讯」同
--   escalate.sh 空答照常升语义）；esc_class/esc_scope 空串→NULL
--   （escalate.sh CCLS/CSCP 同判）；INSERT 成功返回写入行 id，
--   失败异常上抛归调用侧（写径败显形不吞——兜底/自审语义在调用
--   壳不在函件）。
-- 调用面：escalate.sh 薄壳 + deploy_watch.sh esc() 主径（psql -v
-- 参数化喂参——引号问题结构性绝根）；Go 侧参数化 INSERT 不动。
CREATE OR REPLACE FUNCTION ops_esc(
	seat text, ticket text, esc_class text, esc_scope text,
	prefix text, detail text
) RETURNS bigint
LANGUAGE plpgsql AS $fn$
DECLARE
	v_note text := prefix || '——' || detail;
	v_id   bigint;
BEGIN
	-- 同窗去重：note 前缀 24h 内已有升件行=本窗已报（同根因多面齐发
	-- 只报一行）。查败=fail-open 照常写（escalate.sh「去重查询自身
	-- 败=空答照常升」复刻——吞错只兜去重面不兜写面）。
	BEGIN
		IF EXISTS (SELECT 1 FROM ops_event
			WHERE event = '升件' AND note LIKE prefix || '%'
			  AND ts > now() - interval '24 hours') THEN
			RETURN NULL;
		END IF;
	EXCEPTION WHEN OTHERS THEN
		NULL;
	END;

	INSERT INTO ops_event(seat, event, ticket, note, esc_class, esc_scope)
	VALUES (seat, '升件', ticket, v_note,
		NULLIF(esc_class, ''), NULLIF(esc_scope, ''))
	RETURNING id INTO v_id;
	RETURN v_id;
END;
$fn$;

COMMENT ON FUNCTION ops_esc(text,text,text,text,text,text) IS
	'升件写径归一函件（O-30514）——escalate.sh 语义库内复刻：note=prefix——detail、24h 前缀同窗去重(窗内有=返NULL不写)、class/scope空→NULL、成功返行id';
