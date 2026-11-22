package main

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

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

// TestPendingTSLag O-30635 Marten 型根修锚：ops_event id>水位游标带 60s
// 滞后余量——新鲜行本拍不入泄批（未提交窗内相对新行防跳读），回拨行照泄。
func TestPendingTSLag(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	fp := &feedPush{pool: pool}
	var agedID, freshID int64
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note,ts)
		VALUES('施工席','完工回执','aged',now()-interval '2 minutes') RETURNING id`).Scan(&agedID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('施工席','完工回执','fresh') RETURNING id`).Scan(&freshID); err != nil {
		t.Fatal(err)
	}
	rows, err := fp.pending(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	var sawAged, sawFresh bool
	for _, r := range rows {
		if r.id == agedID {
			sawAged = true
		}
		if r.id == freshID {
			sawFresh = true
		}
	}
	if !sawAged {
		t.Fatal("回拨行应入泄批")
	}
	if sawFresh {
		t.Fatal("60s 滞后窗内新鲜行不应入泄批（跳读防护失效）")
	}
}
