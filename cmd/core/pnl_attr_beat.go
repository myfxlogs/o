// pnl_attr_beat.go —— 策略级盈亏归因量表探（业主令 2026-11-12①：
// 盈利报告只有合计口径,live34 谁赚谁赔不可分——策略/族级净盈亏
// 日行入 ops_event 并与账房盈亏桶互证；账房器官 booksBeat 零改动，
// 口径设计归 system 裁）。
//
//	归因径=fin_cashflow.position_ref→fin_position.strategy_ref→
//	fin_strategy(inst_id+template_ref 落签);无 position 链行=未归因
//	桶（不虚构策略归）——Σ(归因+未归因) 对同基盈亏桶实差即咬合差,
//	口径漂移显数不藏。
//	窗=同 booksBeat NAV 锚式（同 equity 帧对）+同六族 kind 闭集=
//	与账房日对账机械咬合,不新开口径。
//	只落量表账页不升件——排序/互证是读面,判词归读者。
package main

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// pnlAttrEvery 归因日帧（量表族同拍——指纹异即重报,同纹 20h 续）。
const pnlAttrEvery = 20 * time.Hour

// pnlAttrShow 账页分列上界——赚列/赔列各取头三,余归「他n策」计。
const pnlAttrShow = 3

// pnlAttrKinds 六族盈亏闭集——与 booksBeat/profitDailyBeat 钦定口径同集
// （咬合前提=同基;改此集须同步账房件）。
var pnlAttrKinds = `'fee','funding_settle','trade_pnl','rollback_cost','spot_buy','spot_sell'`

type pnlAttrRow struct {
	strat, tmpl, inst string
	sum               float64
	n                 int
}

func (c coreService) pnlAttrBeat(ctx context.Context, deadSince map[string]time.Time) {
	// ① 窗锚同 booksBeat（NAV 帧对）——与账房同窗才谈互证。
	var tsNow, tsPrev time.Time
	if err := c.pool.QueryRow(ctx, `SELECT ts FROM md_equity_curve
		WHERE ccy='NAV' ORDER BY ts DESC LIMIT 1`).Scan(&tsNow); err != nil {
		return
	}
	if err := c.pool.QueryRow(ctx, `SELECT ts FROM md_equity_curve
		WHERE ccy='NAV' AND ts<=$1 ORDER BY ts DESC LIMIT 1`,
		tsNow.Add(-pnlAttrEvery)).Scan(&tsPrev); err != nil {
		return // 无前帧=窗不成立（首拍/断供窗让位,同账房律）
	}
	// ② 策略分列（position 链归因;NULL 链=未归因桶如实）。
	rows, err := c.pool.Query(ctx, `SELECT coalesce(s.id::text,'unattr'),
		coalesce(s.template_ref,'-'), coalesce(s.inst_id,'-'),
		sum(c.amount), count(*) FROM fin_cashflow c
		LEFT JOIN fin_position p ON c.position_ref=p.id
		LEFT JOIN fin_strategy s ON p.strategy_ref=s.id
		WHERE c.venue<>'sim' AND c.occurred_at>$1 AND c.occurred_at<=$2
		AND c.kind IN (`+pnlAttrKinds+`)
		GROUP BY 1,2,3 ORDER BY sum(c.amount) DESC`, tsPrev, tsNow)
	if err != nil {
		return
	}
	var rs []pnlAttrRow
	for rows.Next() {
		var r pnlAttrRow
		if rows.Scan(&r.strat, &r.tmpl, &r.inst, &r.sum, &r.n) == nil {
			rs = append(rs, r)
		}
	}
	rows.Close()
	if len(rs) == 0 {
		return // 窗内零现金行=无话可述（窗存而账空,不报空页）
	}
	// ③ 咬合件——同基盈亏桶（两查询实差=口径漂移显数）。
	var earned float64
	_ = c.pool.QueryRow(ctx, `SELECT coalesce(sum(amount),0) FROM fin_cashflow
		WHERE venue<>'sim' AND occurred_at>$1 AND occurred_at<=$2
		AND kind IN (`+pnlAttrKinds+`)`, tsPrev, tsNow).Scan(&earned)
	attrSum, unattr := 0.0, 0.0
	nStrat := 0
	for _, r := range rs {
		attrSum += r.sum
		if r.strat == "unattr" {
			unattr += r.sum
		} else {
			nStrat++
		}
	}
	fp := fmt.Sprintf("%.0f|%.0f|%d", attrSum, unattr, nStrat)
	if !gaugeDue(ctx, c, "pnl_attr_daily", fp, pnlAttrEvery) {
		return
	}
	note := pnlAttrNote(fp, rs, attrSum, unattr, nStrat, earned, tsPrev, tsNow)
	_, _ = c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','pnl_attr_daily',$1)`,
		note)
}

// pnlAttrNote 归因账页文（纯函数供 shape 测）——分列赚头/赔头/未归因+
// 咬合差,超列余收「他n策」。
func pnlAttrNote(fp string, rs []pnlAttrRow, attrSum, unattr float64, nStrat int,
	earned float64, tsPrev, tsNow time.Time) string {
	gain := make([]string, 0, pnlAttrShow)
	loss := make([]string, 0, pnlAttrShow)
	otherN, otherSum := 0, 0.0
	rest := rs
	for i := 0; i < len(rest) && len(gain) < pnlAttrShow; i++ {
		if rest[i].strat != "unattr" && rest[i].sum > 0 {
			gain = append(gain, pnlAttrCell(rest[i]))
			rest = append(rest[:i], rest[i+1:]...)
			i--
		}
	}
	for i := len(rest) - 1; i >= 0 && len(loss) < pnlAttrShow; i-- {
		if rest[i].strat != "unattr" && rest[i].sum < 0 {
			loss = append(loss, pnlAttrCell(rest[i]))
			rest = append(rest[:i], rest[i+1:]...)
		}
	}
	for _, r := range rest {
		if r.strat != "unattr" {
			otherN++
			otherSum += r.sum
		}
	}
	bite := attrSum - earned
	return fmt.Sprintf(
		"v=%s 策略盈亏归因: 窗%.0fh 策动%d Σ%+.2f 对账房盈亏%+.2f 咬合差%+.2f——赚[%s] 赔[%s] 他%d策%+.2f 未归因%+.2f",
		fp, tsNow.Sub(tsPrev).Hours(), nStrat, attrSum, earned, bite,
		strings.Join(gain, "|"), strings.Join(loss, "|"), otherN, otherSum, unattr)
}

func pnlAttrCell(r pnlAttrRow) string {
	inst := r.inst
	if len(inst) > 14 {
		inst = inst[:14]
	}
	return fmt.Sprintf("#%s %s(%s)%+.2f", r.strat, inst, r.tmpl, r.sum)
}
