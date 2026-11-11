package guard

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// 方向族亏损熔断测试锚（工单#10413 验收——模拟族触线全平事件链+闸阈配置化）。

// dirClean 方向件测面自扫（setup() 只清 guard-test 面——本件 dir-test/
// latch/esc 行独立清，跨测不渗漏）。
func dirClean(t *testing.T, ctx context.Context, w *Warden) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM fin_cashflow WHERE position_ref IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='dir-test'))`,
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='dir-test'))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='dir-test')`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='dir-test'))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='dir-test')`,
		`DELETE FROM fin_strategy_transition WHERE strategy_id IN
		   (SELECT id FROM fin_strategy WHERE template_ref='dir-test')`,
		`DELETE FROM fin_strategy WHERE template_ref='dir-test'`,
		`DELETE FROM fin_opp_template WHERE template='dir-test'`,
		`DELETE FROM app_config WHERE key LIKE 'guard.dir_broken.%'`,
		`DELETE FROM ops_event WHERE esc_scope='dir_loss_breaker'`,
		`DELETE FROM md_ticks WHERE inst_id='ETH-USDT-SWAP'`,
		`DELETE FROM md_equity_curve WHERE src='test'`,
	} {
		if _, err := w.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
}

// mkDirPosition 造方向位 fixture：策略 template_ref 可调+单/双腿同侧播种
// （方向位=腿集全同边——side 全 buy 或全 sell；px/qty_base 随参供浮亏算）。
func mkDirPosition(t *testing.T, ctx context.Context, w *Warden, template, stratState string,
	legs []struct {
		inst, side string
		qty, px    float64
	}, openedAt time.Time) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('ETH-USDT-SWAP',$1,$2,100,'okx') RETURNING id`,
		template, stratState).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'ETH-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range legs {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,px,state,sz_unit,qty_base)
			VALUES($1,$2,$3,'okx',$4,$5,$6,$7,'filled','base',$6)`,
			execID, i+1, fmt.Sprintf("cl-%d-%s", execID, lg.inst), lg.side, lg.inst, lg.qty, lg.px); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'ETH-USDT-SWAP',100,$3) RETURNING id`, execID, stratID, openedAt).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return posID
}

func dirSeedNav(t *testing.T, ctx context.Context, w *Warden, nav float64) {
	t.Helper()
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES(now(),'ledger','NAV',$1,'test')`, nav); err != nil {
		t.Fatal(err)
	}
}

func dirBrokenDB(t *testing.T, ctx context.Context, w *Warden, fam string) bool {
	t.Helper()
	var v string
	if err := w.pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key=$1", dirLatchKey(fam)).Scan(&v); err != nil {
		return false
	}
	return v == "true"
}

// TestDirTripFreezesDegradesFlattens 触线动作链：latch→模板禁→族策略降级→升件→全平。
// 方向位=双腿全 buy（同边净裸 delta）；浮亏=mark(80) vs vwap(100)×qty1 两腿=-40；
// 实现=trade_pnl -80——合计 -120 ≥ 预算 100(=NAV1000×10%)。
func TestDirTripFreezesDegradesFlattens(t *testing.T) {
	ctx, w, fx := setup(t)
	dirClean(t, ctx, w)
	dirSeedNav(t, ctx, w, 1000)
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_opp_template(template,venue,surface,enabled,state)
		VALUES('dir-test','okx','x',true,'enabled') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	posID := mkDirPosition(t, ctx, w, "dir-test", "live",
		[]struct {
			inst, side string
			qty, px    float64
		}{
			{"ETH-USDT-SWAP", "buy", 1, 100}, {"ETH-USDT-SWAP", "buy", 1, 100},
		}, time.Now().Add(-time.Hour))
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','ETH-USDT-SWAP',80)
		ON CONFLICT (venue,inst_id) DO UPDATE SET last=80,ts=now()`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','okx','USDT',-80,$1,1,'fin_order_leg',now())`, posID); err != nil {
		t.Fatal(err)
	}
	w.dirScan(ctx)
	if !dirBrokenDB(t, ctx, w, "dir-test") {
		t.Fatal("latch 未落 guard.dir_broken.dir-test")
	}
	var enabled bool
	if err := w.pool.QueryRow(ctx,
		`SELECT enabled FROM fin_opp_template WHERE template='dir-test'`).Scan(&enabled); err != nil || enabled {
		t.Fatalf("模板未禁 enabled=%v err=%v", enabled, err)
	}
	var st string
	if err := w.pool.QueryRow(ctx,
		`SELECT state FROM fin_strategy WHERE template_ref='dir-test'`).Scan(&st); err != nil || st != "degraded" {
		t.Fatalf("策略未降级 state=%s err=%v", st, err)
	}
	if w.rej.Snapshot()["guard/dir_loss_breaker_tripped"] == 0 {
		t.Fatal("tripped 计数门未计")
	}
	if len(fx.calls) != 1 || fx.calls[0].InstID != "ETH-USDT-SWAP" {
		t.Fatalf("全平调用=%+v", fx.calls)
	}
	var esc int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='dir_loss_breaker'`).Scan(&esc); err != nil || esc != 1 {
		t.Fatalf("熔断升件=%d（应=1）", esc)
	}
}

// TestDirBelowBudgetNoTrip 阈下不误触——亏损 50 < 预算 100 零动作。
func TestDirBelowBudgetNoTrip(t *testing.T) {
	ctx, w, fx := setup(t)
	dirClean(t, ctx, w)
	dirSeedNav(t, ctx, w, 1000)
	posID := mkDirPosition(t, ctx, w, "dir-test", "live",
		[]struct {
			inst, side string
			qty, px    float64
		}{
			{"ETH-USDT-SWAP", "buy", 1, 100}, {"ETH-USDT-SWAP", "buy", 1, 100},
		}, time.Now().Add(-time.Hour))
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','ETH-USDT-SWAP',100)
		ON CONFLICT (venue,inst_id) DO UPDATE SET last=100,ts=now()`); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','okx','USDT',-50,$1,1,'fin_order_leg',now())`, posID); err != nil {
		t.Fatal(err)
	}
	w.dirScan(ctx)
	if dirBrokenDB(t, ctx, w, "dir-test") || len(fx.calls) != 0 {
		t.Fatal("阈下误触")
	}
}

