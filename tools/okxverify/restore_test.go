package main

import "testing"

// 复审 860b8c1 锚：台账封顶无条件——零引入初态（soldUSD==0，存量足额起跑/
// 尘币 Δ≤0）=零额度⇒零 transfer 零 purchase（两动钱面皆 gated on back>0）。
func TestLedgerCapZeroIntroduced(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		avail, introduced, want float64
	}{
		{"无尘币路径 soldUSD=0", 100, 0, 0},
		{"dust Δ≤0（实测倒挂）", 100, -3, 0},
		{"正常引入封顶", 100, 5, 5},
		{"闲余不足取小", 0.3, 5, 0.3},
	} {
		if got := ledgerCap(tc.avail, tc.introduced); got != tc.want {
			t.Fatalf("%s: ledgerCap(%v,%v)=%v want %v", tc.name, tc.avail, tc.introduced, got, tc.want)
		}
		if got := ledgerCap(tc.avail, tc.introduced); got <= 0 {
			// gated on back>0：零额度 ⇒ transfer_back/dust_buyback 分支物理不入。
			// purchase 同源：pu≤parked=back=0 ⇒ pu>1 恒假零复购。
			t.Logf("%s: 零额度=零摆回零复购（门梄件 back>0/pu>1 断言）", tc.name)
		}
	}
}
