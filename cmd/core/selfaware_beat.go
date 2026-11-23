// selfAwareBeat 自我意识器官拍（工单#13540 业主令器官——
// spec=docs/drafts/organ-selfaware-spec.md）。本体觉三问帧：
// 赚（NAV/30d净/仓源可证）+动（常驻器官末行 ts 对拍节判 动/静/僵）
// +息（间歇器官基态——静息=设计态非僵）+办（在办/认领未配对行）
// +卡（心跳陈旧/唤起无果/部署滞差/工单过期 显式判读旗集）。
// 每段读败如实 '?' 不整帧弃；INSERT 败=本帧让位静默（量表惯例）。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const selfAwareEvery = 6 * time.Hour

// organPulse 常驻器官册——名+脉源+静/僵双阈；新增常驻件=册内添行禁散装。
var organPulses = []struct {
	name        string
	sql         string
	quiet, dead time.Duration
}{
	{"收", `SELECT max(ts) FROM md_ticks WHERE venue<>'sim'`, time.Minute, 5 * time.Minute},
	{"投", `SELECT max(ts) FROM md_signal`, 5 * time.Minute, 20 * time.Minute},
	{"评", `SELECT max(ts) FROM fin_opp_eval`, 15 * time.Minute, time.Hour},
	{"态", `SELECT max(ts) FROM ops_event WHERE ticket='self_state'`, 7 * time.Hour, 13 * time.Hour},
	{"账", `SELECT max(ts) FROM ops_event WHERE ticket='books_daily'`, 22 * time.Hour, 44 * time.Hour},
	{"监", `SELECT max(ts) FROM ops_event WHERE event='监账'`, 5 * time.Hour, 12 * time.Hour},
}

// organIdle 间歇器官册——事件驱动件：近窗产出=动，余=息（静息合法非僵）。
var organIdle = []struct{ name, sql string }{
	{"做梦", `SELECT max(ts) FROM ops_event WHERE ticket='dream_replay'`},
	{"懂你", `SELECT max(ts) FROM ops_event WHERE ticket='owner_model'`}, // A2:器官产出在 owner_model——owner_intent=静态十键脉源插错线
	{"连续", `SELECT max(ts) FROM ops_event WHERE ticket='self_narrative'`},
	{"研究", `SELECT max(ts) FROM ops_event WHERE seat='研究员' AND event='能力面扫描'`},
	{"部署", `SELECT max(ts) FROM ops_event WHERE seat='DEPLOY' AND event='部署' AND note LIKE '%o-core:%'`},
}

// pulseState 常驻件判态：读败=?，age≤quiet=动，≤dead=静，>dead=僵。
func pulseState(ts *time.Time, err error, quiet, dead time.Duration) string {
	if err != nil || ts == nil {
		return "?"
	}
	a := time.Since(*ts)
	switch {
	case a <= quiet:
		return "动"
	case a <= dead:
		return "静"
	default:
		return "僵"
	}
}

