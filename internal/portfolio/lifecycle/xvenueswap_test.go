package lifecycle

// xvenueswap_test.go —— xvenue mt5 腿 swap 持续记账桩测（T-0377 断链补焊③
// 验收锚②）：方向腿年化取键（long_cfd↔swap_long/short_cfd↔swap_short）+
// Δh 摊提+符号（负=成本）+ts 门防重摊+ann 零/缺/非副腿向跳过如实。
import (
	"context"
	"testing"
)

// seedSwapLeg 副 mt5 腿策略行（paper/notional 100/entered_at 2h 前）。
func seedSwapLeg(t *testing.T, ctx context.Context, r *Runner, inst, dir string) int64 {
	t.Helper()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,direction,venue,entered_at)
		VALUES($1,'xvenue-carry','paper',100,$2,'mt5',now()-interval '2 hours') RETURNING id`,
		inst, dir).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// 锚②：long_cfd 腿吃 swap_long_ann_bps 摊提（负=成本）+ts 门防重摊+新行续摊。
func TestXvenueSwapAccrualLongCfd(t *testing.T) {
	ctx, r := xsetup(t)
	sid := seedSwapLeg(t, ctx, r, "BTCUSDm", "long_cfd")
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('mt5_swap','BTCUSDm',0,'{"swap_long_ann_bps":-794.336,"swap_short_ann_bps":0,"ann_ok":true}',now()-interval '1 hour')`); err != nil {
		t.Fatal(err)
	}
	if err := r.xvenueSwapAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*), COALESCE(sum(amount_usd),0) FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='swap'`, sid).Scan(&n, &amt); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("应落 1 行 swap 账 got %d", n)
	}
	// −794.336bps×100USD×1h/(365×24h)≈−0.009067（负=成本对拍锚②同源）。
	want := -794.336 / 10000 * 100 / (365 * 24)
	if amt > want*0.999 || amt < want*1.001 || amt >= 0 {
		t.Fatalf("摊提值对拍败 got %f want≈%f", amt, want)
	}
	// 同信号行重跑=零重摊（ts 门）。
	if err := r.xvenueSwapAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill WHERE strategy_id=$1 AND kind='swap'`, sid).Scan(&n)
	if n != 1 {
		t.Fatalf("ts 门应防重摊 got %d 行", n)
	}
	// 新信号行（年化漂移 −800）→续摊 Δh。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('mt5_swap','BTCUSDm',0,'{"swap_long_ann_bps":-800,"ann_ok":true}',now())
		ON CONFLICT (kind,key) DO UPDATE SET meta=EXCLUDED.meta, ts=EXCLUDED.ts`); err != nil {
		t.Fatal(err)
	}
	if err := r.xvenueSwapAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill WHERE strategy_id=$1 AND kind='swap'`, sid).Scan(&n)
	if n != 2 {
		t.Fatalf("新行应续摊 got %d 行", n)
	}
}

// short_cfd 腿吃 swap_short_ann_bps（正=收入）；ann 零=跳过如实；非副腿向跳过。
func TestXvenueSwapShortCfdAndSkips(t *testing.T) {
	ctx, r := xsetup(t)
	shortID := seedSwapLeg(t, ctx, r, "ETHUSDm", "short_cfd")
	zeroID := seedSwapLeg(t, ctx, r, "SOLUSDm", "long_cfd")
	oddID := seedSwapLeg(t, ctx, r, "XAUUSDm", "gap_carry")
	for _, q := range []string{
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','ETHUSDm',0,'{"swap_short_ann_bps":50,"swap_long_ann_bps":-796,"ann_ok":true}',now()-interval '1 hour')`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','SOLUSDm',NULL,'{"swap_long_ann_bps":0,"ann_ok":false}',now()-interval '1 hour')`,
		`INSERT INTO md_signal(kind,key,val,meta,ts) VALUES('mt5_swap','XAUUSDm',0,'{"swap_long_ann_bps":-100,"ann_ok":true}',now()-interval '1 hour')`,
	} {
		if _, err := r.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.xvenueSwapAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var amt float64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_usd),0) FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='swap'`, shortID).Scan(&amt); err != nil {
		t.Fatal(err)
	}
	// 50bps×100USD×1h/8760h≈+0.005708（正=收入）。
	if want := 50.0 / 10000 * 100 / (365 * 24); amt < want*0.999 || amt > want*1.001 || amt <= 0 {
		t.Fatalf("short_cfd 腿应正账 got %f want≈%f", amt, want)
	}
	for _, id := range []int64{zeroID, oddID} {
		var n int
		r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill WHERE strategy_id=$1`, id).Scan(&n)
		if n != 0 {
			t.Fatalf("ann 零/非副腿向应零账 got %d（strategy %d）", n, id)
		}
	}
}

// T-0377 方向符号根修锚：long_perp 腿正 rate 付租（负账）——不再恒按 short 记。
func TestPaperFundingDirectionSign(t *testing.T) {
	ctx, r := xsetup(t)
	var longID, shortID int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,direction,venue)
		VALUES('XP-USDT-SWAP','xvenue-carry','paper',100,'long_perp_short_cfd','okx') RETURNING id`).Scan(&longID); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,state,notional_usd,direction,venue)
		VALUES('XS-USDT-SWAP','xvenue-carry','paper',100,'short_perp_long_cfd','okx') RETURNING id`).Scan(&shortID); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'XP-USDT-SWAP',0.01), (now(),'XS-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.paperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var longAmt, shortAmt float64
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_usd),0) FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='funding'`, longID).Scan(&longAmt); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount_usd),0) FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='funding'`, shortID).Scan(&shortAmt); err != nil {
		t.Fatal(err)
	}
	if longAmt != -1.0 { // 0.01×100=1 付租→−1
		t.Fatalf("long_perp 正 rate 应付租（−1）got %f", longAmt)
	}
	if shortAmt != 1.0 { // short_perp 正 rate 收租→+1
		t.Fatalf("short_perp 正 rate 应收租（+1）got %f", shortAmt)
	}
}
