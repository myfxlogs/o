package guard

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/risk"
	"phase/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Warden, *fakeExec) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 测试面隔离：只清本包自有面（fin_position 系 fixture 行按 guard-test 策略打标
	// 级联清——lifecycle/execution 等包并发共用库，无差别清=撞他包在途行）。
	for _, stmt := range []string{
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='guard-test'))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='guard-test')`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref='guard-test'))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='guard-test')`,
		"DELETE FROM fin_strategy WHERE template_ref='guard-test'",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{
		"guard.flip_bps": "5", "guard.max_hold_hours": "72",
		"guard.retry_max_attempts": "3", "guard.retry_backoff_seconds": "0",
		"exec.hedge_band_bps": "20",
		"risk.kill_switch":    "false", "risk.max_notional_usd": "500",
		"risk.max_venue_pct": "0.35", "risk.max_exposure_usd": "1000",
		"risk.token_ttl_seconds": "30"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	fx := &fakeExec{}
	return ctx, NewWarden(pool, cfg, risk.NewGate(pool, cfg), fx), fx
}

// fakeExec 执行缝假件——录调用、可编程序失败（execID 定值 999 供衍生仓关闭断言）。
type fakeExec struct {
	calls []CloseSpec
	fail  error
}

const fakeCloseExecID int64 = 999

func (f *fakeExec) ExecuteClose(_ context.Context, spec CloseSpec) (int64, error) {
	f.calls = append(f.calls, spec)
	return fakeCloseExecID, f.fail
}

// mkPosition 造一仓双腿 fixture（spot 买+perp 卖=正 funding 收方；名义可调），
// 返 position id。
func mkPosition(t *testing.T, ctx context.Context, w *Warden, stratState string,
	perpSz float64, openedAt time.Time) int64 {
	t.Helper()
	var stratID, execID, posID int64
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue) VALUES('BTC-USDT-SWAP','guard-test',$1,100,'okx') RETURNING id`,
		stratState).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open') RETURNING id`,
		stratID).Scan(&execID); err != nil {
		t.Fatal(err)
	}
	for i, lg := range []struct {
		inst, side string
		sz         float64
	}{
		{"BTC-USDT", "buy", 1}, {"BTC-USDT-SWAP", "sell", perpSz}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
			VALUES($1,$2,$3,'okx',$4,$5,$6,'filled')`,
			execID, i+1, fmt.Sprintf("cl-%d-%s", execID, lg.inst), lg.side, lg.inst, lg.sz); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.pool.QueryRow(ctx,
		`INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at)
		VALUES($1,$2,'BTC-USDT-SWAP',100,$3) RETURNING id`, execID, stratID, openedAt).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	return posID
}

func seedFacts(t *testing.T, ctx context.Context, w *Warden, rate float64) {
	t.Helper()
	for _, f := range []struct{ inst, last string }{
		{"BTC-USDT", "100"}, {"BTC-USDT-SWAP", "100"}} {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx',$1,$2)`, f.inst, f.last); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_funding(ts,venue,inst_id,funding_rate) VALUES(now(),'okx','BTC-USDT-SWAP',$1)`, rate); err != nil {
		t.Fatal(err)
	}
}

func posClosed(t *testing.T, ctx context.Context, w *Warden, posID int64) (bool, string, string) {
	t.Helper()
	var closedAt *time.Time
	var state, trig *string
	if err := w.pool.QueryRow(ctx,
		`SELECT p.closed_at, g.state::text, g.trigger_kind::text
		 FROM fin_position p LEFT JOIN fin_guard_close g ON g.position_id=p.id
		 WHERE p.id=$1`, posID).Scan(&closedAt, &state, &trig); err != nil {
		t.Fatal(err)
	}
	if state == nil {
		return false, "open", "" // 无 guard 行=在仓未触发
	}
	return closedAt != nil, *state, *trig
}

func reloadCfg(t *testing.T, ctx context.Context, w *Warden) {
	t.Helper()
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// 锚①：funding 翻面——rate=−10bps 与 sell 腿（收方 +1）异号且 |10bps|>flip_bps(5)。
func TestFundingFlipTrigger(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 1 {
		t.Fatalf("期望一次平仓执行，got %d", len(fx.calls))
	}
	// 反向腿锚：spot buy→sell、perp sell→buy，inst/sz 沿原仓。
	if fx.calls[0].Legs[0].Side != "sell" || fx.calls[0].Legs[1].Side != "buy" ||
		fx.calls[0].Legs[0].InstID != "BTC-USDT" || fx.calls[0].Legs[1].InstID != "BTC-USDT-SWAP" {
		t.Fatalf("双腿反向规格不符: %+v", fx.calls[0].Legs)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || state != "closed" || trig != "funding_flip" {
		t.Fatalf("仓应 closed/funding_flip，got closed=%v state=%q trig=%q", closed, state, trig)
	}
	var kind, decision string
	if err := w.pool.QueryRow(ctx,
		"SELECT kind::text,decision::text FROM fin_risk_decision ORDER BY id DESC LIMIT 1").Scan(&kind, &decision); err != nil {
		t.Fatal(err)
	}
	if kind != "close" || decision != "allow" {
		t.Fatalf("决策行应 close/allow，got %s/%s", kind, decision)
	}
}

// 锚②：delta 漂移——腿名义 100 vs 200（中差 6667bps>band 20bps），funding 正常不触发翻面。
func TestDeltaDriftTrigger(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 2, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "delta_drift" {
		t.Fatalf("仓应 closed/delta_drift，got state=%q trig=%q calls=%d", state, trig, len(fx.calls))
	}
}

// 锚③：持仓超龄——100h>max_hold(72h)，facts 正常不先触发。
func TestMaxHoldTrigger(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-100*time.Hour))
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "max_hold" {
		t.Fatalf("仓应 closed/max_hold，got state=%q trig=%q", state, trig)
	}
}

// 锚④：策略降级联动——degraded 必平（facts 全正常也平）。
func TestStrategyDegradedTrigger(t *testing.T) {
	ctx, w, _ := setup(t)
	seedFacts(t, ctx, w, 0.0001)
	posID := mkPosition(t, ctx, w, "degraded", 1, time.Now())
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	_, state, trig := posClosed(t, ctx, w, posID)
	if state != "closed" || trig != "strategy_degraded" {
		t.Fatalf("仓应 closed/strategy_degraded，got state=%q trig=%q", state, trig)
	}
}

// 锚⑤：kill_switch=true 下 close 仍放行（E2E 过仓巡径）；失败→close_failed→节拍重试→closed。
func TestCloseRetryAfterFailure(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx, `UPDATE app_config SET value='"true"' WHERE key='risk.kill_switch'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	fx.fail = errors.New("venue 歧义")
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, _ := posClosed(t, ctx, w, posID)
	if closed || state != "close_failed" {
		t.Fatalf("执行败应 close_failed，got closed=%v state=%q", closed, state)
	}
	var attempts int
	if err := w.pool.QueryRow(ctx,
		"SELECT attempts FROM fin_guard_close WHERE position_id=$1", posID).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=1 期望，got %d", attempts)
	}
	fx.fail = nil
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || state != "closed" || trig != "funding_flip" {
		t.Fatalf("重试应 closed，got closed=%v state=%q trig=%q", closed, state, trig)
	}
	if len(fx.calls) != 2 {
		t.Fatalf("两次执行调用期望，got %d", len(fx.calls))
	}
}

