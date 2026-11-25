package funding

import "testing"

// O-39460 margin-short 评估锚：借币息入净硬条款+可借性证据面。
func TestEvaluateMarginShort(t *testing.T) {
	p := Params{TakerFeeBps: 5, MakerFeeBps: 2, MakerFeeSet: true,
		SlippageBps: 5, FailCostBps: 10, WindowLossBps: 5,
		MinNetAPR: 0.05, SettlesPerDay: 3, HoldDays: 30}
	// rate=-0.001(−0.1%/8h)→gross=0.001*3*365=1.095；borrow 7.3%/yr+cost 摊销
	b := BorrowQuote{APR: 0.073, MaxLoan: 100, LoanUSD: 50000}
	e := EvaluateMarginShort("TEST-USDT-SWAP", -0.001, p, b)
	if e.Direction != "long_perp_short_spot" {
		t.Fatalf("direction=%s", e.Direction)
	}
	if e.Legs["spot"] != "short_margin" || e.Legs["perp"] != "long" {
		t.Fatalf("legs=%v", e.Legs)
	}
	wantNet := 1.095 - 0.073 - CostAPR(p)
	if diff := e.NetAPR - wantNet; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("net_apr=%.6f want=%.6f（借币息未入净=假账）", e.NetAPR, wantNet)
	}
	if e.Evidence["borrow_cost_in_net"] != true {
		t.Fatal("borrow_cost_in_net 钉证缺")
	}
	if e.Evidence["borrow_apr"] != 0.073 {
		t.Fatalf("borrow_apr evidence=%v", e.Evidence["borrow_apr"])
	}
	if e.Verdict != "proposed" {
		t.Fatalf("verdict=%s reason=%s（净额应过 Earn/静态底）", e.Verdict, e.Reason)
	}
	// 息吞案：借币息 150%/yr→net 转负=拒（息差致死实证）
	b2 := BorrowQuote{APR: 1.5, MaxLoan: 100, LoanUSD: 50000}
	e2 := EvaluateMarginShort("TEST-USDT-SWAP", -0.001, p, b2)
	if e2.Verdict != "rejected" {
		t.Fatalf("息吞案 verdict=%s net=%.4f", e2.Verdict, e2.NetAPR)
	}
	// 微负 funding+高息=拒（|rate| 小不抵息）
	e3 := EvaluateMarginShort("TEST-USDT-SWAP", -0.0001, p, b)
	if e3.Verdict != "rejected" {
		t.Fatalf("微负息吞 verdict=%s net=%.4f", e3.Verdict, e3.NetAPR)
	}
}
