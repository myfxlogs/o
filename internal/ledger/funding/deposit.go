// deposit.go —— 入金检测径（T-0149：venue 入金无自动入账=账实差告警源。
// deposit-history 无推送面 venue=REST 低周期补漏正当面——非主径轮询）。
// bill_id=dep-<depId> 幂等（重启/重放不双记）；state=2 完成态才入账。
package funding

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DepositSrc 充币史读件（装配层=okxexec Get /asset/deposit-history——
// 凭据缺席=读件 nil 让位静默非零报）。
type DepositSrc interface {
	DepositHistory(ctx context.Context) ([]DepositRow, error)
}

// DepositRow 充币行（OKX 方言：depId 幂等键/state=2 完成态/ts ms 串）。
type DepositRow struct {
	DepID string `json:"depId"`
	Ccy   string `json:"ccy"`
	Amt   string `json:"amt"`
	State string `json:"state"`
	Ts    string `json:"ts"`
}

// DepositScanner 入金巡检件（fin_venue_bill 源行→fin_cashflow deposit 入账——
// source_ref 引用链禁推断记账）。
type DepositScanner struct {
	pool *pgxpool.Pool
	src  DepositSrc
}

func NewDepositScanner(pool *pgxpool.Pool, src DepositSrc) *DepositScanner {
	return &DepositScanner{pool: pool, src: src}
}

// Scan 巡检一拍：state=2 完成行→bill 幂等落→cashflow deposit。
// 已记过/读败=零行零告警（幂等切面；读败下拍重试）。
func (s *DepositScanner) Scan(ctx context.Context) (int, error) {
	if s.src == nil {
		return 0, nil // 读件未挂（凭据缺席）——让位非误报
	}
	rows, err := s.src.DepositHistory(ctx)
	if err != nil {
		return 0, err
	}
	var n int
	for _, r := range rows {
		if r.State != "2" {
			continue // 非完成态（确认中/失败）——不入账留档
		}
		tsMs, _ := parseMs(r.Ts)
		// T-0149 返修②：bill+cashflow 同事务原子（注入败→bill 行同回滚
		// 不留孤儿源行——幂等切面不失效语义守恒）。
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return n, err
		}
		var srcID int64
		err = tx.QueryRow(ctx, `INSERT INTO fin_venue_bill
			(bill_id,ts,inst_id,inst_type,ccy,type,sub_type,bal_chg,fee,pnl,sz,px,raw)
			VALUES($1,$2,'','ASSET',$3,'deposit','',$4,0,0,0,0,$5)
			ON CONFLICT (bill_id) DO NOTHING RETURNING id`,
			"dep-"+r.DepID, time.UnixMilli(tsMs), r.Ccy, r.Amt,
			json.RawMessage(mustJSON(r))).Scan(&srcID)
		if errors.Is(err, pgx.ErrNoRows) {
			_ = tx.Rollback(ctx)
			continue // 已记过——幂等切面不双记
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return n, err
		}
		amt, _ := parseFloat(r.Amt)
		if _, err := tx.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
			VALUES('deposit','okx',$1,$2,$3,'fin_venue_bill',$4)`,
			r.Ccy, amt, srcID, time.UnixMilli(tsMs)); err != nil {
			_ = tx.Rollback(ctx)
			return n, err
		}
		if err := tx.Commit(ctx); err != nil {
			return n, err
		}
		log.Printf("[ledger] 入金入账 depId=%s %.4f %s（fin_venue_bill id=%d）", r.DepID, amt, r.Ccy, srcID)
		n++
	}
	return n, nil
}

func parseMs(s string) (int64, error) {
	var v int64
	err := json.Unmarshal([]byte(s), &v)
	if err != nil {
		var f float64
		err2 := json.Unmarshal([]byte(s), &f)
		return int64(f), err2
	}
	return v, nil
}

func parseFloat(s string) (float64, error) {
	var f float64
	err := json.Unmarshal([]byte(s), &f)
	return f, err
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
