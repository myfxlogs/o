package okxexec

import "testing"

// O-33377④ 族路由桩测——287249 幻影退役回归：DCD 产品键形→dcd 族，
// 期权键形→option 族，跨族判不存在=前提错。
func TestInstFamily(t *testing.T) {
	cases := []struct {
		instID string
		want   string
	}{
		{"BTC-USDT-261204-66000-P", "dcd"},         // 287249 幻影案实键
		{"BTC-USDT-261225-95000-C", "dcd"},         // 11448262 LIVE 实键
		{"ETH-USDT-261021-2280-P", "dcd"},          // 11401353 结算实键
		{"SIM-BTC-USDT-261204-66000-P", "unknown"}, // SIM- 前缀超族形——夹具键泄漏是另一病灶不猜族
		{"BTC-USD_UM-261124-68000-C", "option"},    // OKX OPTION 实键形
		{"BTC-USD_UM-261124-68000-P", "option"},
		{"BTC-USDT-261204", "futures"}, // datedFUT -YYMMDD
		{"BTC-USDT-SWAP", "swap"},
		{"BTC-USDT", "spot"},
		{"ETH-USDT", "spot"},
		{"", "unknown"},
		{"BTC-USDT-261204-66000", "unknown"}, // 四段无 C/P 尾=非 venue 键形不猜族
	}
	for _, c := range cases {
		if got := InstFamily(c.instID); got != c.want {
			t.Errorf("InstFamily(%q)=%q want %q", c.instID, got, c.want)
		}
	}
}

// 287249 回归案：DCD 产品键经 instruments 族核验=族错——InstFamily 须判 dcd
// 非 option/futures，使「核验打错族」在判词层可被识别（族标随判词行）。
func TestDcdKeyNotInstrumentsFamily(t *testing.T) {
	if !DcdProductKey("BTC-USDT-261204-66000-P") {
		t.Fatal("DCD 产品键未入 dcd 族——287249 幻影退役病灶未根修")
	}
	if DcdProductKey("BTC-USD_UM-261124-68000-C") {
		t.Fatal("期权合约键误判 dcd 族——族路由越界")
	}
	if DcdProductKey("BTC-USDT-SWAP") {
		t.Fatal("perp 键误判 dcd 族")
	}
	// 判词族标识件——不存在判词行必须携所查族。
	if FamilyCheckNote("dcd") != "[核验族=dcd]" {
		t.Fatal("族标识件形漂移")
	}
}
