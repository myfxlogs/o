// borrowquote.go —— margin-short 借币证据读件（O-39460；md_margin_borrow
// 最新行→BorrowQuote + 三道闸：新鲜度/息尖峰/额度下限——全 fail-closed
// 如实拒不虚构可借性）。
package funding

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// borrowQuote 取基币保证金借币报价。返回 (quote, rejectReason)——
// reason 非空=不可评估拒（证据缺/陈/超闸），quote 仅在可评估时有效。
func (r *Runner) borrowQuote(ctx context.Context, baseCcy, instID string) (BorrowQuote, string) {
	staleH, err := r.cfg.GetFloat("funding.margin_borrow_stale_hours")
	if err != nil || staleH <= 0 {
		return BorrowQuote{}, "borrow_evidence_unconfigured"
	}
	maxAPR, err := r.cfg.GetFloat("funding.margin_borrow_max_apr")
	if err != nil || maxAPR <= 0 {
		return BorrowQuote{}, "borrow_gate_unconfigured"
	}
	minLoanUSD, err := r.cfg.GetFloat("funding.margin_min_loan_usd")
	if err != nil || minLoanUSD <= 0 {
		return BorrowQuote{}, "borrow_gate_unconfigured"
	}
	var apr, maxLoan, px float64
	var ts time.Time
	err = r.pool.QueryRow(ctx, `SELECT borrow_apr, max_loan, ts FROM md_margin_borrow
		WHERE ccy=$1 ORDER BY ts DESC LIMIT 1`, baseCcy).Scan(&apr, &maxLoan, &ts)
	if err == pgx.ErrNoRows {
		return BorrowQuote{}, "borrow_evidence_missing"
	}
	if err != nil {
		return BorrowQuote{}, "borrow_evidence_read_failed"
	}
	if time.Since(ts) > time.Duration(staleH*float64(time.Hour)) {
		return BorrowQuote{}, "borrow_evidence_stale"
	}
	if apr > maxAPR {
		return BorrowQuote{}, fmt.Sprintf("borrow_rate_spike:apr=%.4f>闸%.4f", apr, maxAPR)
	}
	// 额度 USD 名义=借币量×现货价——价格面复用 md_mark_index/最新 tick
	// 不可得时保守按 0=额度不可证（fail-closed 不放行）。
	loanUSD := 0.0
	if px, err = r.spotPx(ctx, baseCcy); err == nil && px > 0 {
		loanUSD = maxLoan * px
	}
	if loanUSD < minLoanUSD {
		return BorrowQuote{}, fmt.Sprintf("borrow_quota_insufficient:loan_usd=%.2f<闸%.0f", loanUSD, minLoanUSD)
	}
	return BorrowQuote{APR: apr, MaxLoan: maxLoan, LoanUSD: loanUSD}, ""
}

// spotPx 基币-USDT 现货最新价（md_mark_index.px∪md_ticks.last 两级——
// diff_px 同面真列名+10min 鲜度窗；O-40707 坑修：md_ticks 无 px 列，
// 旧 UNION 对 ticks 段结构性读败→额度恒 0=闸恒拒——改 last 真列名；
// 读败/无鲜行=调用面按额度不可证 fail-closed）。
func (r *Runner) spotPx(ctx context.Context, baseCcy string) (float64, error) {
	instID := baseCcy + "-USDT"
	var px float64
	err := r.pool.QueryRow(ctx, `SELECT v FROM (
		SELECT px v, ts FROM md_mark_index WHERE inst_id=$1
		UNION ALL SELECT last, ts FROM md_ticks WHERE inst_id=$1 AND venue<>'sim'
	) t WHERE ts > now()-interval '10 minutes' ORDER BY ts DESC LIMIT 1`, instID).Scan(&px)
	return px, err
}

// marginShortEval rate<0 分支评估入口（scan 调用——证据闸不过=如实拒卡
// 不耗提案位；EvaluateMarginShort 净额含借币息）。
func (r *Runner) marginShortEval(ctx context.Context, instID string, rate float64, p Params) Eval {
	base := strings.TrimSuffix(instID, "-USDT-SWAP")
	q, why := r.borrowQuote(ctx, base, instID)
	if why != "" {
		return Eval{InstID: instID, Direction: "long_perp_short_spot",
			Legs:     map[string]any{"perp": "long", "spot": "short_margin"},
			Verdict:  "rejected",
			Reason:   why,
			Evidence: map[string]any{"funding_rate": rate, "borrow_gate": why}}
	}
	return EvaluateMarginShort(instID, rate, p, q)
}
