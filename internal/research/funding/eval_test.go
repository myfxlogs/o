package funding

import "testing"

var p = Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4, MinNetAPR: 0.05, SettlesPerDay: 3}

func TestFiveFold(t *testing.T) {
	// rate=0.001 → gross=0.001*3*365=1.095；cost=(10+3+2+4)bps=0.0019 → net=1.0931
	e := Evaluate("BTC-USDT-SWAP", 0.001, p)
	if e.GrossAPR != 0.001*3*365 {
		t.Fatalf("gross=%v", e.GrossAPR)
	}
	if e.Verdict != "proposed" || e.Direction != "short_perp_long_spot" {
		t.Fatalf("eval: %+v", e)
	}
}

func TestBelowThreshold(t *testing.T) {
	e := Evaluate("X-USDT-SWAP", 0.00001, p)
	if e.Verdict != "rejected" || e.NetAPR >= p.MinNetAPR {
		t.Fatalf("低率应闸下: %+v", e)
	}
}

func TestNegativeFunding(t *testing.T) {
	// 负 funding→long_perp_short_spot——现货做空无支撑=rejected（不可执行非不可评估）
	e := Evaluate("X-USDT-SWAP", -0.002, p)
	if e.Direction != "long_perp_short_spot" || e.Verdict != "rejected" {
		t.Fatalf("反向处置错: %+v", e)
	}
}
