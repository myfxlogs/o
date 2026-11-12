package main

import (
	"strings"
	"testing"
	"time"
)

func TestPnlAttrNoteShape(t *testing.T) {
	p := time.Date(2026, 11, 11, 15, 0, 0, 0, time.UTC)
	n := p.Add(24 * time.Hour)
	rs := []pnlAttrRow{
		{"287059", "funding-hedge", "ALLO-USDT-SWAP", 1.10, 4},
		{"286010", "funding-hedge", "NOT-USDT-SWAP", 0.42, 2},
		{"unattr", "-", "-", -1.50, 7},
		{"286999", "xvenue-carry", "ME-USDT-SWAP", -0.88, 3},
	}
	note := pnlAttrNote("212|2|3", rs, 0.64, -1.50, 3, 0.64, p, n)
	if !strings.HasPrefix(note, "v=212|2|3 策略盈亏归因:") {
		t.Fatalf("头形败: %s", note)
	}
	for _, seg := range []string{"策动3", "Σ+0.64", "账房盈亏+0.64", "咬合差+0.00",
		"#287059", "ALLO-USDT-SWAP", "+1.10", "#286999", "-0.88", "未归因-1.50", "24h"} {
		if !strings.Contains(note, seg) {
			t.Fatalf("缺段 %s: %s", seg, note)
		}
	}
}

// 咬合闭合——Σ(归因+未归因)=attrSum 与盈亏桶差即咬合差（同基时≈0，
// 两查询异基则显数——不藏）。
func TestPnlAttrBiteCaliber(t *testing.T) {
	// attrSum=10.00 earned=10.00→咬合差0;earned=10.50→-0.50 显数
	rs := []pnlAttrRow{{"1", "t", "X", 6, 1}, {"unattr", "-", "-", 4, 1}}
	note := pnlAttrNote("10|4|1", rs, 10, 4, 1, 10.5,
		time.Now().Add(-20*time.Hour), time.Now())
	if !strings.Contains(note, "咬合差-0.50") {
		t.Fatalf("咬合差口径败: %s", note)
	}
}
