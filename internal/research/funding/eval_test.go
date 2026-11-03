package funding

import (
	"math"
	"testing"
)

var p = Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4, MinNetAPR: 0.05, SettlesPerDay: 3, HoldDays: 3}

func TestFiveFold(t *testing.T) {
	// rate=0.001 → gross=0.001*3*365=1.095；cost_rt=32bps→cost_apr=0.0032*365/3≈0.3893 → net≈0.7057
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

// evalParams 三态判据（净 APR 0.06 落静态底 0.05 与 Earn 底 0.08 之间——between 锚定形；
// T-0297 新口径：cost_rt=3050bps 按 HoldDays=365 摊→cost_apr=0.305，net=0.365−0.305=0.06 同锚）。
func evalParams(earn *float64) Params {
	return Params{
		TakerFeeBps: 750, SlippageBps: 25, FailCostBps: 0, WindowLossBps: 0,
		MinNetAPR: 0.05, SettlesPerDay: 1, HoldDays: 365, EarnRateAPR: earn,
	}
}

// TestEvaluateDynamicFloor 动态比较底三态（D-0006）：
// ①net 在静态底与 Earn 底之间→below_earn 拒 ②net 高于 Earn 底→过 ③Earn 缺失→静态底降级。
func TestEvaluateDynamicFloor(t *testing.T) {
	// ①between：net=0.06（0.365 毛收−3050bps 往返×365/365 摊=0.305）∈[0.05, 0.08)→below_earn 拒
	earn := 0.08
	e := Evaluate("TEST-USDT-SWAP", 0.001, evalParams(&earn))
	if e.Verdict != "rejected" || e.Reason[:10] != "below_earn" {
		t.Fatalf("between 态 verdict=%s reason=%s（应 below_earn 拒）", e.Verdict, e.Reason)
	}
	if e.Evidence["floor_source"] != "earn" || e.Evidence["floor_apr"] != 0.08 {
		t.Fatalf("floor 证据=%v（应 earn/0.08）", e.Evidence)
	}
	// ②高于 Earn 底：net=3.345 ≥ 0.08→proposed
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
	base := Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4, MinNetAPR: 0.05, SettlesPerDay: 3, HoldDays: 3}
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
		got, ok := e.Evidence["fee_bps"].(float64) // 断真产出非本地重算（返修①）
		if !ok || got != c.wantFee {
			t.Fatalf("%s: evidence fee_bps=%v want %v", c.name, got, c.wantFee)
		}
		wantNet := 0.0003*p.SettlesPerDay*365 - (2*c.wantFee+2*p.SlippageBps+p.FailCostBps+p.WindowLossBps)/1e4*365/p.HoldDays
		if e.NetAPR != wantNet {
			t.Fatalf("%s: NetAPR=%v want %v", c.name, e.NetAPR, wantNet)
		}
		if ev, ok := e.Evidence["maker_fee_set"].(bool); !ok || ev != c.set {
			t.Fatalf("%s: evidence maker_fee_set=%v", c.name, e.Evidence["maker_fee_set"])
		}
	}
}

// T-0297 验收锚：往返口径+年化摊销——HoldDays 1/3/30 档 NetAPR 单调递增
// 且与手算相等（cost_rt=32bps：1d→cost_apr=1.168/3d→0.3893/30d→0.0389）；
// 薄率被拒（reason 含 net_apr）；evidence 带 cost_rt_bps/hold_days/cost_apr。
func TestRoundTripAmortization(t *testing.T) {
	base := Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4,
		MinNetAPR: 0.05, SettlesPerDay: 3}
	if got := RoundTripBps(base); got != 32 {
		t.Fatalf("RoundTripBps=%v want 32（2×10+2×3+2+4）", got)
	}
	var prev float64
	for i, hd := range []float64{1, 3, 30} {
		pp := base
		pp.HoldDays = hd
		e := Evaluate("T-USDT-SWAP", 0.0003, pp)
		wantCost := 0.0032 * 365 / hd // 手算锚（1d=1.168/3d≈0.3893/30d≈0.0389）
		if got := e.Evidence["cost_apr"].(float64); math.Abs(got-wantCost) > 1e-9 {
			t.Fatalf("hd=%v cost_apr=%v want %v", hd, got, wantCost)
		}
		if got := e.Evidence["cost_rt_bps"].(float64); got != 32 {
			t.Fatalf("hd=%v cost_rt_bps=%v want 32", hd, got)
		}
		if got := e.Evidence["hold_days"].(float64); got != hd {
			t.Fatalf("hd=%v evidence hold_days=%v", hd, got)
		}
		wantNet := 0.0003*3*365 - wantCost
		if math.Abs(e.NetAPR-wantNet) > 1e-9 {
			t.Fatalf("hd=%v NetAPR=%v want %v", hd, e.NetAPR, wantNet)
		}
		if i > 0 && e.NetAPR <= prev {
			t.Fatalf("hd=%v NetAPR 非单调递增: %v<=%v", hd, e.NetAPR, prev)
		}
		prev = e.NetAPR
	}
	// 薄率（≈11% 毛收）按 3d 摊→rejected 且 reason 含 net_apr
	pp := base
	pp.HoldDays = 3
	e := Evaluate("T-USDT-SWAP", 0.0001, pp)
	if e.Verdict != "rejected" || len(e.Reason) < 7 || e.Reason[:7] != "net_apr" {
		t.Fatalf("薄率 verdict=%s reason=%q（应 net_apr 拒）", e.Verdict, e.Reason)
	}
}
