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

// evalParams 三态判据（净 APR 0.06 落静态底 0.05 与 Earn 底 0.08 之间——between 锚定形）。
func evalParams(earn *float64) Params {
	return Params{
		TakerFeeBps: 1500, SlippageBps: 50, FailCostBps: 0, WindowLossBps: 0,
		MinNetAPR: 0.05, SettlesPerDay: 1, EarnRateAPR: earn,
	}
}

// TestEvaluateDynamicFloor 动态比较底三态（D-0006）：
// ①net 在静态底与 Earn 底之间→below_earn 拒 ②net 高于 Earn 底→过 ③Earn 缺失→静态底降级。
func TestEvaluateDynamicFloor(t *testing.T) {
	// ①between：net=0.06（0.365 毛收−3050bps）∈[0.05, 0.08)→below_earn 拒
	earn := 0.08
	e := Evaluate("TEST-USDT-SWAP", 0.001, evalParams(&earn))
	if e.Verdict != "rejected" || e.Reason[:10] != "below_earn" {
		t.Fatalf("between 态 verdict=%s reason=%s（应 below_earn 拒）", e.Verdict, e.Reason)
	}
	if e.Evidence["floor_source"] != "earn" || e.Evidence["floor_apr"] != 0.08 {
		t.Fatalf("floor 证据=%v（应 earn/0.08）", e.Evidence)
	}
	// ②高于 Earn 底：net=10.948 ≥ 0.08→proposed
	e = Evaluate("TEST-USDT-SWAP", 0.01, evalParams(&earn))
	if e.Verdict != "proposed" {
		t.Fatalf("高于 Earn 底 verdict=%s（应 proposed）", e.Verdict)
	}
	// ③Earn 缺失：floor=静态底 0.05，net=0.06→proposed（降级不拒）
	e = Evaluate("TEST-USDT-SWAP", 0.001, evalParams(nil))
	if e.Verdict != "proposed" {
		t.Fatalf("降级态 verdict=%s（应静态底放行）", e.Verdict)
	}
	if e.Evidence["floor_source"] != "static" {
		t.Fatalf("降级 floor 证据=%v", e.Evidence)
	}
	// ④Earn 底低于静态底：floor 取 max=静态底——below_earn 不误杀（net<0.05 走静态拒因）
	low := 0.01
	e = Evaluate("TEST-USDT-SWAP", 0.0005, evalParams(&low)) // net=0.1825-0.305<0→静态拒
	if e.Verdict != "rejected" || e.Reason[:10] == "below_earn" {
		t.Fatalf("低 Earn 底 verdict=%s reason=%s（应静态拒因）", e.Verdict, e.Reason)
	}
}

// T-0247 钦定锚①：存在性与取值分层四分支——0=复合非双 taker / 负=返佣
// 入 cost 负项 / 缺=双 taker 回退 / 正=复合回归。
func TestMakerFeeSetSplit(t *testing.T) {
	base := Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4, MinNetAPR: 0.05, SettlesPerDay: 3}
	mk := func(set bool, maker float64) Params { p := base; p.MakerFeeSet, p.MakerFeeBps = set, maker; return p }
	cases := []struct {
		name           string
		set            bool
		maker, wantFee float64
	}{
		{"缺键回退双 taker", false, 0, 10},
		{"置 0 复合（免 maker 费）", true, 0, 5},
		{"负值返佣入负项", true, -2, 3},
		{"正值复合回归", true, 1, 6},
	}
	for _, c := range cases {
		p := mk(c.set, c.maker)
		e := Evaluate("T-USDT-SWAP", 0.0003, p)
		fee := 2 * p.TakerFeeBps
		if p.MakerFeeSet {
			fee = p.TakerFeeBps + p.MakerFeeBps
		}
		if fee != c.wantFee {
			t.Fatalf("%s: feeBps=%v want %v", c.name, fee, c.wantFee)
		}
		if ev, ok := e.Evidence["maker_fee_set"].(bool); !ok || ev != c.set {
			t.Fatalf("%s: evidence maker_fee_set=%v", c.name, e.Evidence["maker_fee_set"])
		}
	}
}
