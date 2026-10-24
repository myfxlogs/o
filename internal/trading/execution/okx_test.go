package execution

import "testing"

// T-0040 锚：posMode 实读注入——long_short 开仓腿必带 posSide、net/cash 免。
func TestBuildOrderParamsPosSide(t *testing.T) {
	swapLeg := func(close bool, side string) Leg {
		return Leg{InstID: "BTC-USDT-SWAP", Side: side, Sz: "1", Close: close}
	}
	for _, tc := range []struct {
		name, posMode, side string
		close               bool
		wantPosSide         string
		wantReduceOnly      bool
	}{
		{"双向户开仓 sell→short", "long_short_mode", "sell", false, "short", false},
		{"双向户开仓 buy→long", "long_short_mode", "buy", false, "long", false},
		{"双向户平仓 buy→平 short 仓", "long_short_mode", "buy", true, "short", true},
		{"双向户平仓 sell→平 long 仓", "long_short_mode", "sell", true, "long", true},
		{"net 户免 posSide", "net_mode", "sell", false, "", false},
		{"net 户平仓 reduceOnly 仍置", "net_mode", "buy", true, "", true},
	} {
		p := buildOrderParams(swapLeg(tc.close, tc.side), "ph1x1x1", tc.posMode)
		if p.PosSide != tc.wantPosSide || p.ReduceOnly != tc.wantReduceOnly {
			t.Fatalf("%s: posSide=%q reduceOnly=%v want %q/%v", tc.name, p.PosSide, p.ReduceOnly, tc.wantPosSide, tc.wantReduceOnly)
		}
		if p.TdMode != "cross" {
			t.Fatalf("%s: SWAP 应 cross: %q", tc.name, p.TdMode)
		}
	}
	// spot（tdMode=cash）腿——双向户亦免带（cash 无持仓向语义）。
	for _, side := range []string{"buy", "sell"} {
		p := buildOrderParams(Leg{InstID: "BTC-USDT", Side: side, Sz: "0.01"}, "ph1x1x1", "long_short_mode")
		if p.PosSide != "" || p.TdMode != "cash" {
			t.Fatalf("spot %s 免 posSide: got posSide=%q tdMode=%q", side, p.PosSide, p.TdMode)
		}
	}
}
