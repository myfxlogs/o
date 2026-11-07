package execwiring

import (
	"testing"

	"o/internal/guard"
)

// T-0043 锚（随装配缝单源 T-0426 迁驻）：CloseIntent Kind=="close"+双腿
// Close=true（免闸免赎语义位——live_switch 翻 false 时平仓被拦=安全律反向）。
func TestCloseIntentKind(t *testing.T) {
	in := CloseIntent(guard.CloseSpec{
		DecisionID: 7, StrategyRef: 1, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]guard.CloseLeg{
			{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 5},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.0005},
		}})
	if in.Kind != "close" {
		t.Fatalf("Kind=%q（应 close——漏置=平仓被 live_switch 拦）", in.Kind)
	}
	for i, l := range in.Legs {
		if !l.Close {
			t.Fatalf("leg%d Close=false（平仓语义位漏置）", i)
		}
	}
}

// T-0043 锚①（随装配缝单源迁驻）：平仓径双腿必置 Close=true（漏置=按开仓腿
// 装配→short 仓上 buy 反向开 long 非平仓——阻断级雷）。
func TestCloseLegOfSetsClose(t *testing.T) {
	spec := guard.CloseSpec{NotionalUSD: 100, Legs: [2]guard.CloseLeg{
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
		{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.0012},
	}}
	for i, l := range spec.Legs {
		got := CloseLegOf(spec, l)
		if !got.Close {
			t.Fatalf("leg%d Close 漏置（反向开仓雷）: %+v", i+1, got)
		}
		if got.Venue != l.Venue || got.InstID != l.InstID || got.Side != l.Side || got.Notional != 50 {
			t.Fatalf("leg%d 装配串味: %+v", i+1, got)
		}
	}
}
