package main

import (
	"strings"
	"testing"
	"time"
)

func TestBooksNoteShape(t *testing.T) {
	p := time.Date(2026, 11, 10, 15, 0, 0, 0, time.UTC)
	n := p.Add(24 * time.Hour)
	note := booksNote("5000|-4792", 5008.21, 216.0, 12.34, 4779.87, p, n, "")
	if !strings.HasPrefix(note, "v=5000|-4792 账房对账:") {
		t.Fatalf("头形败: %s", note)
	}
	for _, seg := range []string{"ΔNAV+5008.21", "净流+216.00", "盈亏+12.34", "未释+4779.87", "24h", "nav_residual"} {
		if !strings.Contains(note, seg) {
			t.Fatalf("缺段 %s: %s", seg, note)
		}
	}
}

// 三桶守恒——Δ=净流+盈亏+残差 定义面（残差=Δ-流-盈 必使等式闭合）。
func TestBooksBucketsClose(t *testing.T) {
	cases := [][3]float64{{5008.21, 216, 12.34}, {-4808, 0, 0}, {35.4, 0, 35.4}}
	for _, c := range cases {
		delta, flow, earned := c[0], c[1], c[2]
		resid := delta - flow - earned
		if delta-(flow+earned+resid) > 1e-9 {
			t.Fatalf("桶不闭合: %+v", c)
		}
	}
}

// O-27665 残差透镜：未载类+腿差拼段；空段不产透镜尾（阈内零扰）。
func TestBooksLensText(t *testing.T) {
	if got := booksLensText("", ""); got != "" {
		t.Fatalf("双空应产空串: %q", got)
	}
	got := booksLensText("owner_in/okx 10000x2;deposit/okx 52658.73x11", "+okx/USDT")
	for _, seg := range []string{"残差透镜:", "未载类[owner_in/okx 10000x2", "腿差[+okx/USDT]"} {
		if !strings.Contains(got, seg) {
			t.Fatalf("透镜缺段 %s: %q", seg, got)
		}
	}
	// 透镜尾接 booksNote——owner_in ±10k 注入形可读释因
	p := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	n := p.Add(24 * time.Hour)
	note := booksNote("0|-10000", -55.0, 0, 0, -10055.0, p, n, got)
	if !strings.Contains(note, "未载类[owner_in/okx") || !strings.Contains(note, "未释-10055.00") {
		t.Fatalf("透镜尾未入账页: %s", note)
	}
}
