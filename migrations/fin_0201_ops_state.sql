-- fin_0201: ops_state 经营状况状态表（减法B批——业主令#20930/#21264
-- 「经营状况→状态表」）。经营状况=持久可查行非事件流：会话入场
-- 扫一眼即知机器此刻哪里在烧/在堵/在旱，不靠升件叠火反复报。
-- bridge=t 类（经营面）由 dispatch 桥接升件（每况7d至多一火，
-- 况消自闭）；bridge=f 类（自管面）只入视面不唤会话——
-- D-0048④自管层减法，看自己的状况降可视件。
--
-- 口径=各探件原判据平移（平移非重写——产件 SQL 逐件镜抄
-- dispatch.sh/meta_esc.go 原判据，语义零漂）：
--   商业产出←meta_esc.go commercialEscCheck(liveN>0 && net<=0)
--   孤儿账×5←meta_esc.go orphanProbes TTL 钦定册
--   eval流断/产线静默/提炼旱←dispatch.sh 被替三探
--   票滞留/工单面压力←ops_ticket_desk（D-0048④ 可视件非唤件）
BEGIN;

CREATE VIEW ops_state AS
SELECT * FROM (
	-- 商业产出：30d 真净≤0 且有 live/probe 在管
	SELECT '商业产出'::text AS kind, 'commercial_stall'::text AS scope,
		60 AS valence, true AS bridge,
		(SELECT max(occurred_at) FROM fin_cashflow WHERE venue<>'sim') AS since,
		('30d净='||round(c.net,4)||' live+probe='||ln.n||' 在仓='||op.n
		 ||'——名义在赚实付净产出非正（Earn息不入账已剔）')::text AS detail
	FROM (SELECT COALESCE(sum(amount),0) net FROM fin_cashflow
		WHERE venue<>'sim' AND occurred_at>now()-interval '30 days'
		AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')) c,
		(SELECT count(*) n FROM fin_strategy WHERE state IN ('live','probe')) ln,
		(SELECT count(*) n FROM fin_position WHERE closed_at IS NULL) op
	WHERE ln.n>0 AND c.net<=0
	UNION ALL
	-- 孤儿账×5（orphanProbes 原判据平移）
	SELECT '孤儿账','orphan_eng_lesson',30,true,NULL::timestamptz,
		'fin_eng_lesson status=pending 积压 '||n||' 行超TTL24h'
	FROM (SELECT count(*) n FROM fin_eng_lesson
		WHERE status='pending' AND ts<now()-interval '24 hours') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_econ_lesson',30,true,NULL::timestamptz,
		'fin_econ_lesson status=pending 积压 '||n||' 行超TTL48h'
	FROM (SELECT count(*) n FROM fin_econ_lesson
		WHERE status='pending' AND ts<now()-interval '48 hours') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_hypothesis_proposed',30,true,NULL::timestamptz,
		'fin_opp_hypothesis proposed 积压 '||n||' 行超TTL7d'
	FROM (SELECT count(*) n FROM fin_opp_hypothesis
		WHERE state='proposed' AND ts<now()-interval '7 days') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_hypothesis_parked',30,true,NULL::timestamptz,
		'fin_opp_hypothesis parked 未复审 '||n||' 行超TTL30d'
	FROM (SELECT count(*) n FROM fin_opp_hypothesis
		WHERE state='parked' AND COALESCE(
			CASE WHEN evidence->>'last_reviewed' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
				THEN (evidence->>'last_reviewed')::timestamptz END, ts)
			< now()-interval '30 days') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_condition_gap',30,true,NULL::timestamptz,
		'fin_condition_gap pending 积压 '||n||' 行超TTL7d'
	FROM (SELECT count(*) n FROM fin_condition_gap
		WHERE state='pending' AND requested_at<now()-interval '7 days') x WHERE n>0
	UNION ALL
	-- eval流断：全模板零产出24h
	SELECT 'eval流断','evalflow_dead',55,true,
		(SELECT max(ts) FROM fin_opp_eval),
		'fin_opp_eval 全模板零产出24h——枚举臂死征'
	WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval WHERE ts>now()-interval '24 hours')
	UNION ALL
	-- 产线静默：有开放仓模板 eval 断3d
	SELECT '产线静默','lanesil_'||p.tmpl,55,true,NULL::timestamptz,
		p.tmpl||' 开放仓'||p.n||'张('||p.states||')在管但eval流断3d'
	FROM (SELECT s.template_ref tmpl, count(*) n, string_agg(DISTINCT s.state,'/') states
		FROM fin_strategy s WHERE s.state IN ('proposed','paper','live') GROUP BY 1) p
	WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval e
		WHERE e.template=p.tmpl AND e.ts>now()-interval '3 days')
	UNION ALL
	-- 提炼旱：verdicted≥50 且 canon 转化<5%
	SELECT '提炼旱','consol_dry_'||l.tbl,30,true,NULL::timestamptz,
		'fin_'||l.tbl||'_lesson verdicted='||l.v||'/canon='||l.c||'（转化<5%）'
	FROM (SELECT 'eng' tbl, count(*) FILTER (WHERE status='verdicted') v,
			count(*) FILTER (WHERE status='canon') c FROM fin_eng_lesson
		UNION ALL
		SELECT 'econ', count(*) FILTER (WHERE status='verdicted'),
			count(*) FILTER (WHERE status='canon' OR verdict_ref LIKE 'canon:%') FROM fin_econ_lesson) l
	WHERE l.v>=50 AND l.c*20<l.v
	UNION ALL
	-- 票滞留（自管面——入视不唤）
	SELECT '票滞留','ticketstall_'||d.id,20,false,d.last_active,
		'工单#'||d.id||' 零动账'||round(extract(epoch from now()-d.last_active)/3600,1)||'h '||substr(d.note,1,40)
	FROM ops_ticket_desk d
	WHERE d.workable AND d.last_active<now()-interval '6 hours'
	UNION ALL
	-- 工单面压力（自管面——入视不唤）
	SELECT '工单面压力','wip',20,false,now(),
		'窗内dispatchable='||q.da||' 窗外backlog='||q.bl||' 最陈'||q.old||'h'
	FROM (SELECT count(*) FILTER (WHERE dispatchable) da,
			count(*) FILTER (WHERE workable AND NOT dispatchable) bl,
			round(extract(epoch from now()-min(filed_ts) FILTER (WHERE workable))/3600,1) old
		FROM ops_ticket_desk) q
	WHERE q.da>=COALESCE((SELECT value::int FROM app_config WHERE key='dm.cog.wip_limit'),20) OR q.bl>0
) s;

-- 读面授权（fin_0182 同型——roster_rw=唤知取证角,o_fin_ro=只读备角）
DO $$
BEGIN
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='roster_rw') THEN
		EXECUTE 'GRANT SELECT ON ops_state TO roster_rw';
	END IF;
	IF EXISTS (SELECT FROM pg_roles WHERE rolname='o_fin_ro') THEN
		EXECUTE 'GRANT SELECT ON ops_state TO o_fin_ro';
	END IF;
END $$;

COMMIT;
