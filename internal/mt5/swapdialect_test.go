package mt5

// swapdialect_test.go —— swap 点数方言换算桩测（T-0143 验收锚）：Points 径
// 数值形/PercCurPrice 径/换算链缺环 NULL/非 Points NULL。
import (
	"context"
	"runtime"
	"testing"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"

	"o/internal/platform/config"
	"o/internal/platform/rejects"
	"o/internal/platform/testutil"
)

func sgOf(t mt5pb.SwapType, short float64) *mt5pb.SymGroup {
	return &mt5pb.SymGroup{SwapType: t, SwapShort: short, SwapLong: -1786.5}
}

func siOf(tv, ts, pts, cs float64) *mt5pb.SymbolInfo {
	return &mt5pb.SymbolInfo{TickValue: tv, TickSize: ts, Points: pts, ContractSize: cs}
}

// 锚：PercCurPrice 直年化（swap_short=-0.05%/天→-0.05×365×10⁴=-182500 bps）。
func TestSwapAnnualPercCurPrice(t *testing.T) {
	ann, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_PercCurPrice, -0.05), 3000)
	if !ok || ann != -182500 {
		t.Fatalf("perc 径 ann=%v ok=%v", ann, ok)
	}
}

// 锚：Points 径（二返修正典式——真实 venue 形：tick 双 0 不填，points/cs 实填）。
// ETHUSDm 实拍形：points=0.01/cs=1/tick_value=tick_size=0/swap_short=0→空腿免费 ann=0。
func TestSwapAnnualPointsVenueShape(t *testing.T) {
	ann, ok := SwapAnnualBPS(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, 0), 3000)
	if !ok || ann != 0 {
		t.Fatalf("空腿免费形 ann=%v ok=%v（0 值合法产出非缺席）", ann, ok)
	}
	// swap_long=-1786.5 点/天（实拍 raw）代入 mid=79842 →
	// -1786.5×0.01×1×365/(79842×1)×10⁴≈-816.8 bps/yr 量级断。
	ann, ok = SwapAnnualBPS(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -1786.5), 79842)
	if !ok {
		t.Fatal("tick 双 0 不应阻断（候选佐证非必需环）")
	}
	want := -1786.5 * 0.01 * 1 * 365 / 79842 * 10000
	if ann < want*1.01 || ann > want*0.99 {
		t.Fatalf("points 正典式 ann=%v want≈%v", ann, want)
	}
}

// 锚：换算链缺环（tick_value=0）→false（val=NULL 缺席如实）；非 Points/Perc
// 方言→false；mid 缺（0）→false；SymbolInfo 缺→false。
func TestSwapAnnualMissingLinks(t *testing.T) {
	if _, ok := SwapAnnualBPS(siOf(0, 0, 0, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000); ok {
		t.Fatal("points 缺应 false（真缺环）")
	}
	if _, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 0); ok {
		t.Fatal("mid 缺应 false")
	}
	if _, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_MarginCurrency, -50), 3000); ok {
		t.Fatal("非 Points/Perc 方言应 false")
	}
	if _, ok := SwapAnnualBPS(nil, sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000); ok {
		t.Fatal("SymbolInfo 缺应 false")
	}
}

// 返修锚（T-0143）：swapLoop 随会话 ctx 止——sessCtx cancel→goroutine 退出
// （断连桩锚：bridgeSession defer cancel 同源语义，goroutine 零泄漏）。
func TestSwapLoopStopsWithSession(t *testing.T) {
	pool := testutil.Pool(t)
	c := &BridgeCollector{pool: pool, cfg: config.New(pool),
		rej: rejects.New("mt5bridge", "test")}
	sessCtx, cancel := context.WithCancel(context.Background())
	before := runtime.NumGoroutine()
	done := make(chan struct{})
	go func() {
		c.swapLoop(sessCtx, nil) // cl=nil——会话 ctx cancel 即退（poll 零 sym 不触 RPC）
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second): // sched-exempt-ok 桩测期界等待
		t.Fatal("swapLoop 应随会话 ctx 止（零泄漏）")
	}
	if after := runtime.NumGoroutine(); after > before+2 { // 容忍 runtime 波动
		t.Fatalf("goroutine 应归位 before=%d after=%d", before, after)
	}
}

// T-0144 返修①锚：双腿年化——swap_long=-1786.5pts（实拍 raw）代入
// points=0.01/cs=1/mid=79842 → long_ann≈-816bps；short=0→short_ann=0；
// 双端齐 ok。
func TestSwapAnnualBoth(t *testing.T) {
	long, short, ok := SwapAnnualBoth(siOf(0, 0, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, 0), 79842)
	if !ok {
		t.Fatal("双腿应双端齐")
	}
	want := -1786.5 * 0.01 * 365 / 79842 * 10000
	if long < want*1.01 || long > want*0.99 {
		t.Fatalf("long_ann=%v want≈%v", long, want)
	}
	if short != 0 {
		t.Fatalf("short_ann=%v（空腿零成本实拍）", short)
	}
}
