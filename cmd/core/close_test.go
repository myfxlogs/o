package main

import (
	"testing"

	"phase/internal/guard"
)

// T-0043 锚：closeExec Intent.Kind=="close"+双腿 Close=true（免闸免赎语义位——
// live_switch 翻 false 时平仓被拦=安全律反向）。
func TestCloseIntentKind(t *testing.T) {
	in := closeIntent(guard.CloseSpec{
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
