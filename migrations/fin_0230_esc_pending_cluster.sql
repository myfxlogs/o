-- fin_0230: ops_esc_pending 同根簇折叠（工单O-27914——同根断供簇
--   多 scope 续火去重,未决面簇归一 P2）。
--
-- 病灶：票抑键=esc_scope 精确单键压不住跨 scope 同根簇——MT5 vendor
--   断供(#26356)单日续火 6 轮×4 scope,每轮耗 1 会话+4 处置行
--   (canon ticket-scope-suppress-key-mismatch 在案)。
--
-- 取舍=候选①升件面并簇键(视图层最小面,产线零动):
--   簇签(钉死判据)=note 内首个 '#<digits>' ∨ ticket 列数字票根
--   (>=4 位连续数字——'mt5'/'self_state' 类非票根不收);双无=NULL
--   =单簇不折(无根票可证不并=防误判)。
--   ①簇销：后随 event='处置' 行引根票(note '#sig' 边界词 ∨
--     ticket 数字根同 sig)=簇全销——处置行引根票即簇销。
--   ②簇折：同簇签多未决行折叠为最新一行(ts DESC,id DESC),
--     valence=簇内 MAX(惧值账不丢——最强惧值随簇存活)。
--   既有未决口径(升件#id 引用∨升件#N-M 区间)原样保留在 base 层。
--   ops_event 行/销件语义零动——只动视图/判读层。

CREATE OR REPLACE VIEW ops_esc_pending AS
WITH base AS (
	SELECT e.id, e.ts, e.seat, e.esc_class, e.esc_scope, e.note,
		COALESCE(vs.weight, vc.weight, 50) AS valence,
		EXTRACT(EPOCH FROM (now()-e.ts))::int AS age_s,
		COALESCE(
			(regexp_match(e.note, '#([0-9]+)'))[1],
			CASE WHEN e.ticket ~ '[0-9]{4,}'
				THEN (regexp_match(e.ticket, '([0-9]{4,})'))[1] END
		) AS cluster_sig
	FROM ops_event e
	LEFT JOIN ops_esc_valence vs ON vs.esc_key = e.esc_class||'/'||e.esc_scope
	LEFT JOIN ops_esc_valence vc ON vc.esc_key = e.esc_class
	WHERE e.event='升件' AND e.ts > now()-interval '7 days'
	  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
	    AND x.note ~ ('升件#'||e.id||'([^0-9]|$)'))
	  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
	    AND x.note LIKE '%升件#%' AND EXISTS (
	      SELECT 1 FROM regexp_matches(x.note,'升件#(\d+)-(\d+)','g') AS m(mm)
	      WHERE e.id BETWEEN mm[1]::bigint AND mm[2]::bigint))
),
pend AS (
	SELECT b.* FROM base b
	WHERE b.cluster_sig IS NULL OR NOT EXISTS (
		SELECT 1 FROM ops_event x
		WHERE x.ts > b.ts AND x.event='处置'
		  AND (x.note ~ ('#' || b.cluster_sig || '([^0-9]|$)')
		       OR (x.ticket ~ '[0-9]{4,}'
		           AND (regexp_match(x.ticket, '([0-9]{4,})'))[1] = b.cluster_sig)))
),
folded AS (
	SELECT p.*,
		MAX(p.valence) OVER (
			PARTITION BY COALESCE(p.cluster_sig, 'id:'||p.id::text)
		) AS cval
	FROM pend p
),
latest AS (
	SELECT DISTINCT ON (COALESCE(cluster_sig, 'id:'||id::text))
		id, ts, seat, esc_class, esc_scope, note, age_s, cval
	FROM folded
	ORDER BY COALESCE(cluster_sig, 'id:'||id::text), ts DESC, id DESC
)
SELECT id, ts, seat, esc_class, esc_scope, note, cval AS valence, age_s
FROM latest;

COMMENT ON VIEW ops_esc_pending IS '未决升件+惧值+同根簇折叠(fin_0230 O-27914:簇签=note首个#num∨ticket数字根;处置行引根票=簇销;同簇折最新行valence=max;无签不折防误判)——议程排序真源(valence DESC,age DESC;dispatch/会话/自态帧三面同读)';
