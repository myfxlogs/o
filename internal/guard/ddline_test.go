package guard

import (
	"context"
	"fmt"
	"testing"
	"time"

	"o/internal/risk"
)

// 回撤硬线测试锚（T-0025 段一·验收标准一）——共享库纪律：md_equity_curve/快照行
// 属本包 schema 隔离面（T-0021 testutil），测试首拍清本表保确定性，收官不删库行。

// ddSeed 净权益场景播种（快照行现值+曲线历史；src=test 供清扫）。
func ddSeed(t *testing.T, ctx context.Context, w *Warden, cur float64, hist map[string]float64) {
	t.Helper()
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	w.AttachEquity(func(ctx context.Context) (float64, error) { return cur, nil })
	for ago, eq := range hist {
		if _, err := w.pool.Exec(ctx,
			`INSERT INTO md_equity_curve(ts,venue,ccy,equity,src) VALUES(now()-$1::interval,'okx','USDT',$2,'test')`, ago, eq); err != nil {
			t.Fatal(err)
		}
	}
}

func killSwitchDB(t *testing.T, ctx context.Context, w *Warden) bool {
	t.Helper()
	var v string
	if err := w.pool.QueryRow(ctx,
		"SELECT value#>>'{}' FROM app_config WHERE key='risk.kill_switch'").Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v == "true"
}

// TestDDTripFlattensAndKills 触线动作链：kill_switch 先落→逐仓 dd_30d 全平（过闸反
// 向原子平）→告警计数。净权益=venue 自报 equity（会签补正①——浮盈在口径内）。
func TestDDTripFlattensAndKills(t *testing.T) {
	ctx, w, fx := setup(t)
	// 起点 HWM 100，现值 96.5——回撤 3.5% ≥ 3% 触线（史 10d<窗口 30d→HWM 径）
	ddSeed(t, ctx, w, 96.5, map[string]float64{"'10 days'": 100, "'5 days'": 95})
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("触线未落 kill_switch")
	}
	if w.rej.Snapshot()["guard/dd_hardline_tripped"] == 0 {
		t.Fatal("dd_hardline_tripped 计数门未计")
	}
	if len(fx.calls) != 1 {
		t.Fatalf("全平调用=%d（应逐仓一次）", len(fx.calls))
	}
	if fx.calls[0].InstID != "BTC-USDT-SWAP" || len(fx.calls[0].Legs) != 2 {
		t.Fatalf("平仓规格=%+v", fx.calls[0])
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || trig != ddTrigger || state != "closed" {
		t.Fatalf("仓态 closed=%v state=%v 触发=%v（应已平/closed/dd_30d）", closed, state, trig)
	}
}

// TestDDBelowThresholdZeroAction 未触线零动作（阈下不误触）。
func TestDDBelowThresholdZeroAction(t *testing.T) {
	ctx, w, fx := setup(t)
	ddSeed(t, ctx, w, 98, map[string]float64{"'10 days'": 100}) // dd=2%<3%
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("阈下误落闸")
	}
	if len(fx.calls) != 0 {
		t.Fatalf("阈下误平仓: %v", fx.calls)
	}
	if w.rej.Snapshot()["guard/dd_hardline_tripped"] != 0 {
		t.Fatal("阈下误计触线")
	}
}

// TestDDInsufficientHistoryHWM 史不足窗口=起点至今 HWM 判（会签补正②——开业首月
// 无保护洞：全亏实现面照判）。
func TestDDInsufficientHistoryHWM(t *testing.T) {
	ctx, w, fx := setup(t)
	// 史 2d（<30d 窗）但起点回撤 4%：100→96——HWM 径必触
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100, "'1 days'": 97})
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("史不足径未按起点 HWM 判触")
	}
	if len(fx.calls) != 1 {
		t.Fatalf("HWM 触线应全平: %v", fx.calls)
	}
}

