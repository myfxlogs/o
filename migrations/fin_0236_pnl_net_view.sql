-- fin_0236: 净产出口径单源视图 fin_pnl_net + ops_state commercial_stall
-- 臂改吃视图（工单#34619 pnl-phantom-clear-leg）：
--   病灶=position_ref NULL 现货腿（残仓清仓卖/业主手工转换/native 币场
--   腿）单边计益——TRX 卖腿 +9999.08 入账而取得腿记 recon_adjust 不入
--   kind 集，30d净幻影 +9997.30（年化1125%）。三次全史清仓
--   （ETC+2.57/VVV+24.24/TRX+9999.08）position_ref 全 NULL 且无对应
--   取得腿入册——均「无取得腿清仓卖出」形。
--   判别规则（施工定——取舍入回执）：position_ref NULL 的 spot_buy/
--   spot_sell=账外资产转换腿（托管/手工/清仓残仓/未授权持仓变现），
--   非策略损益——两腿同口径取「都不计」。例外=配对豁免（币种级
--   精配）：NULL 腿若 source=fin_risk_decision，其 intent.ccy 与
--   在册仓腿（position_ref 非空,source=fin_order_leg）的
--   split_part(inst_id,'-',1) 同币、反向符号、同 venue/ccy、
--   ±120s 内——=仓腿被误记成账外腿（residual_clear TOCTOU 扫新开
--   对冲腿币实证 VVV#1525↔仓86买腿#1524/order_leg#272 inst
--   VVV-USDT），按仓腿同口径计入保两腿不拆（只计一腿=违 spec
--   核律）。币种级配对防「同窗异币」误吸——amount 容差不设
--   （部分扫残仓场景 amount 失配仍须配对）；非 decision 源的
--   NULL 腿拿不到币种锚=无法证配对应排除（fail-safe 低估向）。
--   无配对 NULL 腿=真账外变现（ETC#731/TRX#1621）排除；取得腿
--   若记 recon_adjust 本不入 kind 集——两腿同口径自然成立。
--   消费面单源：盈利报告(owner_beat)/饿探(hunger_esc)/自观账
--   (selfaware_beat)/commercial_stall(ops_state)/死刑梯度年化
--   (death_grad)/账房对账 earned 桶(books_beat)/策略归因
--   (pnl_attr) 全走本视图——同口径求和不再各抄 kind 集；
--   selfaware 原抄本用 ts 列（余三面用 occurred_at）——漂移一并归正。
--   残清径不改留痕形：算式侧按 position_ref 判别覆盖一切账外现货
--   源（residual_clear/venue_bill/手工转换同闸），改 afterClear 只盖
--   单源不治本。


-- fin_pnl_net 净产出口径单源视图：venue<>'sim' ∧ 损益六 kind ∧
-- USDT/USD 本位域 ∧ position_ref NULL 现货腿排除（账外资产转换
-- 不计业绩——两腿同口径都不计）。
CREATE OR REPLACE VIEW fin_pnl_net AS
SELECT occurred_at, amount, kind, ccy, venue, position_ref
FROM fin_cashflow f
WHERE venue<>'sim'
  AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')
  AND ccy IN ('USDT','USD')
  AND NOT (
    kind IN ('spot_buy','spot_sell') AND position_ref IS NULL
    AND NOT EXISTS (
      SELECT 1 FROM fin_cashflow m
      JOIN fin_order_leg ol
        ON m.source_table='fin_order_leg' AND ol.id=m.source_ref
      JOIN fin_risk_decision d
        ON f.source_table='fin_risk_decision' AND d.id=f.source_ref
      WHERE m.position_ref IS NOT NULL
        AND m.kind IN ('spot_buy','spot_sell')
        AND m.venue=f.venue AND m.ccy=f.ccy
        AND sign(m.amount)=-sign(f.amount)
        AND m.occurred_at BETWEEN f.occurred_at-interval '120 seconds'
                            AND f.occurred_at+interval '120 seconds'
        AND split_part(ol.inst_id,'-',1)=d.intent->>'ccy'));

COMMENT ON VIEW fin_pnl_net IS '净产出口径单源（工单#34619）：损益六kind∧USDT/USD域∧position_ref NULL现货腿排除（decision源NULL腿±120s同币种反向在册仓腿配对豁免=误记仓腿）——盈利报告/饿探/自观账/commercial_stall/死刑梯度/账房对账/策略归因同吃本视图';

-- ops_state 重建：fin_0218 全文同列序，唯一改动=commercial_stall 臂
-- 内层求和改吃 fin_pnl_net（顺带补 ccy 闭集——native-ccy 后 BTC 域
-- 腿混入 USDT 求和的单位漂移根治）。
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

