// drill_cfd_test.go —— ShapeCFDSingleLeg 单腿演习 e2e（O-41368
// mt5_dir_mom 产能凭据锚）：开=ExecuteDir 一张 buy CFD 腿（venue=
// 'exness' mtexec 路由），平=哨戒 degraded→tend 单腿径→ExecuteDir
// sell 平；断言 venue='exness' 腿行+仓行闭+凭据落账 pass。
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

func cfdSetup(t *testing.T) (context.Context, *Drill) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill')))`,
		`DELETE FROM fin_cashflow WHERE position_ref IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill')))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill'))`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill')))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill'))`,
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill'))`,
		"DELETE FROM fin_strategy WHERE template_ref IN ('mt5_dir_mom','simdrill')",
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
	var stratID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES('EURUSD','mt5_dir_mom','probe',30,'mt5','long_perp') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1000, nil })
	return ctx, &Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: simvenue.New()}
}

// CFD 单腿全径：开=一张 buy EURUSD CFD（$30/px≈~2000/ctVal0.001→lots 网格），
// 平=哨戒 degraded→sell 同量；腿 venue='exness' 直记，仓行闭，凭据 pass。
func TestDrillCFDSingleLegFullPath(t *testing.T) {
	ctx, d := cfdSetup(t)
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='mt5_dir_mom' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "pass" {
		t.Fatalf("CFD 单腿全径应绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	if res.Trigger != "strategy_degraded" {
		t.Fatalf("触发应降级联动: %q", res.Trigger)
	}
	var venue, side string
	var qty float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT venue,side,sz FROM fin_order_leg
		 WHERE execution_id=$1`, res.OpenExecID).Scan(&venue, &side, &qty); err != nil {
		t.Fatal(err)
	}
	if venue != "exness" || side != "buy" || qty <= 0 {
		t.Fatalf("开腿应=exness/buy/正手数: venue=%s side=%s sz=%.6f", venue, side, qty)
	}
	var closed bool
	if err := d.Pool.QueryRow(ctx,
		`SELECT state='closed' FROM fin_position WHERE execution_id=$1`, res.OpenExecID).Scan(&closed); err != nil || !closed {
		t.Fatalf("仓行未闭: %v", err)
	}
	var v string
	if err := d.Pool.QueryRow(ctx,
		`SELECT verdict FROM fin_exec_drill WHERE strategy_ref=$1 ORDER BY id DESC LIMIT 1`,
		stratID).Scan(&v); err != nil || v != "pass" {
		t.Fatalf("凭据应落 pass: %v %q", err, v)
	}
	if math.IsNaN(qty) {
		t.Fatal("sz 非数")
	}
}
