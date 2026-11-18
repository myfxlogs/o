// instform_test.go —— 工单#15943 锚：inst 构型 vs 族腿构型闸——期权/远期
// 形 inst 不得进非期权/远期族（BETH-USDT-*-C 误建 funding-hedge 实案同案拒）。
package grammar

import "testing"

func TestAcceptsInstFormGate(t *testing.T) {
	if _, ok := LookupFamily("funding-hedge"); !ok {
		t.Fatal("funding-hedge 未在册")
	}
	cases := []struct {
		fam, inst string
		want      bool
	}{
		// 病灶原案：期权 inst 进 carry 族=拒（econ 矿#273/#274 同款）
		{"funding-hedge", "ETH-RLUSD-261106-2770-C", false},
		{"funding-hedge", "BETH-USDT-261027-2540-C", false},
		// 远期形 inst 进 carry 族=拒
		{"funding-hedge", "BTC-USDT-261225", false},
		// 期权形 inst 进 vol-sell 族=放（dcd 本家）
		{"dcd_call", "BETH-USDT-261027-2540-C", true},
		{"dcd_put", "BETH-USDT-261027-2540-P", true},
		{"vol_sell_strangle", "ETH-USDT-261128-4000-P", true},
		// 远期形进 basis 族=放
		{"quarter_basis_long", "BTC-USDT-261225", true},
		{"quarter_basis_short", "BTC-USDT-261225", true},
		// swap/spot 形现状放行（D-0018 另闸——本闸只核构型不拦现货面）
		{"funding-hedge", "INTC-USDT-SWAP", true},
		{"funding-hedge", "ETH-USDT", true},
		{"dir_unreg", "ETH-USDT", true},
	}
	for _, c := range cases {
		f, ok := LookupFamily(c.fam)
		if !ok {
			if !c.want { // 未注册族=构型闸让位不拦（LookupFamily 返 false 即放行侧）
				continue
			}
			continue // 未注册模板不入核——本闸不拦
		}
		if got := f.AcceptsInst(c.inst); got != c.want {
			t.Fatalf("%s.Accept(%s)=%v want %v", c.fam, c.inst, got, c.want)
		}
	}
}
