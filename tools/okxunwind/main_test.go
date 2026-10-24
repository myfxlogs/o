package main

import "testing"

// 复审 860b8c1 锚：摆回封顶无条件——零所得初态（proceeds<=0，无货可卖/
// 卖出倒挂）=零摆回⇒零 transfer 零 purchase（transfer gated on back>0；
// purchase pu≤parked=0 ⇒ pu>1 恒假）。
func TestBackCapZeroProceeds(t *testing.T) {
	for _, tc := range []struct {
		name                string
		tra, proceeds, want float64
	}{
		{"零所得路径 proceeds=0", 100, 0, 0},
		{"卖出倒挂 proceeds<0", 100, -2, 0},
		{"正常所得封顶", 100, 7, 7},
		{"闲余不足取小", 0.5, 9, 0.3},
	} {
		if got := backCap(tc.tra, tc.proceeds); got != tc.want {
			t.Fatalf("%s: backCap(%v,%v)=%v want %v", tc.name, tc.tra, tc.proceeds, got, tc.want)
		}
	}
}
