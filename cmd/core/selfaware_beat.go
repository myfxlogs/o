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
	{"懂你", `SELECT max(ts) FROM ops_event WHERE ticket='owner_intent'`},
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
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE venue<>'sim' AND ts>now()-interval '30 days'
		AND kind IN ('fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell')`).Scan(&net)
	_ = c.pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE state='live'),
		count(*) FILTER (WHERE state='probe'),
		(SELECT count(*) FROM fin_position WHERE closed_at IS NULL)
		FROM fin_strategy`).Scan(&liveN, &probeN, &posN)
	if nav != nil && navTs != nil {
		earn = fmt.Sprintf("NAV%.0f(%s) 30d净%+.2f live%d/probe%d/仓%d",
			*nav, time.Since(*navTs).Round(time.Minute), net, liveN, probeN, posN)
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

	// ④ 办：未配对 在办/认领 行（2h TTL 同认领律——陈行转卡旗不计办）。
	var doN int
	var doFirst *string
	if err := c.pool.QueryRow(ctx, `SELECT count(*), min(note) FROM ops_event
		WHERE event IN ('认领','在办') AND ts>now()-interval '2 hours'`).
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
	var spawnless int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event e
		WHERE e.event='唤起' AND e.ts>now()-interval '2 hours'
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.event IN ('唤起果','唤起败','会话收口')
			AND x.note LIKE '%'||substring(e.note from 'unit=[^ ]+')||'%')`).
		Scan(&spawnless); err == nil && spawnless > 0 {
		flag("唤起无果", fmt.Sprintf("唤起无果%d", spawnless))
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

func truncRunes(s string, n int) string {
	r := []rune(strings.ReplaceAll(s, "\n", " "))
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
