// stakecarry_paper_test.go —— 工单#21612 锚：stake-carry paper 双腿记账——
// stake 腿=trx_onchain 现率按日摊（vote+rental 口径）；okx 空腿=md_funding
// 行逐条 +rate*notional 实收形（非摊提）；cmc 空腿=30d 实收日均率按日摊；
// 率缺/样本不足=让位不造 fill（fail-closed 同 dcd 径）。
package lifecycle

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// stakeStrategy 造 paper 策略行（entered_at 回 2 天触发日摊节拍）。
func stakeStrategy(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst string, notional float64) int64 {
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,entered_at)
		VALUES($1,'stake-carry','paper',$2,'stake_carry',now()-interval '2 days') RETURNING id`,
		inst, notional).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func TestStakeCarryAccounting(t *testing.T) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_paper_fill", "DELETE FROM fin_strategy",
		"DELETE FROM fin_trx_vote_apr", "DELETE FROM fin_trx_account",
		"DELETE FROM fin_trx_txs", "DELETE FROM md_funding WHERE inst_id='TRX-USDT-SWAP'"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// 链上率源种子：vote 4% + rental 快照（delegated=1e9 sun=1000TRX、7d
	// 实收 1.92TRX→rental≈(1.92/7*365)/1000*100≈10.01%——产线 9.4% 真谱形）。
	for _, q := range []string{
		`INSERT INTO fin_trx_vote_apr(current_rate) VALUES(4.0)`,
		`INSERT INTO fin_trx_account(address,votes_total,balance_sun,allowance_sun,energy_delegated,bandwidth_staked,net_limit)
			VALUES('TX',0,0,0,1000000000,0,0)`,
		`INSERT INTO fin_trx_txs(tx_id,kind,amount_sun,ts) VALUES('tx-stake-1','rental_income',1920000,now())`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// okx 空腿 funding 三行（两窗内一窗外——窗内行全记、窗外不追）。
	old := time.Now().Add(-3 * 24 * time.Hour)
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(inst_id,funding_rate,ts) VALUES
		('TRX-USDT-SWAP',0.0001,$1),('TRX-USDT-SWAP',0.0002,now()),('TRX-USDT-SWAP',0.00005,$2)`,
		old, time.Now().Add(-47*time.Hour)); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	sid := stakeStrategy(t, ctx, pool, "okx:TRX", 1000)
	if err := r.stakeCarryAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	// stake 腿 fill：apr=(4.0+10.01)/100≈0.1401×2d×1000/365≈0.767
	var stakeAmt float64
	if err := pool.QueryRow(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND inst_id='okx:TRX:stake'`, sid).Scan(&stakeAmt); err != nil {
		t.Fatalf("stake 腿 fill 缺: %v", err)
	}
	if stakeAmt < 0.74 || stakeAmt > 0.80 {
		t.Fatalf("stake 腿摊提额 %v 出界（≈0.77）", stakeAmt)
	}
	// 空腿 fill：窗内两行（-72h 行在 entered_at=-48h 窗外不追）
	var n int
	var sum float64
	if err := pool.QueryRow(ctx, `SELECT count(*),COALESCE(sum(amount_usd),0) FROM fin_paper_fill
		WHERE strategy_id=$1 AND inst_id='TRX-USDT-SWAP'`, sid).Scan(&n, &sum); err != nil {
		t.Fatal(err)
	}
	if n != 2 || sum < 0.24 || sum > 0.26 {
		t.Fatalf("okx 空腿 fill n=%d sum=%v（want 2 行 ≈0.25）", n, sum)
	}
}

func TestStakeCarryNoYieldSilent(t *testing.T) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_paper_fill", "DELETE FROM fin_strategy",
		"DELETE FROM fin_trx_vote_apr", "DELETE FROM fin_trx_account", "DELETE FROM fin_trx_txs"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r := NewRunner(pool, cfg)
	sid := stakeStrategy(t, ctx, pool, "okx:TRX", 1000)
	if err := r.stakeCarryAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill
		WHERE strategy_id=$1 AND inst_id='okx:TRX:stake'`, sid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("率缺应让位不造 stake fill（got %d）", n)
	}
}
