package simvenue

import (
	"context"
	"math"
	"testing"

	"o/internal/trading/execution"
)

// fixture CORE 案乐器（T-0419 仓#33 形——数值=fixture 非产线常量，语义锚：
// quote 买 25 USDT@0.0125 费 0.001→实得 1998 CORE；perp 19.98 张×ctVal 100×px=
// 同名义 $24.975 均衡）。
func corePair() (Instrument, Instrument) {
	spot := Instrument{InstID: "CORE-USDT", InstType: "SPOT", LotSz: 0.01, MinSz: 0.01,
		TickSz: 0.0001, TakerFee: 0.001, Px: 0.0125}
	swap := Instrument{InstID: "CORE-USDT-SWAP", InstType: "SWAP", LotSz: 0.01, MinSz: 0.01,
		TickSz: 0.0001, CtVal: 100, CtValCcy: "CORE", TakerFee: 0.0005, Px: 0.0125}
	return spot, swap
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

// 拒单槽序（钦定冻结——业主令：lotSz/minSz→价格带→not_marketable→
// insufficient_balance，首命中即拒，顺序写死）。
func TestRejectSlotOrderFrozen(t *testing.T) {
	ctx := context.Background()
	spot, _ := corePair()
	mk := func(sz string, mod func(*execution.Leg)) execution.Leg {
		l := execution.Leg{Venue: "sim", InstID: "CORE-USDT", Side: "buy", Sz: sz}
		if mod != nil {
			mod(&l)
		}
		return l
	}
	t.Run("槽①网格：低于minSz/不落网格/超max", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 1e6)
		res, err := v.Place(ctx, mk("0.005", nil), "c1") // < minSz 0.01
		if err != nil || res.VenueCode != RejBelowMin || res.State != "absent" {
			t.Fatalf("低于 minSz 应 51006 absent: %+v err=%v", res, err)
		}
		res, _ = v.Place(ctx, mk("0.015", nil), "c2") // 不落 0.01 网格
		if res.VenueCode != RejOffGrid {
			t.Fatalf("不落网格应 51005: %+v", res)
		}
		v2 := New().WithInstrument(Instrument{InstID: "X-USDT", InstType: "SPOT",
			LotSz: 1, MinSz: 1, MaxSz: 10, Px: 1}).WithBalance("USDT", 1e6)
		res, _ = v2.Place(ctx, execution.Leg{Venue: "sim", InstID: "X-USDT", Side: "buy", Sz: "11"}, "c3")
		if res.VenueCode != RejAboveMax {
			t.Fatalf("超 max 应 51007（量上限）: %+v", res)
		}
	})
	t.Run("槽序首命中：网格违+余额不足并存→网格先拒", func(t *testing.T) {
		v := New().WithInstrument(spot) // 零余额
		res, _ := v.Place(ctx, mk("0.015", nil), "c4")
		if res.VenueCode != RejOffGrid {
			t.Fatalf("槽①先于槽④: %+v", res)
		}
	})
	t.Run("槽②价格带：挂价出带→51020；带内过", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 1e6)
		res, _ := v.Place(ctx, mk("100", func(l *execution.Leg) {
			l.Maker, l.LimitPx = true, spot.Px*2 // +100%≫±10% 带
		}), "c5")
		if res.VenueCode != RejPxBand {
			t.Fatalf("出价格带应 51020: %+v", res)
		}
		res, _ = v.Place(ctx, mk("100", func(l *execution.Leg) {
			l.Maker, l.LimitPx = true, spot.Px*1.05
		}), "c6")
		if res.State != "filled" {
			t.Fatalf("带内挂价应成交: %+v", res)
		}
	})
	t.Run("槽②先于槽④：挂价出带+零余额→出带先拒", func(t *testing.T) {
		v := New().WithInstrument(spot) // 零余额
		res, _ := v.Place(ctx, mk("100", func(l *execution.Leg) {
			l.Maker, l.LimitPx = true, spot.Px*2
		}), "c7")
		if res.VenueCode != RejPxBand {
			t.Fatalf("槽②先于槽④: %+v", res)
		}
	})
	t.Run("槽③not_marketable：不在册/state≠live→51001", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 1e6)
		res, _ := v.Place(ctx, execution.Leg{Venue: "sim", InstID: "GHOST-USDT", Side: "buy", Sz: "100"}, "c8")
		if res.VenueCode != RejNotMkt {
			t.Fatalf("不在册应 51001: %+v", res)
		}
		susp := spot
		susp.State = "suspend"
		v2 := New().WithInstrument(susp).WithBalance("USDT", 1e6)
		res, _ = v2.Place(ctx, mk("100", nil), "c9")
		if res.VenueCode != RejNotMkt || res.State != "absent" {
			t.Fatalf("suspend 应 51001 absent: %+v", res)
		}
	})
	t.Run("槽④insufficient_balance：现货买/卖+合约减仓超持", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 1)
		res, _ := v.Place(ctx, mk("100", nil), "c10")
		if res.VenueCode != RejInsuffBal {
			t.Fatalf("余额不足应 51008: %+v", res)
		}
		_, swap := corePair()
		v2 := New().WithInstrument(swap).WithInstrument(spot).WithBalance("USDT", 1e6)
		v2.pos["CORE-USDT-SWAP"] = -1 // 净空 1 张
		res, _ = v2.Place(ctx, execution.Leg{Venue: "sim", InstID: "CORE-USDT-SWAP",
			Side: "buy", Sz: "5", Close: true}, "c11")
		if res.VenueCode != RejInsuffBal {
			t.Fatalf("减仓超净持应 51008（反向新开防）: %+v", res)
		}
	})
	t.Run("拒单一等事件：VenueCode/VenueMsg 携带禁吞+版本指纹", func(t *testing.T) {
		v := New().WithInstrument(spot)
		res, err := v.Place(ctx, mk("0.005", nil), "c12")
		if err != nil || res.VenueMsg == "" || res.RulesVersion != RulesVersion {
			t.Fatalf("拒单回执须携码+msg+指纹: %+v err=%v", res, err)
		}
	})
}

