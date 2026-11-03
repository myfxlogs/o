package ctx

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：四源桩 fetcher→macro High upsert md_macro_event（种子兜底）+ctx 落行。
func TestCollectStub(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool)
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		switch {
		case url == "https://www.mql5.com/en/economic-calendar/content?date="+r.now().UTC().Format("2006-01-02"):
			return []byte(`[{"eventId":1,"event":"CPI y/y","currency":"USD","importance":3,"time":"2026-11-05"},{"eventId":2,"event":"PMI","currency":"USD","importance":1,"time":"2026-11-06"}]`), nil
		case url == "https://www.mql5.com/en/news":
			return []byte(`<div class="news-item__title">Fed speaks about rates outlook today</div>`), nil
		default:
			return []byte(`{"observations":[{"date":"2026-11-01","value":"4.0"}]}`), nil
		}
	}
	if err := r.TickOnce(c); err != nil {
		t.Fatal(err)
	}
	var src string
	if err := pool.QueryRow(c, `SELECT source FROM md_macro_event WHERE kind='cpi' AND title='CPI y/y'`).Scan(&src); err != nil {
		t.Fatal("High 事件未落 md_macro_event:", err)
	}
	if src != "mql5" {
		t.Fatalf("source 应 mql5: %s", src)
	}
	var seedN int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_macro_event WHERE source='seed'`).Scan(&seedN); err != nil || seedN == 0 {
		t.Fatal("种子行不得删")
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE kind IN ('macro','oi','cot','news')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Fatalf("ctx 落行应 ≥3: %d", n)
	}
}
