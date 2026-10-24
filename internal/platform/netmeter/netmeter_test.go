package netmeter

import (
	"context"
	"testing"
)

// TestBudgetGateTwoTiers 80% 预警/95% 临界两档+70% 复位复报——同档不重复推。
func TestBudgetGateTwoTiers(t *testing.T) {
	var got []string
	m := New(1, func(_ context.Context, kind, _ string) { got = append(got, kind) }) // 1KB/s 预算
	m.Add("test", 100000)                                                            // 100000B/60s≈1.6KB/s>>95%=临界
	m.Add("test", 100000)                                                            // 同档不重复推
	if len(got) != 1 || got[0] != "net_budget_critical" {
		t.Fatalf("临界推=%v（应一发不重复）", got)
	}
}

// TestChannelsPerChannel 分通道累计字节账（瘦身建议数据源）。
func TestChannelsPerChannel(t *testing.T) {
	m := New(0, nil)
	m.Add("okx_pub/tickers", 100)
	m.Add("okx_pub/books5", 500)
	m.Add("mt5/quote", 200)
	ch := m.Channels()
	if ch["okx_pub/books5"] != 500 || ch["mt5/quote"] != 200 {
		t.Fatalf("channels=%v", ch)
	}
}
