-- fin_0245 闲置资金机会成本语义化（工单O-36962——业主令「钱闲着」现行
-- 口径无机会成本基线：live_drought 判词 #36202「有理闲置」只说无件过闸，
-- 没载钱停在哪收多少息；候选 net_apr 比较缺停泊基线参照）：
--   ① fin_park_baseline 视图——停泊基线单读面（Earn 实收息率+DCD 在仓），
--     livedrought 归因卡/ops_state idle_capital 臂/判词层同吃一源；
--   ② app_config 键：annualized.park_yield_source='fin_cashflow_interest'
--     （停泊收益数据源标识——实测在册读面=fin_cashflow kind='interest'
--     窗累计÷md_account_events earn_parking 在停本金，生产核实无
--     fin_earn_position 表故不取虚构面）；annualized.risk_premium=0.04
--     （机会成本风险溢价——初值取判据卡在册数：纪元2 middle-band 产线档
--     「跑赢停泊+风险溢价」建议≥4%，建议级唤知可调）；
--   ③ ops_state idle_capital 臂 detail 追加停泊基线段（视图体=fin_0239
--     正本逐字，唯一 delta=该臂 SELECT 列表与 FROM 尾——判据谓词零触）；
--   ④ ops_esc_valence idle_verdict_gate 类登记（判词质量闸探针升件惧值）。
-- 读面实证（2026-11-24 生产对拍）：earn_parking earn_bal=9385.66USDT、
-- fin_cashflow interest 577 行（okx/USDT md_account_events 源）30d 累计
-- 13.64、md_earn_rate 现率 0.0353（ offered 率非本视图口径——实收才是
-- 判词基线）。

BEGIN;

-- ① 停泊基线单读面（fin 域——md_account_events/fin_cashflow 皆 fin 域
-- 正表/桥内表，零跨链读）。单行基线：earn_principal=earn_parking 末行
-- earn_bal（事件锚定快照——缺席=NULL 非 0）；earn_apy=近30d 实收息
-- ÷在停本金×12 年化（本金视作恒定=近似口径，判词层如实标注）；
-- interest_last_ts 载息流鲜度（断流可辨）；dcd_*=dcd_live 末行在仓聚合。
CREATE OR REPLACE VIEW fin_park_baseline AS
WITH earn_snap AS (
	SELECT (e.data->>'earn_bal')::numeric AS principal, e.ts AS snap_ts
	FROM md_account_events e
	WHERE e.channel='earn_parking' AND e.venue='okx' AND e.ccy='USDT'
	ORDER BY e.ts DESC LIMIT 1
), earn_int AS (
	SELECT coalesce(sum(c.amount),0) AS amt_30d, max(c.occurred_at) AS last_ts
	FROM fin_cashflow c
	WHERE c.kind='interest' AND c.venue='okx' AND c.ccy='USDT'
	  AND c.occurred_at > now()-interval '30 days'
), dcd_live AS (
	SELECT e.data FROM md_account_events e
	WHERE e.channel='dcd_live' AND e.venue='okx'
	ORDER BY e.ts DESC LIMIT 1
), dcd_agg AS (
	SELECT count(*) AS legs,
		coalesce(sum((o->>'notionalSz')::numeric)
			FILTER (WHERE o->>'notionalCcy'='USDT'),0) AS usdt
	FROM dcd_live d, jsonb_array_elements(d.data->'orders') o
)
SELECT
	(SELECT principal FROM earn_snap) AS earn_principal,
	(SELECT snap_ts FROM earn_snap) AS earn_snap_ts,
	(SELECT amt_30d FROM earn_int) AS interest_30d,
	(SELECT last_ts FROM earn_int) AS interest_last_ts,
	CASE WHEN (SELECT principal FROM earn_snap) > 0
		THEN round(((SELECT amt_30d FROM earn_int)
			/ (SELECT principal FROM earn_snap) * 12)::numeric,6)
	END AS earn_apy,
	(SELECT legs FROM dcd_agg) AS dcd_legs,
	(SELECT usdt FROM dcd_agg) AS dcd_usdt;

COMMENT ON VIEW fin_park_baseline IS
	'O-36962 停泊基线单读面——Earn 在停本金(earn_parking末行)+近30d实收息率(annualized)'
	'+息流鲜度+DCD在仓聚合；earn_principal NULL=快照缺席 fail-closed 非零。'
	'判词层三处共吃：livedrought归因卡/ops_state idle_capital臂/唤知判词参照';

