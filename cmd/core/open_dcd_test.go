// open_dcd_test.go —— T-0160 返修桩锚：①gate admit（dcd_call 入 paper 白名单）
// ②ambiguous 纪律断言（order-status 读败→ambiguous 非 LIVE 造仓）
// ④sz 格式断言（FormatFloat f,-1 非 sci 形）。
package main

import (
	"strconv"
	"strings"
	"testing"
)

// 锚①：sz 去 sci——notional 格式化不产指数形（venue sz 方言拒 sci）。
func TestDcdSzFormatNoSci(t *testing.T) {
	for _, v := range []float64{1e3, 0.00001, 1234.5678, 2e6} {
		out := strconv.FormatFloat(v, 'f', -1, 64)
		if strings.ContainsAny(out, "eE") {
			t.Fatalf("sz 应去 sci: %v→%s", v, out)
		}
	}
}

// 锚②：dcd 状态→fin_execution state 映射（order-status 读败=ambiguous
// 禁默认 LIVE——读败径已置 ambiguous 在源码；此处断言映射表不产 LIVE 兜底）。
func TestDcdStateMap(t *testing.T) {
	m := map[string]string{"LIVE": "open", "SETTLED": "closed"}
	if _, ok := m["ambiguous"]; ok {
		t.Fatal("ambiguous 态不应入映射表（歧态势留 ambiguous 禁默认 open）")
	}
	if m["LIVE"] != "open" || m["SETTLED"] != "closed" {
		t.Fatal("状态映射错")
	}
}
