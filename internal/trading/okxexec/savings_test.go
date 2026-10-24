package okxexec

import (
	"testing"
)

// T-0059 锚①：fmtAmt 全额划拨语义=floor（向下截断 8 位）——禁四舍五入。
// $75.93 实证：75.92713387720735 第 9 位=7，Round 进位→请求额比实余大 2.8e-9
// →58350 Insufficient balance→死锁环。
func TestFmtAmtFloors(t *testing.T) {
	for _, tc := range []struct {
		in   float64
		want string
	}{
		{75.92713387720735, "75.92713387"}, // 实证案值（Round 会得 75.92713388）
		{75.92713385, "75.92713385"},       // 8 位整——原样
		{0.000000019, "0.00000001"},        // 第 9 位截——1e-8 ≤ 输入（floor 铁律）
		{1.999999999, "1.99999999"},        // 第 9 位=9 亦截断
		{10, "10"},
	} {
		got := fmtAmt(tc.in)
		if got != tc.want {
			t.Fatalf("fmtAmt(%v)=%q want %q", tc.in, got, tc.want)
		}
		if v := parseF(got); v > tc.in+1e-12 {
			t.Fatalf("floor 铁律违例: fmtAmt(%v)=%v > 输入", tc.in, v)
		}
	}
}