// TestDDWindowRollsOff 史满窗口=30d 滚动窗：窗外旧峰不计（滚出窗即回吐豁免）。
func TestDDWindowRollsOff(t *testing.T) {
	ctx, w, _ := setup(t)
	// 40d 前峰 100（窗外），窗内 95@-10d，现值 96.5→窗内 dd=0 不触
	ddSeed(t, ctx, w, 96.5, map[string]float64{"'40 days'": 100, "'10 days'": 95})
	w.ddScan(ctx)
	if killSwitchDB(t, ctx, w) {
		t.Fatal("窗外旧峰误触线（滚动窗未滚出）")
	}
}

// TestDDNoAutoReset 触线后重启语义：无自动复位——监视面见闸落不复动作、不复位；
// open 律持续被拦、close 律放行（T-0021 语义不回退）；翻闸只认显式热改。
func TestDDNoAutoReset(t *testing.T) {
	ctx, w, fx := setup(t)
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	w.ddScan(ctx)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("前置：应已触线落闸")
	}
	n := len(fx.calls)
	// 权益修复+重启（新拍）：闸仍落——不自动复位、不重复动作
	ddSeed(t, ctx, w, 120, map[string]float64{"'2 days'": 100})
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	w.ddScan(ctx)
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("监视面自动复位了 kill_switch——越权")
	}
	if len(fx.calls) != n {
		t.Fatal("触线态重复动作")
	}
	// open 律被拦 / close 律放行（闸面语义）
	in := risk.Intent{StrategyRef: 1, Venue: "okx", InstID: "BTC-USDT-SWAP",
		Legs: []string{"sell@BTC-USDT-SWAP", "buy@BTC-USDT"}, NotionalUSD: 10, Kind: "open"}
	if dec := w.gate.Check(ctx, in); dec.Allow {
		t.Fatal("kill_switch 落下后 open 律未拦")
	}
	in.Kind = "close"
	if dec := w.gate.Check(ctx, in); !dec.Allow {
		t.Fatalf("close 律被拦（T-0021 语义回退）: %s", dec.Reason)
	}
	// 显式热改翻闸后 open 律恢复（无自动复位的反证——复位只认显式）
	if _, err := w.pool.Exec(ctx, "UPDATE app_config SET value='false' WHERE key='risk.kill_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	in.Kind = "open"
	if dec := w.gate.Check(ctx, in); !dec.Allow {
		t.Fatalf("显式翻闸后 open 仍拦: %s", dec.Reason)
	}
}

// TestDDMissingKeysFailClosed 判据键缺=不监视+告警计数（fail-closed 不虚构）。
func TestDDMissingKeysFailClosed(t *testing.T) {
	ctx, w, fx := setup(t)
	for _, k := range []string{"risk.max_dd_30d", "risk.dd_window_days"} {
		if _, err := w.pool.Exec(ctx, "DELETE FROM app_config WHERE key=$1", k); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	ddSeed(t, ctx, w, 90, map[string]float64{"'2 days'": 100}) // 深回撤场景——键缺仍不得动作
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx)
	if w.rej.Snapshot()["guard/dd_missing_param"] == 0 {
		t.Fatal("缺键告警计数未计")
	}
	if killSwitchDB(t, ctx, w) || len(fx.calls) != 0 {
		t.Fatal("缺键面不得动作（判据不虚构）")
	}
	var n int
	if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM md_equity_curve WHERE src='guard_dd'").Scan(&n); err != nil || n != 0 {
		t.Fatalf("缺键面不应采点: n=%d", n)
	}
}

// TestDDEquityStaleSkips 陈快照不采点不判（读径新鲜度门——快照禁令）。
func TestDDEquityStaleSkips(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	w.AttachEquity(func(ctx context.Context) (float64, error) { return 0, fmt.Errorf("现读败") })
	w.ddScan(ctx)
	if w.rej.Snapshot()["guard/equity_missing"] == 0 {
		t.Fatal("现读败计数未计")
	}
	var n int
	if err := w.pool.QueryRow(ctx, "SELECT count(*) FROM md_equity_curve").Scan(&n); err != nil || n != 0 {
		t.Fatalf("陈快照不应采点: n=%d", n)
	}
}

