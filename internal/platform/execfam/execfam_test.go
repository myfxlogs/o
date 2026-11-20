package execfam

import "testing"

// TestRiskShapeRegistryComplete 册内族必有形态标——扩族漏登=闸按
// leveraged 缺册兜底从严（行为仍 fail-closed，但册面缺=判词归因失真；
// 本测把缺口前置到测试期而非运行期静默兜底）。
func TestRiskShapeRegistryComplete(t *testing.T) {
	for tmpl := range set {
		if _, ok := riskShapes[tmpl]; !ok {
			t.Fatalf("execfam 族 %s 缺 risk_shape 标（缺册兜底按 leveraged 从严——须显式登记）", tmpl)
		}
	}
	for tmpl := range riskShapes {
		if !set[tmpl] {
			t.Fatalf("risk_shapes 含册外族 %s（两册漂移——须同册同步）", tmpl)
		}
	}
}

// TestShapeBoundedOnly 直过判据：bounded 单标族=true，含 leveraged/
// funded 位族与缺册族=false（后者 fail-closed 入保证金面审）。
func TestShapeBoundedOnly(t *testing.T) {
	for _, tmpl := range []string{"stake-carry", "grid_spot", "stable-park-usdg", "staking-defi-parking"} {
		if !ShapeBoundedOnly(tmpl) {
			t.Fatalf("族 %s 应 bounded 单标直过", tmpl)
		}
	}
	for _, tmpl := range []string{"funding-hedge", "xvenue-carry", "dcd_call", "vol_sell_strangle", "quarter_basis_long", "sprd-atomic-basis", "signal-bot-relay", "no-such-fam"} {
		if ShapeBoundedOnly(tmpl) {
			t.Fatalf("族 %s 不应 bounded 直过（缺册也不得直过——fail-closed）", tmpl)
		}
	}
}

// TestShapeGateVerdictBounded bounded 直过先于参数判——参数全坏/本金零
// 仍过（判例#1 免审族不受保证金面参数面影响）。
func TestShapeGateVerdictBounded(t *testing.T) {
	ok, why, ownerEsc := ShapeGateVerdict("grid_spot", 100, 0, 0, 0)
	if !ok || ownerEsc {
		t.Fatalf("grid_spot 应直过：ok=%v ownerEsc=%v why=%s", ok, ownerEsc, why)
	}
}

// TestShapeGateVerdictMargin 保证金面审三态：
// ①需求≤本金→放行（面审过）；②需求>本金→硬拒+越线归业主标；
// ③参数非正/本金非正→判据缺 fail-closed 拒且不归业主（参数件修复径）。
func TestShapeGateVerdictMargin(t *testing.T) {
	// funding-hedge=leveraged|funded；demand=100/1+100×1=200≤1000 放行
	ok, why, ownerEsc := ShapeGateVerdict("funding-hedge", 100, 1000, 1.0, 1.0)
	if !ok || ownerEsc {
		t.Fatalf("面审过应放行：ok=%v ownerEsc=%v why=%s", ok, ownerEsc, why)
	}
	// demand=100000/1+100000×1=200000>1000→硬拒+ownerEsc
	ok, why, ownerEsc = ShapeGateVerdict("funding-hedge", 100000, 1000, 1.0, 1.0)
	if ok || !ownerEsc {
		t.Fatalf("越线应硬拒且归业主：ok=%v ownerEsc=%v why=%s", ok, ownerEsc, why)
	}
	// lev=0→判据缺拒且不归业主
	ok, _, ownerEsc = ShapeGateVerdict("funding-hedge", 100, 1000, 0, 1.0)
	if ok || ownerEsc {
		t.Fatalf("lev=0 应判据缺拒不归业主：ok=%v ownerEsc=%v", ok, ownerEsc)
	}
	// principal=0（本金线读缺）→判据缺拒不归业主
	ok, _, ownerEsc = ShapeGateVerdict("funding-hedge", 100, 0, 1.0, 1.0)
	if ok || ownerEsc {
		t.Fatalf("principal=0 应判据缺拒不归业主：ok=%v ownerEsc=%v", ok, ownerEsc)
	}
	// 缺册族按 leveraged 从严同面审（nonexist 同 funding-hedge 判词路径）
	ok, _, _ = ShapeGateVerdict("no-such-fam", 100, 0, 1.0, 1.0)
	if ok {
		t.Fatal("缺册族本金线读缺应 fail-closed 拒")
	}
}

// TestShapeGateVerdictLeveragedParam 杠杆参数入算：lev=2 时保证金项
// 减半——demand=100/2+100×1=150；同本金线下 lev=1 的 200 过而更大
// notional 越线的分界随参数动（参数=判据输入件非硬线，闸按实参判）。
func TestShapeGateVerdictLeveragedParam(t *testing.T) {
	// notional=10000,lev=2,lossFrac=0.05,principal=6000:
	// demand=5000+500=5500≤6000 放行；lev=1 时 demand=10500>6000 硬拒——
	// 同笔同本金，杠杆假设翻转判词（证明参数真入算非摆设）。
	ok1, _, _ := ShapeGateVerdict("funding-hedge", 10000, 6000, 2.0, 0.05)
	ok2, _, esc2 := ShapeGateVerdict("funding-hedge", 10000, 6000, 1.0, 0.05)
	if !ok1 {
		t.Fatal("lev=2 面审应过（demand=5500≤6000）")
	}
	if ok2 || !esc2 {
		t.Fatal("lev=1 应硬拒且归业主（demand=10500>6000）")
	}
}

// TestRiskShapeMaskBits 形态位语义抽查：funding-hedge 应双位齐
// （leveraged|funded），dcd_call 单 leveraged 位（无资金费拖磨）。
func TestRiskShapeMaskBits(t *testing.T) {
	m, ok := RiskShape("funding-hedge")
	if !ok || m != ShapeLeveraged|ShapeFunded {
		t.Fatalf("funding-hedge 形态位应=leveraged|funded，实得 %08b ok=%v", m, ok)
	}
	m, ok = RiskShape("dcd_call")
	if !ok || m != ShapeLeveraged {
		t.Fatalf("dcd_call 形态位应=leveraged 单标，实得 %08b ok=%v", m, ok)
	}
	m, ok = RiskShape("grid_spot")
	if !ok || m != ShapeBounded {
		t.Fatalf("grid_spot 形态位应=bounded 单标，实得 %08b ok=%v", m, ok)
	}
	if _, ok := RiskShape("no-such-fam"); ok {
		t.Fatal("缺册族 RiskShape 应 registered=false")
	}
}
