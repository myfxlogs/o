package pgstore

// trx_book.go —— TRX 自营账（业主令 2026-11-09：TRX=业主自营盘,
// 收益/出入金只记账不入公司盈亏）。产线=collector 写径 tx_id
// 幂等锚同切面,USDT 等值+source_ref 溯源保真。
import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// UpsertTRXTxBooked 幂等写交易流+自营账同 tx 原子落（业主令 2026-11-09：
// TRX=自营盘只记账不入公司盈亏）。原子因：分库写则 cashflow 败而 txs 已
// 提交→下轮 conflict→srcID=0→永久漏记；同 tx 则败=两写齐滚,重采重记。
// 幂等切面：tx_id 冲突零行=已记过,cashflow 不重记（重放安全）。
// 自营账口径：ccy 闭集 USDT——amount=TRX 原生额×鲜价(<1d mark),
// 原生额由 source_ref→fin_trx_txs.id 溯源保真；无鲜价=fail-closed
// 报错回滚（下拍重采重试,宁缺勿假价入账）。
func (s *Store) UpsertTRXTxBooked(ctx context.Context, t TRXTx) error {
	if t.TxID == "" {
		return errors.New("trx txs: empty tx_id — field mapping broken (R3 防御：空键坍缩成一行)")
	}
	ck := TRXPropKind(t.Kind)
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO fin_trx_txs (tx_id, ts, kind, amount_sun, counterparty, cheat_status, fetched_at)
			VALUES ($1,$2,$3,$4,$5,$6,COALESCE($7,now()))
			ON CONFLICT (tx_id) DO NOTHING
			RETURNING id
		`, t.TxID, t.Ts, t.Kind, t.AmountSun, t.Counterparty, t.CheatStatus, t.FetchedAt).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("upsert trx tx: %w", err)
		}
		if ck == "" {
			return nil // 链上动作类非价值移动——不入账
		}
		var px float64
		if err := tx.QueryRow(ctx, `SELECT px FROM md_mark_index
			WHERE inst_id='TRX-USDT-SWAP' AND ts>now()-interval '1 day'
			ORDER BY ts DESC LIMIT 1`).Scan(&px); err != nil {
			return fmt.Errorf("trx prop px 鲜价缺席 fail-closed: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO fin_cashflow
			(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
			VALUES($1,'tron','USDT',$2,$3,'fin_trx_txs',$4)`,
			ck, float64(t.AmountSun)/1e6*px, id, t.Ts)
		if err != nil {
			return fmt.Errorf("trx prop cashflow: %w", err)
		}
		return nil
	})
}

// TRXPropKind 自营账映射：fin_trx_txs.kind → fin_cashflow.kind。
// 收益/出入金→owner_in/out（transfer 类天然不入 P&L）；
// vote/stake/delegate 链上动作非价值移动不记账。空串=不入账。
func TRXPropKind(txKind string) string {
	switch txKind {
	case "rental_income", "vote_claim", "deposit":
		return "owner_in"
	case "withdraw":
		return "owner_out"
	}
	return ""
}