// 锚⑥：重试耗尽——attempts 达 guard.retry_max_attempts 后不再扫（升级不沉默）。
func TestCloseGiveUp(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001)
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	fx.fail = errors.New("venue 持续败")
	for i := 0; i < 3; i++ {
		if err := w.scanOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if len(fx.calls) != 3 {
		t.Fatalf("三次重试期望，got %d", len(fx.calls))
	}
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 3 {
		t.Fatalf("耗尽后应停扫，got %d 次调用", len(fx.calls))
	}
	_, state, _ := posClosed(t, ctx, w, posID)
	if state != "close_failed" {
		t.Fatalf("耗尽仓应停留 close_failed 待人工，got %q", state)
	}
	if n := w.rej.Snapshot()["guard/close_gave_up"]; n != 1 {
		t.Fatalf("close_gave_up 计数=1 期望，got %d", n)
	}
}

// 锚⑦：判据键缺=fail-closed 跳过该触发面（仓不误平、循环不炸）。
func TestMissingKeyFailClosed(t *testing.T) {
	ctx, w, fx := setup(t)
	seedFacts(t, ctx, w, -0.001) // 翻面条件成立
	posID := mkPosition(t, ctx, w, "live", 1, time.Now())
	if _, err := w.pool.Exec(ctx, `DELETE FROM app_config WHERE key='guard.flip_bps'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	if err := w.scanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fx.calls) != 0 {
		t.Fatalf("flip_bps 缺键应跳过触发不误平，got %+v", fx.calls)
	}
	closed, _, _ := posClosed(t, ctx, w, posID)
	if closed {
		t.Fatal("仓应仍在")
	}
	// 节拍键缺=整轮 fail-closed 报错不炸。
	if _, err := w.pool.Exec(ctx, `DELETE FROM app_config WHERE key='guard.retry_backoff_seconds'`); err != nil {
		t.Fatal(err)
	}
	reloadCfg(t, ctx, w)
	if err := w.scanOnce(ctx); err == nil {
		t.Fatal("retry 键缺应整轮报错 fail-closed")
	}
}
