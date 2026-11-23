package dirmom

import "testing"

// TestLegNameCfdArm O-35163：mt5 CFD 臂 eval.legs 证据载荷=模板#2074
// legs 契源 cfd_long/cfd_short（direction 沿用 long_perp/short_perp
// 族位约定——direction CHECK 闭集已在册词不新增迁移）。
func TestLegNameCfdArm(t *testing.T) {
	mt5 := &Runner{template: "mt5_dir_mom"}
	if got := mt5.legName("long_perp"); got != "cfd_long" {
		t.Fatalf("mt5 long_perp leg=%q want cfd_long", got)
	}
	if got := mt5.legName("short_perp"); got != "cfd_short" {
		t.Fatalf("mt5 short_perp leg=%q want cfd_short", got)
	}
	okx := &Runner{template: Template}
	if got := okx.legName("long_perp"); got != "perp_long" {
		t.Fatalf("okx long_perp leg=%q want perp_long", got)
	}
}