-- ② 配置键（park_yield_source=读面标识非收益率数——实测在册面
-- fin_cashflow_interest；risk_premium=机会成本溢价分数实）。
INSERT INTO app_config(key,value,updated_by) VALUES
  ('annualized.park_yield_source','"fin_cashflow_interest"','fin_0245'),
  ('annualized.risk_premium','0.04','fin_0245')
ON CONFLICT (key) DO NOTHING;

-- ③ idle_verdict_gate 判词质量闸升件惧值（处置判词缺停泊基线=不合格
-- 件再升——探针只报不修，权重中位：判词瑕疵非钱径断流）。
INSERT INTO ops_esc_valence(esc_key,weight,note) VALUES
  ('idle_verdict_gate/判词质量',65,'有理闲置判词缺停泊基线=判词不合格（O-36962必备字段闸）')
ON CONFLICT (esc_key) DO UPDATE SET weight=EXCLUDED.weight, note=EXCLUDED.note;

-- ④ ops_state idle_capital 臂停泊基线段（视图体=fin_0239 正本逐字，
-- 唯一 delta=该臂 SELECT 列表尾追加 COALESCE 停泊段 + FROM 尾加
-- fin_park_baseline/app_config 参照——判据谓词 WHERE 零触）。
CREATE OR REPLACE VIEW ops_state AS
SELECT * FROM (
	-- 商业产出：30d 真净≤0 且有 live/probe 在管
	SELECT '商业产出'::text AS kind, 'commercial_stall'::text AS scope,
		60 AS valence,
		(SELECT max(occurred_at) FROM fin_cashflow WHERE venue<>'sim') AS since,
		('30d净='||round(c.net,4)||' live+probe='||ln.n||' 在仓='||op.n
		 ||'——名义在赚实付净产出非正（Earn息不入账已剔）')::text AS detail
	FROM (SELECT COALESCE(sum(amount),0) net FROM fin_pnl_net
		WHERE occurred_at>now()-interval '30 days') c,
		(SELECT count(*) n FROM fin_strategy WHERE state IN ('live','probe')) ln,
		(SELECT count(*) n FROM fin_position WHERE state<>'closed') op
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
	-- O-36962：停泊基线段——钱停泊≠闲着（实收息率+DCD在仓+机会成本参照系）
	SELECT '资金闸','idle_capital',45,NULL::timestamptz,
		('净值='||round(q.eq::numeric,0)||' 需额='||round(q.d::numeric,0)||'(live×'||q.ln
		 ||'+probe) 利用率='||round((100*q.d/nullif(q.exp,0))::numeric,0)||'%——部署不足净值1/4'
		 ||COALESCE('｜停泊=Earn '||round(pb.earn_principal::numeric,2)||'USDT@'
			||round((pb.earn_apy*100)::numeric,2)||'%/yr(实收30d)+DCD '
			||pb.dcd_legs||'笔/'||round(pb.dcd_usdt::numeric,0)||'USDT'
			||'｜机会成本=net_apr≥停泊'||round((pb.earn_apy*100)::numeric,2)
			||'%+溢价'||round((COALESCE((SELECT value::float8 FROM app_config
				WHERE key='annualized.risk_premium'),0)*100)::numeric,2)||'%',
		 '｜停泊基线缺（earn_parking快照缺席——Earn腿不可判'
		 ||'，DCD在仓='||pb.dcd_legs||'笔/'||round(pb.dcd_usdt::numeric,0)
		 ||'USDT仍载——判词层不得按0载）')
		)::text AS detail
	FROM (SELECT (SELECT equity FROM md_equity_curve WHERE ccy='NAV'
			AND ts>now()-interval '48 hours' ORDER BY ts DESC LIMIT 1) eq,
		(SELECT coalesce(sum(notional_usd),0) FROM fin_strategy WHERE state IN ('live','probe')) d,
		(SELECT count(*) FROM fin_strategy WHERE state='live') ln,
		(SELECT COALESCE(value::float8,0) FROM app_config WHERE key='risk.max_exposure_usd') exp) q
	CROSS JOIN fin_park_baseline pb
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
			count(*) FILTER (WHERE workable AND dispatchable IS NOT TRUE) bl,
			round(extract(epoch from now()-min(filed_ts) FILTER (WHERE workable))/3600,1) old
		FROM ops_ticket_desk) q
	WHERE q.da>=COALESCE((SELECT value::int FROM app_config WHERE key='dm.cog.wip_limit'),20) OR q.bl>0
) s;

COMMIT;
