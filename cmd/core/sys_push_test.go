package main

import (
	"testing"
	"time"
)

// 去黑话族表驱——生产实录形对拍（业主令：卡体去黑话处理）。

func TestShortUnit(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"o-cognition-1794678602", "唤知·尾8602"},
		{"o-cognition-monologue", "唤知·独白"},
		{"o-cog", "唤知"},
		{"o-cognition-ab", "唤知·ab"},
		{"o-sweeper", "sweeper"},
		{"plain", "plain"},
	} {
		if got := shortUnit(tc.in); got != tc.want {
			t.Errorf("shortUnit(%q)=%q 要 %q", tc.in, got, tc.want)
		}
	}
}

func TestHumanizeKV(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"唤起果全字段", "unit=o-cognition-1794678602 host=ant rc=0 reason=session_incomplete dur=32s",
			"唤知·尾8602 · @ant · 会话未完整 · 32 秒"},
		{"独白果", "unit=o-cognition-monologue host=models rc=0 reason=session_incomplete dur=68s",
			"唤知·独白 · @models · 会话未完整 · 1 分 8 秒"},
		{"收口带正文", "unit=o-cognition-1794677643 第159轮:面净(deploy=master e50b5929ce09,升件0) 用帧=无",
			"唤知·尾7643｜第159轮:面净(部署 e50b5929ce09,升件0) 未用帧"},
		{"独白席无kv", "独白#16934 产=同一性边界洞成族",
			"独白#16934 产=同一性边界洞成族"},
	} {
		if got := humanizeKV(tc.in); got != tc.want {
			t.Errorf("%s: humanizeKV=%q 要 %q", tc.name, got, tc.want)
		}
	}
	if got := humanizeKV("unit=o-cognition-1 host=ant rc=1 reason=error dur=5s"); got != "唤知·1 · @ant · 退出码1 · 出错 · 5 秒" {
		t.Errorf("rc非零显: %q", got)
	}
}

func TestHumanizeNarr(t *testing.T) {
	in := "第6日: 唤71场/收65/败1; 处置46件; 工单13项 |在想=审计二巡干净 |昨收=unit=o-cognition-1794677643"
	want := "第 6 日: 唤起 71·收口 65·败 1｜处置 46｜工单 13 ｜在想：审计二巡干净 ｜昨收：唤知·尾7643"
	if got := humanizeNarr(in); got != want {
		t.Fatalf("humanizeNarr=%q\n要 %q", got, want)
	}
}

func TestDurCN(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"32s", "32 秒"},
		{"68s", "1 分 8 秒"},
		{"2h5m", "2 小时 5 分"},
		{"bogus", "bogus"},
	} {
		if got := durCN(tc.in); got != tc.want {
			t.Errorf("durCN(%q)=%q 要 %q", tc.in, got, tc.want)
		}
	}
}

func TestAgoS(t *testing.T) {
	if got := agoS(42 * time.Minute); got != "42 分钟前" {
		t.Errorf("agoS(42m)=%q", got)
	}
	if got := agoS(30 * time.Second); got != "刚刚" {
		t.Errorf("agoS(30s)=%q", got)
	}
	if got := agoS(185 * time.Minute); got != "3 小时 5 分前" {
		t.Errorf("agoS(185m)=%q", got)
	}
}

func TestSHTZ(t *testing.T) {
	l := shTZ()
	ts := time.Date(2026, 11, 15, 1, 52, 0, 0, l)
	if ts.UTC().Hour() != 17 {
		t.Fatalf("沪时偏移错: %v→UTC %v", ts, ts.UTC())
	}
}
