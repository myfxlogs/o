// drill_dir_test.go —— ShapeOKXSingleLeg 单腿演习 e2e（棒5c dir_perp_mom
// 产能凭据锚）：开=ExecuteDir 一张 sell SWAP 腿，平=哨戒 degraded→tend
// 单腿径→ExecuteDir buy 平；断言 venue 合约净持归零+凭据落账 pass。
package simdrill

import (
	"context"
	"math"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/simvenue"
)

func dirSetup(t *testing.T) (context.Context, *Drill) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill')))`,
		`DELETE FROM fin_cashflow WHERE position_ref IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill')))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill'))`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill')))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill'))`,
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill'))`,
		"DELETE FROM fin_strategy WHERE template_ref IN ('dir_perp_mom','simdrill')",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{
		"invest.lifecycle.live_switch": "true",
		"exec.hedge_band_bps":          "20",
		"guard.flip_bps":               "5", "guard.max_hold_hours": "72",
		"guard.retry_max_attempts": "3", "guard.retry_backoff_seconds": "0",
		"guard.dir_loss_budget_pct": "0.10",
		"risk.kill_switch":          "false", "risk.max_notional_usd": "500",
		"risk.max_venue_pct": "0.35", "risk.max_exposure_usd": "1000",
		"risk.token_ttl_seconds": "30",
		"risk.max_dd_30d":        "0.03", "risk.dd_window_days": "30",
		"risk.dir.loss_budget_pct": "0.10", "risk.dir.trade_loss_pct": "0.005",
		"risk.dir.max_notional_pct": "0.20", "lifecycle.dir.stop_frac": "0.15"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim','CORE-USDT-SWAP',0.0125)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`); err != nil {
		t.Fatal(err)
	}
	var stratID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES('CORE-USDT-SWAP','dir_perp_mom','probe',30,'okx','short_perp') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1000, nil })
	venue := simvenue.New().
		WithInstrument(simvenue.Instrument{InstID: "CORE-USDT-SWAP", InstType: "SWAP",
			LotSz: 0.01, MinSz: 0.01, TickSz: 0.0001, CtVal: 100, CtValCcy: "CORE",
			TakerFee: 0.0005, Px: 0.0125}).
		WithBalance("USDT", 1000)
	return ctx, &Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: venue}
}

// 单腿全径：开=一张 sell SWAP（$30/ctVal100/px0.0125=400 张），平=buy 同量；
// venue 合约净持归零，凭据 verdict=pass。
func TestDrillDirSingleLegFullPath(t *testing.T) {
	ctx, d := dirSetup(t)
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='dir_perp_mom' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "pass" {
		t.Fatalf("单腿全径应绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	if res.Trigger != "strategy_degraded" {
		t.Fatalf("触发应降级联动: %q", res.Trigger)
	}
	var side, szUnit string
	var qty float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT side,COALESCE(sz_unit,''),sz FROM fin_order_leg
		 WHERE execution_id=$1`, res.OpenExecID).Scan(&side, &szUnit, &qty); err != nil {
		t.Fatal(err)
	}
	if side != "sell" || math.Abs(qty-24) > 1e-6 {
		t.Fatalf("开腿应=sell 24 张: side=%s sz_unit=%s sz=%.6f", side, szUnit, qty)
	}
	if p := d.Venue.Position("CORE-USDT-SWAP"); math.Abs(p) > 0.01 {
		t.Fatalf("venue 合约净持未归零: %.6f 张", p)
	}
}