func (c coreService) selfAwareBeat(ctx context.Context) {
	c.selfModelBeat(ctx) // 自模账帧同拍（自我意识帧组——能力账翼;独立自闸 20h;feed_esc 棘轮满线不入册唤点）
	var earn, move, idle, doing, stuck string

	// ① 赚：NAV 末行+陈 | 30d 净产出（commercial_stall 同径——
	//    venue<>'sim' 损益 kind 集,Earn 息已剔）| live/probe/仓。
	var nav *float64
	var navTs *time.Time
	var net float64
	var liveN, probeN, posN int
	_ = c.pool.QueryRow(ctx, `SELECT equity,ts FROM md_equity_curve
		WHERE ccy='NAV' ORDER BY ts DESC LIMIT 1`).Scan(&nav, &navTs)
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_pnl_net
		WHERE occurred_at>now()-interval '30 days' `).Scan(&net)
	_ = c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE state='live'),
		count(*) FILTER (WHERE state='probe'),
		(SELECT count(*) FROM fin_position WHERE state<>'closed')
		FROM fin_strategy`).Scan(&liveN, &probeN, &posN)
	if nav != nil && navTs != nil && *nav > 0 {
		earn = fmt.Sprintf("NAV%.0f(%s) 30d净%+.2f 年化%+.1f%%vs%.0f%% live%d/probe%d/仓%d",
			*nav, time.Since(*navTs).Round(time.Minute), net,
			net*365/30 / *nav * 100, hungerTargetPct, liveN, probeN, posN)
		earn += " |钱在:" + c.moneyDist(ctx, *nav)
	} else {
		earn = "?"
	}

	// ② 动：常驻器官册逐件判态（僵件名顺带进卡旗）。
	//   mvFp=结构纹件——只载名:态,年龄属浮点噪声不触纹。
	var mv, mvFp, dead []string
	for _, o := range organPulses {
		var ts *time.Time
		err := c.pool.QueryRow(ctx, o.sql).Scan(&ts)
		st := pulseState(ts, err, o.quiet, o.dead)
		age := "?"
		if ts != nil {
			age = time.Since(*ts).Round(time.Minute).String()
		}
		mv = append(mv, fmt.Sprintf("%s:%s(%s)", o.name, st, age))
		mvFp = append(mvFp, o.name+":"+st)
		if st == "僵" || st == "?" {
			dead = append(dead, o.name)
		}
	}
	move = strings.Join(mv, " ")

	// ③ 息：间歇器官册——24h 内产出=动，余=息列名（静息合法）。
	var idl, act []string
	for _, o := range organIdle {
		var ts *time.Time
		if err := c.pool.QueryRow(ctx, o.sql).Scan(&ts); err == nil && ts != nil &&
			time.Since(*ts) < 24*time.Hour {
			act = append(act, o.name)
		} else {
			idl = append(idl, o.name)
		}
	}
	if len(act) > 0 {
		idle = fmt.Sprintf("动[%s] 息[%s]", strings.Join(act, ","), strings.Join(idl, ","))
	} else {
		idle = "息[" + strings.Join(idl, ",") + "]"
	}

	// ④ 办：未配对 在办/认领 行（2h TTL 同认领律——陈行转卡旗不计办；
	//   A3 配对契同 runbook ⓪ 未完在办口径——认领行对象引证(工单#∨
	//   升件#)被后场处置引证=出场;在办行 unit 名被收口/果/败行引证∨
	//   处置引 在办#id 销账=出场;配对过的行不计办——已完场的活不显在办）。
	var doN int
	var doFirst *string
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(note) FROM (
		SELECT e.note FROM ops_event e WHERE e.event='认领' AND e.ts>now()-interval '2 hours'
		  AND (substring(e.note from '[工单升]件#[0-9]+') IS NULL
		    OR NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置' AND d.ts>e.ts
		      AND d.note ~ (substring(e.note from '[工单升]件#[0-9]+')||'([^0-9]|$)')))
		UNION ALL
		SELECT e.note FROM ops_event e WHERE e.event='在办' AND e.ts>now()-interval '2 hours'
		  AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
		    AND x.note LIKE '%'||e.ticket||'%'
		    AND x.event IN ('会话收口','唤起果','唤起败'))
		  AND NOT EXISTS (SELECT 1 FROM ops_event d WHERE d.event='处置' AND d.ts>e.ts
		    AND d.note ~ ('在办#'||e.id::text||'([^0-9]|$)'))) p`).
		Scan(&doN, &doFirst); err == nil {
		if doN == 0 {
			doing = "0"
		} else {
			first := ""
			if doFirst != nil {
				first = " " + truncRunes(*doFirst, 24)
			}
			doing = fmt.Sprintf("%d%s", doN, first)
		}
	} else {
		doing = "?"
	}

	// ⑤ 卡：显式判读旗集——各旗独立源（心跳陈旧/唤起无果/部署滞差/
	//    工单过期/器官僵）；读败段如实标 ?。
	//   fpFlags=结构纹件——旗名不带龄/数,数值漂不触纹。
	var flags, fpFlags []string
	flag := func(name, disp string) { flags = append(flags, disp); fpFlags = append(fpFlags, name) }
	var lastClose *time.Time
	if err := c.pool.QueryRow(ctx, `SELECT max(ts) FROM ops_event
		WHERE event='会话收口'`).Scan(&lastClose); err != nil {
		flag("收口?", "收口?")
	} else if lastClose != nil && time.Since(*lastClose) > 2*time.Hour {
		flag("收口陈", fmt.Sprintf("收口陈%s", time.Since(*lastClose).Round(time.Minute)))
	}
	if n, err := c.spawnlessOrphans(ctx); err == nil && n > 0 {
		flag("唤起无果", fmt.Sprintf("唤起无果%d", n))
	} else if err != nil {
		flag("唤起?", "唤起?")
	}
	var depNote *string
	if err := c.pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE seat='DEPLOY' AND event='部署' ORDER BY id DESC LIMIT 1`).
		Scan(&depNote); err == nil && depNote != nil &&
		!strings.Contains(*depNote, "rc=0") {
		flag("部署败", "部署败")
	}
	var overdue int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event t
		WHERE t.event='工单' AND t.ts>now()-interval '30 days'
		AND t.ts+COALESCE((substring(t.note from '期限=([0-9]+)h'))::int,24)*interval '1 hour'<now()
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.event='处置' AND x.ts>t.ts
			AND x.note ~ ('工单#'||t.id::text||'([^0-9]|$)'))`).Scan(&overdue); err == nil && overdue > 0 {
		flag("票过期", fmt.Sprintf("票过期%d", overdue))
	}
	flags = append(flags, dead...)
	fpFlags = append(fpFlags, dead...)
	if len(flags) > 0 {
		stuck = strings.Join(flags, ",")
	} else {
		stuck = "0"
	}

	// 纹=结构态指纹（器官态集+间歇件分面+旗名集+办数——
	//   年龄/计数等数值漂不触纹:mvFp 只载 名:态,fpFlags 只载旗名）。
	fpParts := append(mvFp, act...)
	fpParts = append(fpParts, idl...)
	fpParts = append(fpParts, fmt.Sprintf("%d", doN))
	fpParts = append(fpParts, fpFlags...)
	fp := fpHash(fpParts)
	if !gaugeDue(ctx, c, "self_aware", fp, selfAwareEvery) {
		return
	}
	note := fmt.Sprintf("v=%s 自观账: 赚=%s | 动=%s | 息=%s | 办=%s | 卡=%s——自我意识器官帧(工单#13540;段段可复核非自称)",
		fp, earn, move, idle, doing, stuck)
	_, _ = c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','self_aware',$1)`, note)
}

