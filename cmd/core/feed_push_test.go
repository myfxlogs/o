package main

import "testing"

func TestFeedTidy(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"unit=o-cognition-1794678602 host=ant rc=0 reason=session_incomplete dur=32s 第6轮:干了三件",
			"值班·尾8602 · @ant · 会话未完整 · 32 秒｜第6轮:干了三件"},
		{"eval-belief-history: eval 行被 upsert 覆写\n造成时代错配幻影",
			"eval-belief-history: eval 行被 upsert 覆写 造成时代错配幻影"},
		{"审计 P2 干净——零疑点", "审计 P2 干净——零疑点"},
	} {
		if got := feedTidy(c.in); got != c.want {
			t.Fatalf("feedTidy(%q)=%q want %q", c.in, got, c.want)
		}
	}
}
