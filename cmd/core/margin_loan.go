package main

// margin_loan.go —— O-40707 在借仓拍三件（margin.go 采集细胞姊妹件）：
// 息腿 accrual（borrow_interest 负腿）+退役/平仓触发 close 序+
// marginRatio 三档健康闸（warn 升件/kill flatten→repay 钉死序）。

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/ledger"
	"o/internal/platform/config"
	"o/internal/trading/okxexec"
)

// marginLoanTick 在借仓拍：①息腿 accrual ②退役/平仓触发 close ③健康闸。
// 次序=息先行（钱已动先记账）→触发序→闸（闸后行不可反——健康判词用
// 本拍前全部事实）。
func marginLoanTick(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client,
	venue marginVenue) error {
	loans, err := marginOpenLoans(ctx, pool)
	if err != nil {
		return err
	}
	if len(loans) == 0 {
		return nil // 零在借=健康闸/息腿全让位（闸先于执行活——无仓无闸可判）
	}
	if err := marginAccrue(ctx, pool, loans); err != nil {
		log.Printf("[margin] 息腿 accrual 败（下拍续）: %v", err)
	}
	// 退役/仓平触发——策略 retired∪position 已闭=仓位事实终结，负债须还。
	for _, l := range loans {
		var stratID, posID any
		if l.StrategyRef != nil {
			stratID = *l.StrategyRef
		}
		if l.PositionRef != nil {
			posID = *l.PositionRef
		}
		var dead bool
		_ = pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM fin_strategy s WHERE s.id=$1 AND s.state='retired')
			OR EXISTS(SELECT 1 FROM fin_position p WHERE p.id=$2 AND p.closed_at IS NOT NULL)`,
			stratID, posID).Scan(&dead)
		if dead {
			if err := marginCloseLoan(ctx, pool, venue, l, "strategy_dead"); err != nil {
				log.Printf("[margin] 退役触发平仓序 %s/%s 败: %v", l.Ccy, l.InstID, err)
			}
		}
	}
	return marginHealthGate(ctx, pool, cfg, venue, loans)
}

// marginOpenLoans 在借仓全枚举（open+closing——closing=平仓序在途仍计息
// 负债，息腿不停）。
func marginOpenLoans(ctx context.Context, pool *pgxpool.Pool) ([]marginLoan, error) {
	rows, err := pool.Query(ctx, `SELECT id,venue,ccy,inst_id,amt,apr,status,
		strategy_ref,position_ref,execution_id,last_accrual_at
		FROM fin_margin_loan WHERE status IN ('open','closing') ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []marginLoan
	for rows.Next() {
		var l marginLoan
		if err := rows.Scan(&l.ID, &l.Venue, &l.Ccy, &l.InstID, &l.Amt, &l.APRSnap,
			&l.Status, &l.StrategyRef, &l.PositionRef, &l.ExecutionID, &l.LastAccrualAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// marginAccrue 息腿簿记：elapsed×amt×apr/8760（apr 取 md_margin_borrow
// 最新行——缺行回退借入快照；elapsed 计时制=重启不重计不双计）。
// ledger.Post 败=该币跳过不推进 last_accrual_at（下拍补记——禁先标后记）。
func marginAccrue(ctx context.Context, pool *pgxpool.Pool, loans []marginLoan) error {
	for _, l := range loans {
		elapsedH := time.Since(l.LastAccrualAt).Hours()
		if elapsedH <= 0.001 { // <3.6s——拍距内零头不产尘行
			continue
		}
		var apr float64
		if err := pool.QueryRow(ctx, `SELECT borrow_apr FROM md_margin_borrow
			WHERE ccy=$1 ORDER BY ts DESC LIMIT 1`, l.Ccy).Scan(&apr); err != nil || apr <= 0 {
			apr = l.APRSnap
		}
		interest := l.Amt * apr / 8760 * elapsedH
		if interest <= 0 {
			continue
		}
		if _, err := ledger.Post(ctx, pool, ledger.Entry{
			Kind: "borrow_interest", Venue: l.Venue, Ccy: l.Ccy,
			Amount: -interest, PositionRef: l.PositionRef,
			SourceRef: l.ID, SourceTable: "fin_margin_loan",
		}); err != nil {
			log.Printf("[margin] 息腿入账 %s #%d 败: %v", l.Ccy, l.ID, err)
			continue
		}
		if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan
			SET last_accrual_at=now(), accrued_interest=accrued_interest+$2 WHERE id=$1`,
			l.ID, interest); err != nil {
			log.Printf("[margin] 息腿标记 %d 败（下拍或重计——对账面可见）: %v", l.ID, err)
		}
	}
	return nil
}

// marginHealthGate marginRatio 三档：负债USD（Σamt×现货价）/抵押物
// adjEq——warn=升件去重 6h，kill=逐仓平仓序+升件。adjEq 读败/≤0 且有负债
// =ratio 不可证→按 kill 处置（fail-closed——裸奔负债不等读面恢复）。
func marginHealthGate(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client,
	venue marginVenue, loans []marginLoan) error {
	warnR, err1 := cfg.GetFloat("margin.ratio_warn")
	killR, err2 := cfg.GetFloat("margin.ratio_kill")
	if err1 != nil || err2 != nil || warnR <= 0 || killR <= warnR {
		return fmt.Errorf("margin.ratio_* 判据键缺/倒置 fail-closed: %v/%v", err1, err2)
	}
	var loanUSD float64
	for _, l := range loans {
		if px := marginSpotPx(ctx, pool, l.InstID); px > 0 {
			loanUSD += l.Amt * px
		} else {
			loanUSD += l.Amt // 无价按名义保守下界——币价≥0 低估不致放行 kill 档
		}
	}
	eq, err := venue.AccountAdjEquity(ctx)
	if err != nil {
		if loanUSD > 0 {
			log.Printf("ALERT [margin] adjEq 读败且有在借负债 %.2f USD——按 kill 平仓序处置", loanUSD)
			return marginKillAll(ctx, pool, venue, loans, "adjEquity_read_failed")
		}
		return nil
	}
	ratio := math.Inf(1)
	if eq > 0 {
		ratio = loanUSD / eq
	}
	if ratio >= killR {
		return marginKillAll(ctx, pool, venue, loans,
			fmt.Sprintf("ratio=%.3f≥kill %.2f（loan=%.2f adjEq=%.2f）", ratio, killR, loanUSD, eq))
	}
	if ratio >= warnR {
		return marginWarnOnce(ctx, pool, ratio, loanUSD, eq)
	}
	return nil
}

// marginWarnOnce 预警升件（6h 去重——同级持续不重复刷件；seat=探测席
// 升件契约同 hunger 探针族，esc_class/esc_scope=fin_0264 ops_esc_valence
// 在册两值）。
func marginWarnOnce(ctx context.Context, pool *pgxpool.Pool, ratio, loanUSD, eq float64) error {
	var dup bool
	_ = pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class='margin_ratio_warn'
		AND ts>now()-interval '6 hours')`).Scan(&dup)
	if dup {
		return nil
	}
	_, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','O-40707',$1,'margin_ratio_warn','okx')`,
		fmt.Sprintf("marginRatio=%.3f 预警档（负债 %.2f USD/抵押 %.2f）——逼近 kill 平仓序线", ratio, loanUSD, eq))
	return err
}

// marginKillAll kill 档：升件留痕+逐仓 flatten→repay 序。
func marginKillAll(ctx context.Context, pool *pgxpool.Pool, venue marginVenue,
	loans []marginLoan, reason string) error {
	_, _ = pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','O-40707',$1,'margin_ratio_kill','okx')`,
		fmt.Sprintf("marginRatio kill 平仓序触发：%s——在借 %d 仓", reason, len(loans)))
	for _, l := range loans {
		if l.Status == "open" {
			if err := marginCloseLoan(ctx, pool, venue, l, "ratio_kill:"+reason); err != nil {
				log.Printf("[margin] kill 平仓序 %s 败: %v", l.InstID, err)
			}
		}
	}
	return nil
}

// marginCloseLoan 平仓序（spec 钉死先 flatten 后 repay）：
// ①市价买回 amt 基币（tgtCcy=base_ccy——sz=币量）②定态 filled 才进
// ③SpotManualBorrowRepay repay——flatten 败/歧态=flatten_failed 不 repay
//
//	（币没买回就还=裸空事故）；repay 败=留 closing 态下拍续（幂等：
//	venue 侧 repay 重复=重复还——先查历史对账，见 marginRepayOnce）。
func marginCloseLoan(ctx context.Context, pool *pgxpool.Pool, venue marginVenue,
	l marginLoan, reason string) error {
	if _, err := pool.Exec(ctx, `UPDATE fin_margin_loan SET status='closing',
		note=$2 WHERE id=$1 AND status='open'`, l.ID, reason); err != nil {
		return err
	}
	l.Status = "closing"
	// ①flatten——买回借币量（spot market buy tgtCcy=base_ccy）。
	clOrdID := fmt.Sprintf("mgc%d", l.ID)
	_, err := venue.PlaceOrder(ctx, okxexec.OrderParams{
		InstID: l.InstID, TdMode: "cash", Side: "buy", OrdType: "market",
		Sz: fmt.Sprintf("%g", l.Amt), ClOrdID: clOrdID, TgtCcy: "base_ccy"})
	if err != nil {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='flatten_failed',
			note=$2 WHERE id=$1`, l.ID, "flatten_place:"+err.Error())
		return fmt.Errorf("flatten 下单败（不 repay——裸空防）: %w", err)
	}
	// 歧态势律：下单传输成败皆须定态——filled 才进 repay；live/partial/
	// unknown=flatten_failed 留账人工/下拍核（市价单应瞬时 filled，非
	// filled=异常面显形不猜）。
	st, _, fillSz, _, _, rerr := venue.ResolveOrderDetail(ctx, l.InstID, clOrdID)
	if rerr != nil || st != okxexec.StateFilled || fillSz <= 0 {
		_, _ = pool.Exec(ctx, `UPDATE fin_margin_loan SET status='flatten_failed',
			note=$2 WHERE id=$1`, l.ID,
			fmt.Sprintf("flatten_state:%v fill=%.8g err=%v", st, fillSz, rerr))
		return fmt.Errorf("flatten 未 filled（state=%s fill=%.8g err=%v）——不 repay", st, fillSz, rerr)
	}
	// ②repay——先历史对账防重复还（venue 无幂等键：同 ccy 近 10min 已有
	// repay 行且本仓未标 repaid=疑似在途/已还——跳过本次 repay 直接核）。
	if err := marginRepayOnce(ctx, pool, venue, l); err != nil {
		return err // 留 closing——下拍续（repay 对账幂等不双还）
	}
	// ③终态：仓+执行+头寸同事务收口。
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE fin_margin_loan SET status='repaid',
		repaid_at=now(), note=$2 WHERE id=$1`, l.ID, "closed:"+reason); err != nil {
		return err
	}
	if l.PositionRef != nil {
		if _, err := tx.Exec(ctx, `UPDATE fin_position SET state='closed',
			closed_at=now() WHERE id=$1 AND closed_at IS NULL`, *l.PositionRef); err != nil {
			return err
		}
	}
	if l.ExecutionID != nil {
		if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state='closed'
			WHERE id=$1 AND state<>'closed'`, *l.ExecutionID); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("[margin] 平仓序成 %s #%d（%s）——flatten filled=%.8g→repaid", l.Ccy, l.ID, reason, fillSz)
	return nil
}

// marginRepayOnce repay+历史对账（venue 无幂等键——先查 spot-borrow-
// repay-history：近 10min 已有本币 repay 行=疑似在途/已还→跳过发单
// 直接由调用面收口；无行才发 repay。发单败=err 上抛留 closing 下拍续）。
func marginRepayOnce(ctx context.Context, pool *pgxpool.Pool, venue marginVenue,
	l marginLoan) error {
	hist, err := venue.SpotBorrowRepayHistory(ctx, l.Ccy, "", 10)
	if err != nil {
		return fmt.Errorf("repay 前历史对账读败（禁盲发）: %w", err)
	}
	for _, h := range hist {
		if h.Side != "repay" {
			continue
		}
		if ts, perr := strconv.ParseInt(h.TS, 10, 64); perr == nil &&
			time.Since(time.UnixMilli(ts)) < 10*time.Minute {
			return nil // 近 10min 已有 repay 在案——不重发（歧态势律）
		}
	}
	if _, err := venue.SpotManualBorrowRepay(ctx, l.Ccy, l.Amt, "repay"); err != nil {
		return fmt.Errorf("repay 发单败（留 closing 下拍续）: %w", err)
	}
	return nil
}
