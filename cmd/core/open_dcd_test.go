// open_dcd_test.go —— T-0160 返修桩锚：dcd 行为级断言（测源码函数非字面；
// T-0461 实现体抽立 internal/trading/dcdexec——本件测同一函数面）。
package main

import (
	"strings"
	"testing"

	"o/internal/trading/dcdexec"
)

// 锚：ExecState 行为级——LIVE→open/SETTLED→closed/CANCELLED→failed；
// 未载态=空串让位（歧态势由调用径置 ambiguous——禁默认 open 造仓）。
func TestDcdExecStateMapping(t *testing.T) {
	for st, want := range map[string]string{
		"LIVE": "open", "SETTLED": "closed", "CANCELLED": "failed"} {
		if got := dcdexec.ExecState(st); got != want {
			t.Fatalf("dcdexec.ExecState(%s)=%q want %q", st, got, want)
		}
	}
	for _, bad := range []string{"", "PENDING", "bogus"} {
		if got := dcdexec.ExecState(bad); got != "" {
			t.Fatalf("未载态 %q 应空串让位（禁默认 open）: %q", bad, got)
		}
	}
}

// 锚：sz 去 sci——notional 格式化不产指数形（venue sz 方言拒 sci）。
func TestDcdSzFormatNoSci(t *testing.T) {
	for _, v := range []float64{1e3, 0.00001, 1234.5678, 2e6} {
		out := dcdexec.Sz(v) // 桩绑调用点——测源码函数非字面
		if strings.ContainsAny(out, "eE") {
			t.Fatalf("sz 应去 sci: %v→%s", v, out)
		}
	}
}
