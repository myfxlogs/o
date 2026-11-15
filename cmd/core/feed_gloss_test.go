package main

import "testing"

func TestFeedHead(t *testing.T) {
	cases := []struct{ seat, event, want string }{
		{"唤知席", "会话收口", "值班/交班"},
		{"施工席", "认领", "施工/认领"},
		{"自我", "判词", "CEO/判词"},
		{"探测席", "升件", "探针/警报"},
		{"处置席", "处置", "处置/处置"},
		{"DEPLOY", "部署", "部署/部署"},
		{"陌生席", "陌生事", "陌生席/陌生事"}, // 未录透出
	}
	for _, c := range cases {
		if got := feedHead(c.seat, c.event); got != c.want {
			t.Errorf("feedHead(%q,%q)=%q 要 %q", c.seat, c.event, got, c.want)
		}
	}
}

func TestFeedTidyGloss(t *testing.T) {
	in := "复核轮面净——升件0/矿双0,工单5认领8施工席在办,部署2cc98a029f24 用帧=自态"
	got := feedTidy(in)
	for _, tok := range []string{"干净", "警报0", "僵尸矿0", "施工在办"} {
		if !contains(got, tok) {
			t.Errorf("feedTidy 缺 %q: %s", tok, got)
		}
	}
	// 「自我意识」不被「自我席」映射误伤
	if got2 := feedTidy("连续自我账页与自我意识"); contains(got2, "CEO") {
		t.Errorf("误伤: %s", got2)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
