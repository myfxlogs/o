-- fin_0205: ops_state 票滞留面两臂补齐（工单#19193+#19738 同根合迁——
-- 同一 SELECT 行双病灶一次修）：
--   ① #19193 挂起窗抑臂缺：d.workable 对 susp_cls='capacity' 挂起窗内
--      票仍放行（desk 视 workalbe 谓词本就含窗内 capacity 挂），滞留
--      探对「明示挂起待产能」票每拍复火=假滞留。补 susp_until 窗检：
--      挂起窗内票不入 ticketstall（窗期自然到点回列复验）。
--   ② #19738 裁载缺「验回执」一裁：票有未验完工回执时滞留 note 无
--      「唤知验收道入径」提示——detail 加 act 层判（receipt 在 ∧ close
--      不在=未验回执在案）则载「|有未验完工回执→裁=验回执(唤知道)」。
-- 视图整体重建（CREATE OR REPLACE 同列序，fin_0202 面其余行原样）。

BEGIN;

CREATE OR REPLACE VIEW ops_state AS
SELECT * FROM (
	-- 商业产出：30d 真净≤0 且有 live/probe 在管
	SELECT '商业产出'::text AS kind, 'commercial_stall'::text AS scope,
		60 AS valence,
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
	-- live 哑弹：live≥7d 零仓位（supply_esc.go liveDudEscCheck 判据平移——n/样本同限8条口径）
	SELECT '商业产出','live_dud',55,NULL::timestamptz,
		'live≥7d 零仓位哑弹策略 '||q.n||' 条('||COALESCE(q.sample,'')||')'
	FROM (SELECT count(*) n, string_agg('#'||s.id||' '||s.inst_id,'、' ORDER BY s.entered_at) sample
		FROM (SELECT s.id, s.inst_id, s.entered_at FROM fin_strategy s
			WHERE s.state='live' AND s.entered_at<now()-interval '7 days'
			AND NOT EXISTS (SELECT 1 FROM fin_position fp WHERE fp.strategy_ref=s.id)
			ORDER BY s.entered_at LIMIT 8) s) q
	WHERE q.n>0
	UNION ALL
	-- 资本闲置：live 在册 ∧ 需额<25% 净值（deployEscCheck 判据平移；cap_hit 闸到顶仍走升件不在此面）
	SELECT '资金闸','idle_capital',45,NULL::timestamptz,
		'净值='||round(q.eq::numeric,0)||' 需额='||round(q.d::numeric,0)||'(live×'||q.ln
		 ||'+probe) 利用率='||round((100*q.d/nullif(q.exp,0))::numeric,0)||'%——部署不足净值1/4'
	FROM (SELECT (SELECT equity FROM md_equity_curve WHERE ccy='NAV'
			AND ts>now()-interval '48 hours' ORDER BY ts DESC LIMIT 1) eq,
		(SELECT coalesce(sum(notional_usd),0) FROM fin_strategy WHERE state IN ('live','probe')) d,
		(SELECT count(*) FROM fin_strategy WHERE state='live') ln,
		(SELECT COALESCE(value::float8,0) FROM app_config WHERE key='risk.max_exposure_usd') exp) q
	WHERE q.eq>0 AND q.ln>0 AND q.d<0.25*q.eq
	UNION ALL
	-- 孤儿账×5（orphanProbes 原判据平移）
	SELECT '孤儿账','orphan_eng_lesson',30,NULL::timestamptz,
		'fin_eng_lesson status=pending 积压 '||n||' 行超TTL24h'
	FROM (SELECT count(*) n FROM fin_eng_lesson
		WHERE status='pending' AND ts<now()-interval '24 hours') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_econ_lesson',30,NULL::timestamptz,
		'fin_econ_lesson status=pending 积压 '||n||' 行超TTL48h'
	FROM (SELECT count(*) n FROM fin_econ_lesson
		WHERE status='pending' AND ts<now()-interval '48 hours') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_hypothesis_proposed',30,NULL::timestamptz,
		'fin_opp_hypothesis proposed 积压 '||n||' 行超TTL7d'
	FROM (SELECT count(*) n FROM fin_opp_hypothesis
		WHERE state='proposed' AND ts<now()-interval '7 days') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_hypothesis_parked',30,NULL::timestamptz,
		'fin_opp_hypothesis parked 未复审 '||n||' 行超TTL30d'
	FROM (SELECT count(*) n FROM fin_opp_hypothesis
		WHERE state='parked' AND COALESCE(
			CASE WHEN evidence->>'last_reviewed' ~ '^[0-9]{4}-[0-9]{2}-[0-9]{2}'
				THEN (evidence->>'last_reviewed')::timestamptz END, ts)
			< now()-interval '30 days') x WHERE n>0
	UNION ALL
	SELECT '孤儿账','orphan_condition_gap',30,NULL::timestamptz,
		'fin_condition_gap pending 积压 '||n||' 行超TTL7d'
	FROM (SELECT count(*) n FROM fin_condition_gap
		WHERE state='pending' AND requested_at<now()-interval '7 days') x WHERE n>0
	UNION ALL
	-- eval流断：全模板零产出24h
	SELECT 'eval流断','evalflow_dead',55,
		(SELECT max(ts) FROM fin_opp_eval),
		'fin_opp_eval 全模板零产出24h——枚举臂死征'
	WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval WHERE ts>now()-interval '24 hours')
	UNION ALL
	-- 产线静默：有开放仓模板 eval 断3d
	SELECT '产线静默','lanesil_'||p.tmpl,55,NULL::timestamptz,
		p.tmpl||' 开放仓'||p.n||'张('||p.states||')在管但eval流断3d'
	FROM (SELECT s.template_ref tmpl, count(*) n, string_agg(DISTINCT s.state,'/') states
		FROM fin_strategy s WHERE s.state IN ('proposed','paper','live') GROUP BY 1) p
	WHERE NOT EXISTS (SELECT 1 FROM fin_opp_eval e
		WHERE e.template=p.tmpl AND e.ts>now()-interval '3 days')
	UNION ALL
	-- 提炼旱：verdicted≥50 且 canon 转化<5%
	SELECT '提炼旱','consol_dry_'||l.tbl,30,NULL::timestamptz,
		'fin_'||l.tbl||'_lesson verdicted='||l.v||'/canon='||l.c||'（转化<5%）'
	FROM (SELECT 'eng' tbl, count(*) FILTER (WHERE status='verdicted') v,
			count(*) FILTER (WHERE status='canon') c FROM fin_eng_lesson
		UNION ALL
		SELECT 'econ', count(*) FILTER (WHERE status='verdicted'),
			count(*) FILTER (WHERE status='canon' OR verdict_ref LIKE 'canon:%') FROM fin_econ_lesson) l
	WHERE l.v>=50 AND l.c*20<l.v
	UNION ALL
	-- 机会面断供：曾活模板族 48h 零新评估（oppSurfaceStallEscCheck 判据平移——
	-- 30d≥20 评估行=曾活 ∧ md_eval_pulse.last_eval_ts 陈>48h）
	SELECT '数据缺','opp_eval_stall',50,NULL::timestamptz,
		'曾活模板族 '||q.n||' 支 48h 零评估('||q.tpls||')'
	FROM (SELECT count(*) n, string_agg(template,'、' ORDER BY last_eval_ts) tpls
		FROM (SELECT e.template, p.last_eval_ts FROM fin_opp_eval e
			JOIN md_eval_pulse p ON p.template=e.template
			GROUP BY e.template, p.last_eval_ts
			HAVING count(*) FILTER (WHERE e.ts>now()-interval '30 days')>=20
			AND p.last_eval_ts<now()-interval '48 hours') x) q
	WHERE q.n>0
	UNION ALL
	-- 校准失真：族预测-兑现系统性失真（calibDriftEscCheck 判据平移——n≥10 ∧ |pred−real|>5pp）
	SELECT '世模','calib_drift',55,NULL::timestamptz,
		q.n||' 族预测-兑现系统性失真('||q.tpls||')'
	FROM (SELECT count(*) n, string_agg(x.template_ref,'、' ORDER BY abs(x.pred-x.real) DESC) tpls
		FROM (SELECT template_ref,n,pred,real FROM (
			SELECT s.template_ref,
				count(*) FILTER (WHERE e.ts<=pr.created_at) n,
				avg(e.net_apr) FILTER (WHERE e.ts<=pr.created_at) pred,
				avg((COALESCE(cf.net,0)+COALESCE(pf.net,0))/NULLIF(s.notional_usd,0)
					/GREATEST(EXTRACT(EPOCH FROM(now()-COALESCE(s.entered_at,s.created_at)))/86400,1)*365)
					FILTER (WHERE e.ts<=pr.created_at) real
			FROM fin_strategy s
			JOIN fin_proposal pr ON pr.id=s.proposal_ref
			JOIN fin_opp_eval e ON e.id=pr.eval_id
			LEFT JOIN (SELECT p.strategy_ref,sum(c.amount) net FROM fin_position p
				JOIN fin_cashflow c ON c.position_ref=p.id GROUP BY 1) cf
				ON cf.strategy_ref=s.id
			LEFT JOIN (SELECT f.strategy_id,sum(f.amount_usd) net
				FROM fin_paper_fill f GROUP BY 1) pf ON pf.strategy_id=s.id
			WHERE s.template_ref NOT LIKE 'simdrill%' AND e.net_apr IS NOT NULL
				AND (cf.net IS NOT NULL OR pf.net IS NOT NULL)
			GROUP BY 1) t
		WHERE n>=10 AND abs(pred-real)>0.05
		ORDER BY abs(pred-real) DESC LIMIT 5) x) q
	WHERE q.n>0
	UNION ALL
	-- 票滞留（自管面——入视不唤；#19193①:挂起窗内票抑——workable 谓词
	-- 对 capacity 挂起窗内放行,此处补 susp_until 窗检;#19738②:有未验
	-- 完工回执(receipt∧¬close)载「验回执」裁径——act 层判不猜散文）
	SELECT '票滞留','ticketstall_'||d.id,20,d.last_active,
		'工单#'||d.id||' 零动账'||round(extract(epoch from now()-d.last_active)/3600,1)||'h'
		||CASE WHEN EXISTS(SELECT 1 FROM ops_event_act ra
				WHERE ra.rel=d.id AND ra.act='receipt')
			AND NOT EXISTS(SELECT 1 FROM ops_event_act ca
				WHERE ca.rel=d.id AND ca.act='close')
			THEN ' |有未验完工回执→裁=验回执(唤知道)' ELSE '' END
		||' '||substr(d.note,1,40)
	FROM ops_ticket_desk d
	WHERE d.workable AND d.last_active<now()-interval '6 hours'
		AND (d.susp_until IS NULL OR d.susp_until<=now())
	UNION ALL
	-- 工单面压力（自管面——入视不唤）
	SELECT '工单面压力','wip',20,now(),
		'窗内dispatchable='||q.da||' 窗外backlog='||q.bl||' 最陈'||q.old||'h'
	FROM (SELECT count(*) FILTER (WHERE dispatchable) da,
			count(*) FILTER (WHERE workable AND NOT dispatchable) bl,
			round(extract(epoch from now()-min(filed_ts) FILTER (WHERE workable))/3600,1) old
		FROM ops_ticket_desk) q
	WHERE q.da>=COALESCE((SELECT value::int FROM app_config WHERE key='dm.cog.wip_limit'),20) OR q.bl>0
) s
UNION ALL
-- 产件写行面（Go 侧不可 SQL 化判据——capability_* 等；变了才更新一行）
SELECT kind, scope, valence, since, detail FROM ops_state_row;

-- 读面授权幂等保持（OR REPLACE 保授权本不丢，块留作显式约定）
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