// moneyDist 资金分布三段面（工单O-30472——「钱在哪赚」自观帧加段，
// 只增不替既有判词形）。口径实读定：
//
//	交易部署=fin_position 开仓 notional_usd（state<>'closed'——与本帧
//	  仓数 posN 同集）+ fin_position_view kind='dcd' 策略域产品仓凭据
//	  Σamt×最新 tick 估值 USD（cockpit 敞口同径——USDT/USD 直入、余币
//	  种按 md_ticks.last∨md_mark_index.px 取新、无 tick=0 不虚计；
//	  kind='dcd_owner' 业主独立仓剔出=T-0364 同源；paper 仓走
//	  fin_paper_fill 不入本腿）；
//	赚息腿=fin_holding kind LIKE 'earn%' 持仓 USD 估值（同 tick 径；
//	  stale 行不剔——陈=venue 拉败标非钱消，ledger report earnBal
//	  同径含陈）；
//	闲置=NAV−部署−赚息 残差——腿间时移/跨域资产全归残差，加总恒=NAV，
//	  口径漂移由残差显形不藏；
//	缺段降级：每腿独立读败=该段 ?（部署? 息? 闲?），不炸帧。
func (c coreService) moneyDist(ctx context.Context, nav float64) string {
	var dep, earnLeg float64
	depOK := c.pool.QueryRow(ctx, `SELECT COALESCE(
	  (SELECT sum(notional_usd) FROM fin_position WHERE state<>'closed'),0)
	+ COALESCE((SELECT sum(v.notional*CASE WHEN v.ccy IN ('USDT','USD') THEN 1
		ELSE COALESCE(px.p,0) END)
	  FROM fin_position_view v
	  LEFT JOIN LATERAL (SELECT p FROM (
	      SELECT ts,last AS p FROM md_ticks WHERE inst_id=v.ccy||'-USDT'
	      UNION ALL
	      SELECT ts,px AS p FROM md_mark_index WHERE inst_id=v.ccy||'-USDT'
	    ) u ORDER BY ts DESC LIMIT 1) px ON true
	  WHERE v.kind='dcd'),0)`).Scan(&dep) == nil
	earnOK := c.pool.QueryRow(ctx, `SELECT COALESCE(sum(h.amount*CASE
	  WHEN h.ccy IN ('USDT','USD') THEN 1 ELSE COALESCE(px.p,0) END),0)
	FROM fin_holding h
	LEFT JOIN LATERAL (SELECT p FROM (
	    SELECT ts,last AS p FROM md_ticks WHERE inst_id=h.ccy||'-USDT'
	    UNION ALL
	    SELECT ts,px AS p FROM md_mark_index WHERE inst_id=h.ccy||'-USDT'
	  ) u ORDER BY ts DESC LIMIT 1) px ON true
	WHERE h.kind LIKE 'earn%'`).Scan(&earnLeg) == nil
	switch {
	case depOK && earnOK:
		return fmt.Sprintf("部署%.0f(%.1f%%) 息%.0f 闲%.0f",
			dep, dep/nav*100, earnLeg, nav-dep-earnLeg)
	case depOK:
		return fmt.Sprintf("部署%.0f 息? 闲?", dep)
	case earnOK:
		return fmt.Sprintf("部署? 息%.0f 闲?", earnLeg)
	default:
		return "部署? 息? 闲?"
	}
}

func truncRunes(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// spawnlessOrphans 唤起无果计数——2h 窗内唤起行无 unit 配对收口/果/败。
// 在飞宽限15min：唤起后秒级无配对=会话在跑非孤儿（观测时长32-202s），
// 无宽限=每拍必中在飞唤，旗沦为常开噪声掩真孤儿（87帧实证）。
func (c coreService) spawnlessOrphans(ctx context.Context) (int, error) {
	var n int
	err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='唤起' AND e.ts>now()-interval '2 hours'
			AND e.ts<now()-interval '15 minutes'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.event IN ('唤起果','唤起败','会话收口')
			AND x.note ~ (substring(e.note from 'unit=[^ ]+')||'([^0-9A-Za-z_-]|$)'))`).
		Scan(&n)
	return n, err
}