// 量纲语义（flow.legPx 同款——quote 买实得 base 币量=T-0116 成交额锚）。
func TestQuoteUnitFillMath(t *testing.T) {
	ctx := context.Background()
	spot, swap := corePair()
	v := New().WithInstrument(spot).WithInstrument(swap).WithBalance("USDT", 100)
	res, err := v.Place(ctx, execution.Leg{Venue: "sim", InstID: "CORE-USDT",
		Side: "buy", Sz: "25", TgtQuote: true, Notional: 25}, "q1")
	if err != nil || res.State != "filled" {
		t.Fatalf("quote 买应成交: %+v err=%v", res, err)
	}
	// 实得=25×(1−0.001)/0.0125=1998；实扣=成交额 1998×0.0125=24.975<委托额（T-0116）。
	if !approx(res.FillSz, 1998.0) || !approx(res.FillPx, 0.0125) {
		t.Fatalf("quote 买实得币量: got %.6f want 1998", res.FillSz)
	}
	if !approx(v.Balance("CORE"), 1998.0) || !approx(v.Balance("USDT"), 100-24.975) {
		t.Fatalf("余额真相: CORE=%.6f USDT=%.6f", v.Balance("CORE"), v.Balance("USDT"))
	}
	f := v.Fills()
	if len(f) != 1 || !approx(f[0].QuoteAmt, -24.975) {
		t.Fatalf("流水现金流: %+v", f)
	}
	// 卖（平仓）——费计价币内扣：实收=毛额×(1−费)。
	res, err = v.Place(ctx, execution.Leg{Venue: "sim", InstID: "CORE-USDT",
		Side: "sell", Sz: "1998", Close: true, Notional: 25}, "q2")
	if err != nil || res.State != "filled" || res.FillSz != 1998 {
		t.Fatalf("卖应满成: %+v err=%v", res, err)
	}
	if !approx(v.Balance("CORE"), 0) {
		t.Fatalf("卖后 CORE 应清零: %.8f", v.Balance("CORE"))
	}
	if !approx(v.Balance("USDT"), 100-24.975+24.975*0.999) {
		t.Fatalf("卖后 USDT 实收毛×(1−费): %.8f", v.Balance("USDT"))
	}
}

