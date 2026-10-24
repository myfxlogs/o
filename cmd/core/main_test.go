package main

import (
	"testing"

	"phase/internal/guard"
)

// T-0043 锚①：Warden 平仓径双腿必置 Close=true（漏置=按开仓腿装配→
// short 仓上 buy 反向开 long 非平仓——阻断级雷）。
func TestCloseLegOfSetsClose(t *testing.T) {
	spec := guard.CloseSpec{NotionalUSD: 100, Legs: [2]guard.CloseLeg{
		{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
		{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.0012},
	}}
	for i, l := range spec.Legs {
		got := closeLegOf(spec, l)
		if !got.Close {
			t.Fatalf("leg%d Close 漏置（反向开仓雷）: %+v", i+1, got)
		}
		if got.Venue != l.Venue || got.InstID != l.InstID || got.Side != l.Side || got.Notional != 50 {
			t.Fatalf("leg%d 装配串味: %+v", i+1, got)
		}
	}
}

// T-0043 欠件锚（T-0046 补）：closeIntent Kind 必为 "close"（live_switch 翻 false
// 时平仓被拦=安全律反向——Kind 错=平仓被开仓闸吞）。
func TestCloseIntentKindIsClose(t *testing.T) {
	in := closeIntent(guard.CloseSpec{DecisionID: 1, StrategyRef: 1, InstID: "BTC-USDT-SWAP", NotionalUSD: 100,
		Legs: [2]guard.CloseLeg{{Venue: "okx", InstID: "BTC-USDT-SWAP", Side: "buy", Sz: 1},
			{Venue: "okx", InstID: "BTC-USDT", Side: "sell", Sz: 0.001}}})
	if in.Kind != "close" {
		t.Fatalf("close Intent Kind=%q（应 close——免闸免赎语义位）", in.Kind)
	}
}