// TestDirHedgedNotWatched 对冲双腿（buy+sell）不入方向闸域——腿集双边自然排除。
func TestDirHedgedNotWatched(t *testing.T) {
	ctx, w, _ := setup(t)
	dirClean(t, ctx, w)
	dirSeedNav(t, ctx, w, 1000)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','okx','USDT',-500,$1,1,'fin_order_leg',now())`, posID); err != nil {
		t.Fatal(err)
	}
	w.dirScan(ctx)
	if dirBrokenDB(t, ctx, w, "guard-test") {
		t.Fatal("对冲族误入方向闸域")
	}
}

// TestDirBrokenLatchResume latch 态续平——latch 在而仓未净每拍续平不重复熔断升件。
func TestDirBrokenLatchResume(t *testing.T) {
	ctx, w, fx := setup(t)
	dirClean(t, ctx, w)
	if _, err := w.pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('guard.dir_broken.dir-test','true'::jsonb)`); err != nil {
		t.Fatal(err)
	}
	mkDirPosition(t, ctx, w, "dir-test", "degraded",
		[]struct {
			inst, side string
			qty, px    float64
		}{
			{"ETH-USDT-SWAP", "buy", 1, 100}, {"ETH-USDT-SWAP", "buy", 1, 100},
		}, time.Now().Add(-time.Hour))
	w.dirScan(ctx)
	if len(fx.calls) != 1 {
		t.Fatalf("latch 续平调用=%d（应=1）", len(fx.calls))
	}
	var esc int
	if err := w.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_scope='dir_loss_breaker'`).Scan(&esc); err != nil || esc != 0 {
		t.Fatalf("latch 态重复升件=%d（应=0）", esc)
	}
}

// TestDirMissingNavFailClosed NAV 陈/缺=本拍不判（fail-closed 不虚构基线）。
func TestDirMissingNavFailClosed(t *testing.T) {
	ctx, w, fx := setup(t)
	dirClean(t, ctx, w)
	mkDirPosition(t, ctx, w, "dir-test", "live",
		[]struct {
			inst, side string
			qty, px    float64
		}{
			{"ETH-USDT-SWAP", "buy", 1, 100}, {"ETH-USDT-SWAP", "buy", 1, 100},
		}, time.Now().Add(-time.Hour))
	w.dirScan(ctx) // 无 NAV 行（ddline 测面清库）——不判不平
	if len(fx.calls) != 0 || dirBrokenDB(t, ctx, w, "dir-test") {
		t.Fatal("NAV 缺席误触")
	}
}
