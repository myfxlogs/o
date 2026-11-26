package earn

// O-41364 抵押保底桩测锚（spec 验收①②③+截量+泊资+冷却）：
// ①exec_enabled∧avail<reserve→意向豁免让位零动钱；②exec=false→
// reserve 失效照常扫；③reserve 键缺=fail-closed 豁免；④avail≥reserve
// →只扫超部（sweepable=idle−floor，禁无限豁免）；⑤短额>0→泊资
// RedeemToTrading 摆回；⑥冷却窗内不重复发起；⑦无短额不动作。

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/config"
)

// collateralKeys 保底判据三键+扫阈（exec/reserve 由各测覆写）。
func collateralKeys(exec, reserve string) map[string]string {
	return map[string]string{
		"earn.sweep_min_usd":            "100",
		"margin.exec_enabled":           exec,
		"margin.collateral_reserve_usd": reserve,
	}
}

// attachIdleUSDT 交易户可用现读件（w.idle=USDT availBal≈USD 口径——
// 判据⑥与泊资判定同源）。
func attachIdleUSDT(w *Worker, avail float64) {
	w.AttachIdle(func(context.Context) (float64, error) { return avail, nil })
}

// TestCollateralExemptBelowReserve ①exec=true∧avail=150<reserve=500→
// sweepIdle 全段让位零动钱（funding 段同罩——判据⑥复合源同格）+计数门。
func TestCollateralExemptBelowReserve(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, collateralKeys("true", "500"))
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachBothIdle(w) // funding 500+trading 150 双源俱超阈
	attachIdleUSDT(w, 150)
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "抵押保底下沿")
	if w.rej.Snapshot()["earn/sweep_intent_exempt"] == 0 {
		t.Fatal("sweep_intent_exempt 计数门未计")
	}
}

// TestCollateralExecOffReserveDead ②exec=false→reserve 失效不豁免——
// avail=600 全量照扫（回滚径=闸本身实证）。
func TestCollateralExecOffReserveDead(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, collateralKeys("false", "500"))
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachIdleUSDT(w, 600)
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) == 0 || len(fake.purchases) == 0 {
		t.Fatalf("exec=false 应照常扫 transfers=%v purchases=%v", fake.transfers, fake.purchases)
	}
}

// TestCollateralReserveKeyMissing ③exec=true∧reserve 键缺=fail-closed
// 豁免（宁停泊不误扫同 intentExempt 缺省律）——迁移种子键删后重载快照。
func TestCollateralReserveKeyMissing(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":  "100",
		"margin.exec_enabled": "true",
	})
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	if _, err := w.pool.Exec(ctx,
		"DELETE FROM app_config WHERE key='margin.collateral_reserve_usd'"); err != nil {
		t.Fatal(err)
	}
	cfg2 := config.New(w.pool)
	if err := cfg2.Load(ctx); err != nil {
		t.Fatal(err)
	}
	w.cfg = cfg2
	attachBothIdle(w)
	attachIdleUSDT(w, 150)
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	noMoves(t, fake, "reserve 键缺 fail-closed")
}

// TestCollateralSweepExcessOnly ④avail=700≥floor=500→判据⑥放行，
// 交易户段只扫超部 sweepable=200（禁无限豁免——超部照扫不误钉死）。
func TestCollateralSweepExcessOnly(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, collateralKeys("true", "500"))
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachIdleUSDT(w, 700)
	if err := w.sweepIdle(ctx); err != nil {
		t.Fatal(err)
	}
	want := "18->6:200.0000" // AcctTrading→AcctFunding 超部=700−500
	for _, tr := range fake.transfers {
		if tr == want {
			return
		}
	}
	t.Fatalf("超部扫额错——want %q got %v", want, fake.transfers)
}

