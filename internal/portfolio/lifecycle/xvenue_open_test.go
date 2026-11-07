// xvenue_open_test.go —— T-0145 openLive 路由桩：xvenue-carry 主行（venue=
// 'okx'）落缝/副腿行（venue='mt5'）永不独立开仓；lifetimeOK 向位前缀判
// （long_perp_* 收负租径）；breakevenOK 豁免（上游 eval 已四件分解）。
package lifecycle

import (
	"context"
	"testing"
)

// mkXvenueStrategy xvenue 行造具——venue/template/direction/proposal_ref 可编程。
func mkXvenueStrategy(t *testing.T, ctx context.Context, r *Runner,
	inst, venue, direction string, proposalRef int64, state string) int64 {
	t.Helper()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,venue,proposal_ref,entered_at)
		VALUES($1,'xvenue-carry',$2,50,$3,$4,$5,now()) RETURNING id`,
		inst, state, direction, venue, proposalRef).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// 锚⑤a：副腿行（venue='mt5'）永不独立开仓——主行驱动双腿原子序；
// 主行正常落缝（暗发闸在 cmd 层非 lifecycle 面）。
func TestXvenueSubLegNeverOpens(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	// 同 proposal 双行——副腿行先于主行入册（序位覆盖）。
	mkXvenueStrategy(t, ctx, r, "BTCUSDm", "mt5", "long_perp_short_cfd", 77, "live")
	mainID := mkXvenueStrategy(t, ctx, r, "BTC-USDT-SWAP", "okx", "long_perp_short_cfd", 77, "live")
	// long_perp 收负租径——负 funding=健康（非翻面）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'BTC-USDT-SWAP',-0.0006,-0.0004)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 1 || fo.calls[0].StrategyRef != mainID {
		t.Fatalf("副腿行越闸/主行未开: %+v", fo.calls)
	}
	if r.rej.Snapshot()["lifecycle/xvenue_sub_leg"] == 0 {
		t.Fatal("副腿行跳过计数未计")
	}
}

// 锚⑤b：lifetimeOK 向位前缀判——long_perp_short_cfd（perp 多=收负 funding
// 方 sign=-1）：正 funding>flip_bps=收付向反题=拒开；负 funding=健康放行。
// （旧等值串判 cfd 向位漏判=符号反置病灶——根修断言。）
func TestXvenueLifetimeDirectionSign(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	mkXvenueStrategy(t, ctx, r, "BTC-USDT-SWAP", "okx", "long_perp_short_cfd", 77, "live")
	// 正 funding 0.0006：long_perp 径=付正租=反题——lifetime_below_min 拒。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'BTC-USDT-SWAP',0.0006,0.0004)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("long_perp 正 funding 反题仍开——sign 判未修: %+v", fo.calls)
	}
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_eval_reject
		WHERE inst_id='BTC-USDT-SWAP' ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil || reason != "lifetime_below_min" {
		t.Fatalf("拒因: %s %v", reason, err)
	}
}

// 锚⑤c：short_perp_long_cfd（perp 空=收正 funding sign=+1）+薄 funding=
// lifetimeOK 过径+breakevenOK 豁免证——薄 funding（回本窗必拒域）仍放行
// 落缝（上游 eval 四件分解已判，双计成本误杀防线）。
func TestXvenueBreakevenExempt(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	mainID := mkXvenueStrategy(t, ctx, r, "BTC-USDT-SWAP", "okx", "short_perp_long_cfd", 77, "live")
	// 薄 funding 0.00005 正率——funding-hedge 径 breakevenOK 必拒域；
	// xvenue 豁免=仍落缝（short_perp 收正租 lifetimeOK 过）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'BTC-USDT-SWAP',0.00005,0.00005)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 1 || fo.calls[0].StrategyRef != mainID {
		t.Fatalf("xvenue 薄 funding 被 breakevenOK 误杀（豁免未生效）: %+v", fo.calls)
	}
}
