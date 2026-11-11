package main

import (
	"strings"
	"testing"
)

func TestArmedNoteShape(t *testing.T) {
	as := []armedRow{
		{inst: "VVV-USDT-SWAP", reason: "breakeven_beyond_hold", ratio: 0.36, hasR: true},
		{inst: "MET-USDT-SWAP", reason: "breakeven_beyond_hold", ratio: -0.42, hasR: true},
		{inst: "XTZ-USDT-SWAP", reason: "breakeven_beyond_hold", ratio: 0.36, hasR: true},
		{inst: "BTC-USDT-SWAP", reason: "lifetime_below_min"},
		{inst: "SOMEPERP", reason: ""},
	}
	note, fp := armedNote(as)
	if !strings.HasPrefix(note, "v="+fp+" armed距阈 N=5") {
		t.Fatalf("头形败: %s", note)
	}
	if !strings.Contains(note, "breakeven_beyond_hold×3") ||
		!strings.Contains(note, "lifetime_below_min×1") ||
		!strings.Contains(note, "无拒件×1") {
		t.Fatalf("主因谱败: %s", note)
	}
	// 近阈序：两 0.36 居前，负值殿后；无 ratio 件不入近阈榜
	i1, i2, i3 := strings.Index(note, "VVV-USDT-SWAP"), strings.Index(note, "XTZ-USDT-SWAP"),
		strings.Index(note, "MET-USDT-SWAP")
	if i1 < 0 || i2 < 0 || i3 < 0 || i3 < i1 || i3 < i2 {
		t.Fatalf("近阈序败: %s", note)
	}
	if strings.Contains(note, "BTC-USDT-SWAP ") || strings.Contains(note, "SOMEPERP ") {
		t.Fatalf("无 ratio 件混入近阈榜: %s", note)
	}
	if !strings.Contains(note, "36%") || !strings.Contains(note, "-42%") {
		t.Fatalf("比值渲败: %s", note)
	}
}

func TestArmedNoteEmpty(t *testing.T) {
	note, _ := armedNote(nil)
	if !strings.Contains(note, "N=0") || strings.Contains(note, "近阈") {
		t.Fatalf("空集形败: %s", note)
	}
}

func TestFpHashStable(t *testing.T) {
	a := fpHash([]string{"A:0.360", "B:0.220"})
	b := fpHash([]string{"A:0.360", "B:0.220"})
	if a != b {
		t.Fatal("同集异纹")
	}
	if fpHash([]string{"A:0.361", "B:0.220"}) == a {
		t.Fatal("比值微动纹不变——指纹失效")
	}
	if fpHash([]string{"A:0.360", "C:0.220"}) == a {
		t.Fatal("成员换纹不变——指纹失效")
	}
}
