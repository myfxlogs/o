// withdraw.go —— 出金常设翻译径（T-0274）。/account/bills type=1 划转行行面
// 无内外向分辨域（raw={} 无 from/to——真户探针 2026-11-01 实证 /asset/bills
// 行亦然），划转行禁凭 sub_type/符号猜内外向。择源=/asset/withdrawal-history：
// 外向出金唯一在册源——链上提币与 OKX 内部户转出（本案 wdId 440849407 形态）
// 皆在册且 state=2 完成态终证，内转双跳（trading↔funding↔earn）天然缺席
// =行面即外向定性，零分辨赌猜。bill_id=wd-<wdId> 幂等（deposit.go T-0149
// 范式镜像——幂等/基线/原子三锚同构）。
package funding

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/rejects"
)

// WithdrawSrc 提币史读件（装配层=okxexec Get /asset/withdrawal-history——
// 凭据缺席=读件 nil 让位静默非零报）。
type WithdrawSrc interface {
	WithdrawalHistory(ctx context.Context) ([]WithdrawalRow, error)
}

// WithdrawalRow 提币行（OKX 方言：wdId 幂等键/state=2 完成态/ts ms 串——
// deposit-history 同构；amt=出金本金面。fee 另列不入本径：费面归既有
// bills 记账 fee kind 支，混读=双计）。
type WithdrawalRow struct {
	WdID  string `json:"wdId"`
	Ccy   string `json:"ccy"`
	Amt   string `json:"amt"`
	Fee   string `json:"fee"`
	State string `json:"state"`
	Ts    string `json:"ts"`
}

// WithdrawalScanner 出金巡检件（fin_venue_bill 源行→fin_cashflow withdraw
// 入账——amount 负号=出向；source_ref 引用链禁推断记账）。
type WithdrawalScanner struct {
	pool *pgxpool.Pool
	src  WithdrawSrc
	rej  *rejects.Door
}

func NewWithdrawalScanner(pool *pgxpool.Pool, src WithdrawSrc) *WithdrawalScanner {
	return &WithdrawalScanner{pool: pool, src: src,
		rej: rejects.New("withdraw", "write_err", "non_usdt_withdrawal", "pre_baseline")}
}

// Scan 巡检一拍：state=2 完成行→bill 幂等落→cashflow withdraw（-amt）。
// 已记过/读败=零行零告警（幂等切面；读败下拍重试不炸事件主径）。
func (s *WithdrawalScanner) Scan(ctx context.Context) (int, error) {
	if s.src == nil {
		return 0, nil // 读件未挂（凭据缺席）——让位非误报
	}
	rows, err := s.src.WithdrawalHistory(ctx)
	if err != nil {
		return 0, err
	}
	// 账基线截止谓词（T-0149 范式镜像）——账基线前 venue 史出金=非基金
	// 出向（2026-11-01 实证：wdId 433855125 等 pre-baseline 旧提币在册）。
	var baseline time.Time
	_ = s.pool.QueryRow(ctx, `SELECT COALESCE(min(occurred_at),now()) FROM fin_cashflow`).Scan(&baseline)
	var n int
	for _, r := range rows {
		if r.State != "2" {
			continue // 非完成态（提现中/失败）——不入账留档
		}
		if r.Ccy != "USDT" {
			s.rej.Inc("non_usdt_withdrawal") // 杂币闭集外（USDT 本位 T-0149 对偶）
			continue
		}
		tsMs, _ := parseMs(r.Ts)
		occurred := time.UnixMilli(tsMs).UTC()
		if !occurred.After(baseline) {
			s.rej.Inc("pre_baseline") // 账基线前史=非基金出向不入账
			continue
		}
		amt, _ := parseFloat(r.Amt)
		// bill+cashflow 同事务原子（T-0149 返修②范式）：注入败→bill 行同回滚
		// 不留孤儿源行——幂等切面不失效语义守恒。
		tx, err := s.pool.Begin(ctx)
		if err != nil { // 逐行隔离：单行败=log+rej+continue 不 abort 整扫
			s.rej.Inc("write_err")
			log.Printf("[ledger] 出金行 begin 败（wdId=%s 跳过续扫）: %v", r.WdID, err)
			continue
		}
		var srcID int64
		err = tx.QueryRow(ctx, `INSERT INTO fin_venue_bill
			(bill_id,ts,inst_id,inst_type,ccy,type,sub_type,bal_chg,fee,pnl,sz,px,raw)
			VALUES($1,$2,'','ASSET',$3,'withdraw','',$4,0,0,0,0,$5)
			ON CONFLICT (bill_id) DO NOTHING RETURNING id`,
			"wd-"+r.WdID, occurred, r.Ccy, -amt,
			json.RawMessage(mustJSON(r))).Scan(&srcID)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			continue // 已记过——幂等切面不双记
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			s.rej.Inc("write_err")
			log.Printf("[ledger] 出金行 bill 写败（wdId=%s 跳过续扫）: %v", r.WdID, err)
			continue
		}
		if _, err2 := tx.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
			VALUES('withdraw','okx',$1,$2,$3,'fin_venue_bill',$4)`,
			r.Ccy, -amt, srcID, occurred); err2 != nil {
			_ = tx.Rollback(ctx)
			s.rej.Inc("write_err")
			log.Printf("[ledger] 出金行 cashflow 写败（wdId=%s 同回滚跳过续扫）: %v", r.WdID, err2)
			continue
		}
		if err := tx.Commit(ctx); err != nil {
			s.rej.Inc("write_err")
			log.Printf("[ledger] 出金行 commit 败（wdId=%s 跳过续扫）: %v", r.WdID, err)
			continue
		}
		log.Printf("[ledger] 出金入账 wdId=%s -%.4f %s（fin_venue_bill id=%d）", r.WdID, amt, r.Ccy, srcID)
		n++
	}
	return n, nil
}