// TestCollateralTopup ⑤avail=100<floor=500→泊资 RedeemToTrading 摆回
// 短额 400（+buffer 50bps=402）——earn_redeem decision 行+funding→trading
// 划转双留痕（sweep 反向=redeem 既有臂复用实证）。
func TestCollateralTopup(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":            "100",
		"margin.exec_enabled":           "true",
		"margin.collateral_reserve_usd": "500",
		"earn.redeem_buffer_bps":        "50",
	})
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachIdleUSDT(w, 100)
	w.collateralReserveTopup(ctx)
	if len(fake.redeems) != 1 {
		t.Fatalf("泊资赎回应恰一次 redeems=%v", fake.redeems)
	}
	// 短额=500−100=400，buffer 50bps→402.0
	if fake.redeems[0] < 401.9 || fake.redeems[0] > 402.1 {
		t.Fatalf("泊资额=%.4f want≈402（短额400+buffer）", fake.redeems[0])
	}
	var sawF2T bool
	for _, tr := range fake.transfers {
		if strings.HasPrefix(tr, "6->18:") {
			sawF2T = true
		}
	}
	if !sawF2T {
		t.Fatalf("泊资划转应 funding→trading transfers=%v", fake.transfers)
	}
}

// TestCollateralTopupCooldown ⑥冷却窗内不重复发起——读面滞后防锤 venue。
func TestCollateralTopupCooldown(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":            "100",
		"margin.exec_enabled":           "true",
		"margin.collateral_reserve_usd": "500",
		"earn.redeem_buffer_bps":        "50",
	})
	defer w.pool.Exec(ctx, "DELETE FROM fin_risk_decision WHERE intent->>'kind'='transfer'")
	attachIdleUSDT(w, 100)
	w.collateralReserveTopup(ctx)
	w.collateralReserveTopup(ctx) // 冷却窗内第二发——应被闸
	if len(fake.redeems) != 1 {
		t.Fatalf("冷却窗内应恰一次赎回 redeems=%v", fake.redeems)
	}
	if w.rej.Snapshot()["earn/reserve_topup_cooldown"] == 0 {
		t.Fatal("reserve_topup_cooldown 计数门未计")
	}
}

// TestCollateralTopupNoShortfall ⑦avail=600≥floor=500→无短额不动作。
func TestCollateralTopupNoShortfall(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":            "100",
		"margin.exec_enabled":           "true",
		"margin.collateral_reserve_usd": "500",
		"earn.redeem_buffer_bps":        "50",
	})
	attachIdleUSDT(w, 600)
	w.collateralReserveTopup(ctx)
	if len(fake.redeems) != 0 {
		t.Fatalf("无短额不应赎回 redeems=%v", fake.redeems)
	}
}

// TestCollateralTopupExecOff ⑧exec=false→泊资判据整体失效（休眠不留钱）。
func TestCollateralTopupExecOff(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, map[string]string{
		"earn.sweep_min_usd":            "100",
		"margin.exec_enabled":           "false",
		"margin.collateral_reserve_usd": "500",
		"earn.redeem_buffer_bps":        "50",
	})
	attachIdleUSDT(w, 0)
	w.collateralReserveTopup(ctx)
	if len(fake.redeems) != 0 {
		t.Fatalf("exec=false 不应泊资 redeems=%v", fake.redeems)
	}
}

// TestCollateralReserveState 三态直读锚：exec 关→(0,false)；键缺→hold；
// reserve≤0→(0,false)；正常→(floor,false)。
func TestCollateralReserveState(t *testing.T) {
	ctx, w, _ := setupW(t, &fakeAPI{}, map[string]string{
		"margin.exec_enabled":           "true",
		"margin.collateral_reserve_usd": "500",
	})
	floor, hold := w.reserveState()
	if floor != 500 || hold {
		t.Fatalf("reserveState=(%v,%v) want (500,false)", floor, hold)
	}
	// reserve≤0=显式关保底（不豁免不截量）
	if _, err := w.pool.Exec(ctx,
		"UPDATE app_config SET value='0' WHERE key='margin.collateral_reserve_usd'"); err != nil {
		t.Fatal(err)
	}
	cfg2 := config.New(w.pool)
	if err := cfg2.Load(ctx); err != nil {
		t.Fatal(err)
	}
	w.cfg = cfg2
	floor, hold = w.reserveState()
	if floor != 0 || hold {
		t.Fatalf("reserve=0 reserveState=(%v,%v) want (0,false)", floor, hold)
	}
}