// 合约量纲：张数成交+净持仓带号+减仓归零。
func TestContractUnitFill(t *testing.T) {
	ctx := context.Background()
	_, swap := corePair()
	v := New().WithInstrument(swap).WithBalance("USDT", 1e6)
	res, err := v.Place(ctx, execution.Leg{Venue: "sim", InstID: "CORE-USDT-SWAP",
		Side: "sell", Sz: "19.98", Notional: 25}, "s1")
	if err != nil || res.State != "filled" || res.FillSz != 19.98 {
		t.Fatalf("SWAP 卖应满成张数: %+v err=%v", res, err)
	}
	if !approx(v.Position("CORE-USDT-SWAP"), -19.98) {
		t.Fatalf("净持仓应 -19.98: %v", v.Position("CORE-USDT-SWAP"))
	}
	if _, err := v.Place(ctx, execution.Leg{Venue: "sim", InstID: "CORE-USDT-SWAP",
		Side: "buy", Sz: "19.98", Close: true, Notional: 25}, "s2"); err != nil {
		t.Fatal(err)
	}
	if !approx(v.Position("CORE-USDT-SWAP"), 0) {
		t.Fatalf("减仓后净持应归零: %v", v.Position("CORE-USDT-SWAP"))
	}
}

// 订单生命周期态机：部分成交→Resolve 补成→filled；撤挂；ResolveCap 耗尽补完。
func TestOrderLifecycleStateMachine(t *testing.T) {
	ctx := context.Background()
	spot, _ := corePair()
	leg := execution.Leg{Venue: "sim", InstID: "CORE-USDT", Side: "buy", Sz: "25", TgtQuote: true}
	t.Run("部分成→补成→filled（增量落账）", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 100).
			WithPlan("CORE-USDT", Plan{FillRatio: 0.5, ResolveCap: 1})
		res, err := v.Place(ctx, leg, "p1")
		if err != nil || res.State != "partially_filled" || !approx(res.FillSz, 999) {
			t.Fatalf("首拍应半成: %+v err=%v", res, err)
		}
		if !approx(v.Balance("CORE"), 999) {
			t.Fatalf("首拍增量落账: %.4f", v.Balance("CORE"))
		}
		res, err = v.Resolve(ctx, leg, "p1")
		if err != nil || res.State != "filled" || !approx(res.FillSz, 1998) {
			t.Fatalf("补成拍应满成: %+v err=%v", res, err)
		}
		if !approx(v.Balance("CORE"), 1998) {
			t.Fatalf("补成增量落账: %.4f", v.Balance("CORE"))
		}
	})
	t.Run("ResolveCap=0 冻结部分态", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 100).
			WithPlan("CORE-USDT", Plan{FillRatio: 0.5})
		if res, _ := v.Place(ctx, leg, "p2"); res.State != "partially_filled" {
			t.Fatalf("应部分态: %+v", res)
		}
		res, _ := v.Resolve(ctx, leg, "p2")
		if res.State != "partially_filled" {
			t.Fatalf("无 Cap 应冻结部分态: %+v", res)
		}
		if err := v.Cancel(ctx, leg, "p2"); err != nil {
			t.Fatalf("部分态可撤: %v", err)
		}
		res, _ = v.Resolve(ctx, leg, "p2")
		if res.State != "cancelled" {
			t.Fatalf("撤后态: %+v", res)
		}
	})
	t.Run("已成交撤挂=错；不在册 Resolve=absent（51603 同语义）", func(t *testing.T) {
		v := New().WithInstrument(spot).WithBalance("USDT", 100)
		if _, err := v.Place(ctx, leg, "p3"); err != nil {
			t.Fatal(err)
		}
		if err := v.Cancel(ctx, leg, "p3"); err == nil {
			t.Fatal("已成交撤挂应错")
		}
		res, err := v.Resolve(ctx, leg, "ghost")
		if err != nil || res.State != "absent" {
			t.Fatalf("不在册 Resolve 应 absent: %+v err=%v", res, err)
		}
	})
}

// Sizer 缝语义（产线同语义：不在册=错 fail-closed）。
func TestSizerSemantics(t *testing.T) {
	ctx := context.Background()
	spot, swap := corePair()
	v := New().WithInstrument(spot).WithInstrument(swap).WithBalance("CORE", 123.45)
	v.pos["CORE-USDT-SWAP"] = -7.5
	if a, err := v.SpotAvailBal(ctx, "CORE-USDT"); err != nil || !approx(a, 123.45) {
		t.Fatalf("SpotAvailBal: %v %v", a, err)
	}
	if p, err := v.ContractPos(ctx, "CORE-USDT-SWAP"); err != nil || !approx(p, 7.5) {
		t.Fatalf("ContractPos abs: %v %v", p, err)
	}
	if l, err := v.LotSz(ctx, "CORE-USDT"); err != nil || l != 0.01 {
		t.Fatalf("LotSz: %v %v", l, err)
	}
	if _, err := v.SpotAvailBal(ctx, "GHOST-USDT"); err == nil {
		t.Fatal("不在册应 fail-closed 错")
	}
}
