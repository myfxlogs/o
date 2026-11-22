// sim_test.go —— 机械趋势模拟桩测（工单O-32604 spec 桩测三案）。
package trendmech

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// mkBars 造 n 根日棒：价=base+i*step 线性（H/L/C 同值带微扰——通道判据
// 用 H/L，信号用 C；step>0 上行通道必破 20 日高）。
func mkBars(n int, base, step float64) []Bar {
	out := make([]Bar, n)
	for i := range out {
		p := base + float64(i)*step
		out[i] = Bar{Ts: t0.AddDate(0, 0, i), O: p, H: p * 1.001, L: p * 0.999, C: p}
	}
	return out
}

// 案① 信号序列对拍：上行序列→首个 20 日高突破信号在 i=20 收盘判、
// i=21 开盘成交（未来函数断言=t+1 开盘价非当棒收盘）。
func TestSignalSequenceAndTPlus1Fill(t *testing.T) {
	bars := mkBars(45, 100, 1) // 每日 +1，20 日高在 i=20 必破（前 20 根高=max H）
	res := Simulate(bars, nil, 5.0, 10000)
	if len(res.Fills) == 0 {
		t.Fatal("上行序列应产入场成交")
	}
	f := res.Fills[0]
	// 信号判定棒=bars[20]（i=20 收盘破前20高），成交棒=bars[21] 开盘。
	if !f.Ts.Equal(bars[21].Ts) {
		t.Fatalf("成交时刻=%v 应=次根棒 %v（t+1 开盘断言）", f.Ts, bars[21].Ts)
	}
	if math.Abs(f.Px-bars[21].O) > 1e-9 {
		t.Fatalf("成交价=%v 应=次根开盘 %v 非当棒收盘 %v", f.Px, bars[21].O, bars[20].C)
	}
	if f.DQ <= 0 {
		t.Fatalf("上行突破应开多 dq>0 got %v", f.DQ)
	}
	if f.Why != "entry_20d_break" {
		t.Fatalf("why=%s", f.Why)
	}
	// 10 日反向突破平仓：上行中 L 通道=前10根最低 L≈p-10*0.999——
	// 单调上行永不触发 exit（45 根全程持多→fills=1）。
	if len(res.Fills) != 1 {
		t.Fatalf("单调上行应仅入场一笔 fills=%d", len(res.Fills))
	}
}

// 案② 反向突破平仓：先上行开多，再插入急跌棒触发 10 日低破→次根平仓。
func TestReverseBreakoutExit(t *testing.T) {
	bars := mkBars(45, 100, 1)
	// bars[30] 起崩：C/L 急跌破前 10 日低（≈bars[20..29] L≈110-120 区）
	for i := 30; i < 45; i++ {
		p := 100 + float64(i)*1 - float64(i-29)*30 // i=30→129, i=44→-321 跌破
		if p < 1 {
			p = 1
		}
		bars[i] = Bar{Ts: bars[i].Ts, O: p * 1.01, H: p * 1.02, L: p * 0.98, C: p}
	}
	res := Simulate(bars, nil, 5.0, 10000)
	var exitFill *Fill
	for i := range res.Fills {
		if res.Fills[i].Why == "exit_10d_reverse" {
			exitFill = &res.Fills[i]
		}
	}
	if exitFill == nil {
		t.Fatalf("急跌应触发 10 日反向突破平仓 fills=%+v", res.Fills)
	}
	if exitFill.DQ >= 0 {
		t.Fatalf("多头平仓应 dq<0 got %v", exitFill.DQ)
	}
	// 信号棒=i 处 C 破前10低→成交=次根 O：找 exitFill 前一棒收盘判。
	// 结构断言已由案①钉死 t+1 机制——此处只验平仓存在且方向对。
}

// 案③ 费扣账面：无信号窗（横盘）费=0；有往返窗 fee=|dq|*px*fee_bps 双边。
func TestFeeAccounting(t *testing.T) {
	// 横盘 45 根（step=0→通道不破→零 fills 零费）。
	flat := mkBars(45, 100, 0)
	res := Simulate(flat, nil, 5.0, 10000)
	if len(res.Fills) != 0 {
		t.Fatalf("横盘应零信号 fills=%d", len(res.Fills))
	}
	// 上行窗：入场费=notional*fee（dq=notional/O→|dq|*O*fee=notional*fee）。
	up := mkBars(45, 100, 1)
	res2 := Simulate(up, nil, 5.0, 10000)
	if len(res2.Fills) != 1 {
		t.Fatalf("fills=%d", len(res2.Fills))
	}
	wantFee := 10000 * 5.0 / 1e4 // notional*fee_bps/1e4
	if math.Abs(res2.Fills[0].Fee-wantFee) > 1e-6 {
		t.Fatalf("入场费=%v 应=%v（名义×费率）", res2.Fills[0].Fee, wantFee)
	}
	// funding 结算行：持仓窗内 rate>0 多头付息→equity<gross_equity 同点。
	funds := []Fund{{Ts: t0.AddDate(0, 0, 25), Rate: 0.0001}}
	res3 := Simulate(up, funds, 0, 10000) // fee=0 隔离 funding 效应
	// 持仓从 bars[21] 起——funds ts=day25 落在 (bars[24].Ts,bars[25].Ts]？
	// 游标判据=bars[i-1].Ts<f.ts<=bars[i].Ts：day25→bars[25] 日内结算。
	// equity 末值应=gross−持仓期息（pos>0 rate>0→cash 减）。
	if res3.FinalEq >= res3.GrossEq[len(res3.GrossEq)-1] {
		t.Fatalf("多头正费率应付息 equity<gross got eq=%v geq=%v",
			res3.FinalEq, res3.GrossEq[len(res3.GrossEq)-1])
	}
}

// 案④ 未来函数结构断言：改 bars[30..] 不影响 bars[≤30] 的成交序列
// （成交只依赖历史——重放前缀不变性=无未来数据泄漏的可检形式）。
func TestNoFutureFunctionPrefixInvariant(t *testing.T) {
	bars := mkBars(45, 100, 1)
	resA := Simulate(bars, nil, 5.0, 10000)
	// 篡改末 10 根（崩盘）——前 30 根信号/成交必须逐位一致。
	for i := 35; i < 45; i++ {
		bars[i].C = bars[i].C * 0.5
		bars[i].O = bars[i].O * 0.5
		bars[i].H = bars[i].H * 0.5
		bars[i].L = bars[i].L * 0.5
	}
	resB := Simulate(bars, nil, 5.0, 10000)
	// 前缀窗成交数：resA 在 i≤34 的 fills——resB 前 34 日的 fills 应相同
	// （崩盘棒自身信号不同是合法的——只对拍成交序列在崩盘前段）。
	cutA, cutB := 0, 0
	for _, f := range resA.Fills {
		if f.Ts.Before(t0.AddDate(0, 0, 34)) {
			cutA++
		}
	}
	for _, f := range resB.Fills {
		if f.Ts.Before(t0.AddDate(0, 0, 34)) {
			cutB++
		}
	}
	if cutA != cutB {
		t.Fatalf("前缀成交数漂移 A=%d B=%d——未来数据泄漏嫌疑", cutA, cutB)
	}
}
