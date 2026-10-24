// Package report —— 经营报表（信使岗）：日/周/月三粒度聚合单源账本行
// （fin_cashflow 净额/fin_strategy 开平/fin_opp_eval 计数/fin_holding 停泊额/
// fin_ledger_snapshot 净值）→fin_report 幂等行+GET /report 取径。
// 铁律：禁账本外数字；期内零行照样出行+「本期无开平」注记。
package report

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Bounds 期界。
type Bounds struct{ Start, End time.Time }

// Gen 一期报表聚合（幂等 UPSERT）。
func Gen(ctx context.Context, pool *pgxpool.Pool, period string, b Bounds) error {
	m := map[string]any{}
	q := func(sql string, dst *float64, args ...any) {
		var v float64
		if err := pool.QueryRow(ctx, sql, args...).Scan(&v); err == nil {
			*dst = v
		}
	}
	q1 := func(sql string, dst *int, args ...any) {
		var v int
		if err := pool.QueryRow(ctx, sql, args...).Scan(&v); err == nil {
			*dst = v
		}
	}
	var net, funding, earnBal, cumNav float64
	var opened, closed, proposed, rejected, active, alerts, capital int
	q(`SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE occurred_at>=$1 AND occurred_at<$2`, &net, b.Start, b.End)
	q(`SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='funding_settle' AND occurred_at>=$1 AND occurred_at<$2`, &funding, b.Start, b.End)
	q(`SELECT COALESCE(sum(amount),0) FROM fin_cashflow`, &cumNav)
	q(`SELECT COALESCE(sum(amount),0) FROM fin_holding WHERE kind LIKE 'earn%' AND stale=false`, &earnBal)
	q1(`SELECT count(*) FROM fin_strategy WHERE entered_at>=$1 AND entered_at<$2`, &opened, b.Start, b.End)
	q1(`SELECT count(*) FROM fin_position WHERE closed_at IS NOT NULL AND closed_at>=$1 AND closed_at<$2`, &closed, b.Start, b.End)
	q1(`SELECT count(*) FROM fin_opp_eval WHERE verdict='proposed' AND ts>=$1 AND ts<$2`, &proposed, b.Start, b.End)
	q1(`SELECT count(*) FROM fin_opp_eval WHERE verdict='rejected' AND ts>=$1 AND ts<$2`, &rejected, b.Start, b.End)
	q1(`SELECT count(*) FROM fin_risk_decision WHERE decision='deny' AND ts>=$1 AND ts<$2`, &alerts, b.Start, b.End)
	q1(`SELECT count(*) FROM fin_strategy WHERE state<>'retired'`, &active)
	q1(`SELECT count(*) FROM fin_cashflow WHERE kind='deposit'`, &capital)
	note := ""
	if net == 0 && opened == 0 && closed == 0 {
		note = "本期无开平/零现金流（缺席≠静默——如实出行）"
	}
	if period == "monthly" {
		var base float64
		q(`SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='deposit'`, &base)
		ann := 0.0
		if base > 0 {
			ann = net / base * 12
		}
		m["annualized"] = ann
		m["capital_base"] = base
		var retired int
		q1(`SELECT count(*) FROM fin_strategy_transition WHERE to_state='retired' AND ts>=$1 AND ts<$2`, &retired, b.Start, b.End)
		m["retired"] = retired
		note = fmt.Sprintf("月净=%.4f 年化=%.4f（基数 %.0f）——赚亏/因/步三行交卷", net, ann, base)
	}
	m["net"] = net
	m["cum_nav"] = cumNav
	m["funding_income"] = funding
	m["earn_balance"] = earnBal
	m["opened"], m["closed"] = opened, closed
	m["proposed"], m["rejected"], m["active"], m["alerts"], m["capital_flows"] = proposed, rejected, active, alerts, capital
	if _, err := pool.Exec(ctx, `INSERT INTO fin_report(period,period_start,period_end,metrics,note)
		VALUES($1,$2,$3,$4,$5) ON CONFLICT(period,period_start) DO UPDATE
		SET period_end=EXCLUDED.period_end,metrics=EXCLUDED.metrics,note=EXCLUDED.note,generated_at=now()`,
		period, b.Start, b.End, m, note); err != nil {
		return fmt.Errorf("report upsert: %w", err)
	}
	log.Printf("[report] %s %s net=%.4f opened=%d closed=%d", period, b.Start.Format("2006-01-02"), net, opened, closed)
	return nil
}
