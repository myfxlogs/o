-- 0218 fin_position.state 物化列+转移闸（工单#25806 FSM 收口——派工单
-- fsm-converge-prompt.md=SSOT；开源架构审计#25771 裁定③adopt，僵尸单
-- #1117/#1134 脱钩 14 天实证，age-gate=补丁非根治）：
--   ①state 词表钉死最小集 open/closing/closed/orphaned（orphaned=execution
--     断腿后对账发现的孤儿持仓——探针可见态，非执行终态）；
--   ②BEFORE INSERT OR UPDATE 触发器单点收口（SQL 级挡选此钉死）：
--     closed_at 置位→state 恒随 'closed'（物化防漏写——closed_at 留
--     时间戳语义不再判读，读面全改 state<>'closed'）；UPDATE 转移合法集
--     显式枚举，非法迁移（closed→* 等）抛错=落账拒；
--   ③ops_state 视图+fin_active_set_maintain 函数重建——判读谓词
--     closed_at IS NULL→state<>'closed'（禁残留双判读源；视图/函数内
--     判读点随迁移同改，Go 读面同票全谱换）。

ALTER TABLE fin_position
    ADD COLUMN IF NOT EXISTS state text NOT NULL DEFAULT 'open';
COMMENT ON COLUMN fin_position.state IS
  'FSM 态（工单#25806）：open/closing/closed/orphaned——closed_at 置位恒随 closed（fin_position_fsm 触发器物化）；判开读面=state<>''closed''（未闭合全态，orphaned 非终态仍计敞口）';

-- 存量回填：closed_at 在=closed；余行 DEFAULT 'open' 已覆（两段式——
-- DEFAULT 先立再 UPDATE 校正，CHECK 两段式 NOT VALID→VALIDATE 在下）
UPDATE fin_position SET state='closed' WHERE closed_at IS NOT NULL;

ALTER TABLE fin_position
    ADD CONSTRAINT fin_position_state_chk
    CHECK (state IN ('open','closing','closed','orphaned')) NOT VALID;
ALTER TABLE fin_position VALIDATE CONSTRAINT fin_position_state_chk;

-- 转移闸+物化随动（单点收口——Go 写径零分散断言）：
--   · closed_at 置位=闭合事实锚，state 恒随 'closed'（INSERT/UPDATE 同律；
--     存量与测试径直写 closed_at 自愈归位，不另开双写口）；
--   · state='closed' 而 closed_at NULL=矛盾行禁立（闭合必带时标）；
--   · UPDATE 态迁合法集：open→{closing,closed,orphaned}、
--     closing→{closed,orphaned}、orphaned→{closed}；closed→*=落账拒；
--   · 同态 UPDATE（state 列不动）自由过——settled/bare_bag 等旁路写不受阻。
CREATE OR REPLACE FUNCTION fin_position_fsm() RETURNS trigger AS $$
BEGIN
    -- ①转移合法性先判（NEW.state 按写入值——closed→* 等非法边显式抛错，
    --   不被 closed_at 随动静默吞掉）
    IF TG_OP = 'UPDATE' AND NEW.state IS DISTINCT FROM OLD.state THEN
        IF NOT (
            (OLD.state = 'open'     AND NEW.state IN ('closing','closed','orphaned')) OR
            (OLD.state = 'closing'  AND NEW.state IN ('closed','orphaned'))         OR
            (OLD.state = 'orphaned' AND NEW.state = 'closed')
        ) THEN
            RAISE EXCEPTION 'fin_position state 转移非法: % -> % (id=%)',
                OLD.state, NEW.state, OLD.id;
        END IF;
    END IF;
    -- ②closed_at 置位=闭合事实锚——state 恒随 'closed'（INSERT/UPDATE 同律；
    --   存量与测试径直写 closed_at 自愈归位，不另开双写口）
    IF NEW.closed_at IS NOT NULL THEN
        NEW.state := 'closed';
    END IF;
    -- ③state='closed' 而 closed_at NULL=矛盾行禁立（闭合必带时标）
    IF NEW.state = 'closed' AND NEW.closed_at IS NULL THEN
        RAISE EXCEPTION 'fin_position state=closed 而 closed_at NULL——闭合必带时标 (id=%)',
            COALESCE(NEW.id, 0);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER fin_position_fsm_trg
    BEFORE INSERT OR UPDATE ON fin_position
    FOR EACH ROW EXECUTE FUNCTION fin_position_fsm();

-- ops_state 视图重建（fin_0205 全文同列序，唯一改动=「在仓」计数谓词
-- closed_at IS NULL→state<>'closed'——单源化后判读不再走时标列）。
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

-- fin_active_set_maintain 重建（fin_0069 同函数体，唯一改动=活跃判据
-- p.closed_at IS NULL→p.state<>'closed'——单源化；触发器本体不动，
-- state 列变更随 UPDATE 触发器自动重算入集）。
CREATE OR REPLACE FUNCTION fin_active_set_maintain() RETURNS trigger AS $$
DECLARE
    inst text;
BEGIN
    IF TG_OP = 'DELETE' THEN
        inst := OLD.inst_id;
    ELSE
        inst := NEW.inst_id;
    END IF;
    IF EXISTS (SELECT 1 FROM fin_position p
                 WHERE p.inst_id = inst AND p.state <> 'closed')
       OR EXISTS (SELECT 1 FROM fin_strategy s
                    WHERE s.inst_id = inst AND s.state <> 'retired') THEN
        INSERT INTO fin_active_set(inst_id) VALUES (inst)
            ON CONFLICT (inst_id) DO UPDATE SET updated_at = now();
    ELSE
        DELETE FROM fin_active_set WHERE inst_id = inst;
    END IF;
    PERFORM pg_notify('ev_active_set', '');
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- 0069 种子回填同改谓词重跑一遍（幂等——ON CONFLICT 兜；存量行经
-- 回填后 closed_at↔state 已恒等，重跑结果应与现集一致，跑它=对账自证）。
INSERT INTO fin_active_set(inst_id)
SELECT inst_id FROM fin_position WHERE state <> 'closed'
UNION SELECT inst_id FROM fin_strategy WHERE state <> 'retired'
ON CONFLICT (inst_id) DO NOTHING;

-- fin_position_view 不重建：该视图纯列透传（closed_at 列照旧出）不含
-- 判读谓词——Go 消费面本票改直查 fin_position，视图列语义不动。

-- fin_position 表级授权已在 fin_0106 在册（列级继承表授权，零新增 GRANT）。

COMMIT;