// TestDDCaptureAppendsCurve 采点面：鲜快照每拍落 md_equity_curve 事实行（append-only）。
func TestDDCaptureAppendsCurve(t *testing.T) {
	ctx, w, _ := setup(t)
	if _, err := w.pool.Exec(ctx, "DELETE FROM md_equity_curve"); err != nil {
		t.Fatal(err)
	}
	ddSeed(t, ctx, w, 100, nil)
	w.ddScan(ctx)
	var n int
	var src string
	if err := w.pool.QueryRow(ctx, `SELECT count(*),max(src) FROM md_equity_curve WHERE src='guard_dd'`).Scan(&n, &src); err != nil || n != 1 {
		t.Fatalf("采点行=%d err=%v", n, err)
	}
	_ = fmt.Sprint(src)
}

// TestDDRetryAfterFirstFail 触线续平循环：首败→次拍自动重平→closed（打回返工锚①
// ——敞口禁在最大回撤时刻无限期开放）。
func TestDDRetryAfterFirstFail(t *testing.T) {
	ctx, w, fx := setup(t)
	fx.failFirstN = 1 // 首平瞬时败
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	posID := mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	w.ddScan(ctx) // 触发拍：落闸+首平败（close_failed，attempts=1）
	if !killSwitchDB(t, ctx, w) || len(fx.calls) != 1 {
		t.Fatalf("触发拍 kill=%v calls=%d", killSwitchDB(t, ctx, w), len(fx.calls))
	}
	if closed, _, _ := posClosed(t, ctx, w, posID); closed {
		t.Fatal("首败不应已平")
	}
	w.ddScan(ctx) // 次拍：触线态续平（节拍窗内 attempts=1<3）→成
	if len(fx.calls) != 2 {
		t.Fatalf("次拍未自动重平: calls=%d", len(fx.calls))
	}
	closed, state, trig := posClosed(t, ctx, w, posID)
	if !closed || state != "closed" || trig != ddTrigger {
		t.Fatalf("重平后 closed=%v state=%v trig=%v", closed, state, trig)
	}
	if !killSwitchDB(t, ctx, w) {
		t.Fatal("闸不应复位")
	}
	w.ddScan(ctx) // 全平后归静默：无第三平
	if len(fx.calls) != 2 {
		t.Fatalf("全平后重复动作: calls=%d", len(fx.calls))
	}
}

// TestDDExhaustedPersistentAlert 重试上限耗尽→持续告警不静默（打回返工锚②；
// tend 自身耗尽升级单次，dd 续平面每拍声张耗尽仓）。
func TestDDExhaustedPersistentAlert(t *testing.T) {
	ctx, w, fx := setup(t)
	fx.failFirstN = 100 // 全败
	ddSeed(t, ctx, w, 96, map[string]float64{"'2 days'": 100})
	mkPosition(t, ctx, w, "live", 1, time.Now().Add(-time.Hour))
	for i := 0; i < 3; i++ { // 三拍耗尽上限（retry_max_attempts=3）
		w.ddScan(ctx)
	}
	if len(fx.calls) != 3 {
		t.Fatalf("耗尽前平仓尝试=%d（应 3）", len(fx.calls))
	}
	if w.rej.Snapshot()["guard/close_gave_up"] != 1 {
		t.Fatalf("tend 耗尽升级=%v（应单次）", w.rej.Snapshot()["guard/close_gave_up"])
	}
	w.ddScan(ctx)
	if w.rej.Snapshot()["guard/dd_close_exhausted"] != 1 {
		t.Fatal("耗尽仓首拍声张未计")
	}
	w.ddScan(ctx) // 后续拍持续告警（禁静默漂移）
	if w.rej.Snapshot()["guard/dd_close_exhausted"] != 2 {
		t.Fatalf("持续告警未逐拍声张: %v", w.rej.Snapshot()["guard/dd_close_exhausted"])
	}
	if len(fx.calls) != 3 {
		t.Fatal("耗尽后不应再打 venue")
	}
}
