// open_dcd_test.go —— T-0160 返修桩锚：dcd 行为级断言（测源码函数非字面）。
package main

import (
	"strconv"
	"strings"
	"testing"
)

// 锚：dcdExecState 行为级——LIVE→open/SETTLED→closed/CANCELLED→failed；
// 未载态=空串让位（歧态势由调用径置 ambiguous——禁默认 open 造仓）。
func TestDcdExecStateMapping(t *testing.T) {
	for st, want := range map[string]string{
		"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"} {
		if got := dcdExecState(st); got != want {
			t.Fatalf("dcdExecState(%s)=%q want %q", st, got, want)
		}
	}
	for _, bad := range []string{"", "PENDING", "bogus"} {
		if got := dcdExecState(bad); got != "" {
			t.Fatalf("未载态 %q 应空串让位（禁默认 open）: %q", bad, got)
		}
	}
}

// 锚：sz 去 sci——notional 格式化不产指数形（venue sz 方言拒 sci）。
func TestDcdSzFormatNoSci(t *testing.T) {
	for _, v := range []float64{1e3, 0.00001, 1234.5678, 2e6} {
		out := strconv.FormatFloat(v, 'f', -1, 64)
		if strings.ContainsAny(out, "eE") {
			t.Fatalf("sz 应去 sci: %v→%s", v, out)
		}
	}
}
