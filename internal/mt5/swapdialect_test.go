package mt5

// swapdialect_test.go —— swap 点数方言换算桩测（T-0143 验收锚）：Points 径
// 数值形/PercCurPrice 径/换算链缺环 NULL/非 Points NULL。
import (
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"
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

// 锚：Points 径——swap_short=-50 点/天，tv=1USDT/tick，ts=0.01，pts=0.01，
// cs=1，mid=3000 → 币值/手/天=-50×1×0.01/0.01=-50 → 年化=-50×365/3000×10⁴。
func TestSwapAnnualPoints(t *testing.T) {
	ann, ok := SwapAnnualBPS(siOf(1, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000)
	if !ok {
		t.Fatal("points 径应可年化")
	}
	want := -50 * 1 * 0.01 / 0.01 * 365 / 3000 * 10000
	if ann < want*1.001 || ann > want*0.999 {
		t.Fatalf("points 径 ann=%v want≈%v", ann, want)
	}
}

// 锚：换算链缺环（tick_value=0）→false（val=NULL 缺席如实）；非 Points/Perc
// 方言→false；mid 缺（0）→false；SymbolInfo 缺→false。
func TestSwapAnnualMissingLinks(t *testing.T) {
	if _, ok := SwapAnnualBPS(siOf(0, 0.01, 0.01, 1), sgOf(mt5pb.SwapType_SwapType_InPoints, -50), 3000); ok {
		t.Fatal("tick_value 缺应 false")
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
