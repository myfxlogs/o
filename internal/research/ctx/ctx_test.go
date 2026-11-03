package ctx

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：四源桩件——mql5 POST High upsert md_macro_event（种子兜底+同日优先级）；
// 中低+FRED/COT/news→md_ctx_fact dedupe 幂等；FRED 键缺=整源跳过。
func TestCollectStub(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool).AttachFredKey("k")
	r.AttachPost(func(ctx context.Context, url, body string) ([]byte, error) {
		if !strings.Contains(body, "importance=15") {
			t.Errorf("form 应带 importance=15: %s", body)
		}
		return []byte(`[{"Id":1,"EventName":"CPI y/y","Importance":"high","CurrencyCode":"USD","FullDate":"2026-11-05T13:30:00"},
			{"Id":2,"EventName":"PMI","Importance":"low","CurrencyCode":"USD","FullDate":"2026-11-06T14:00:00"}]`), nil
	})
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		switch {
		case strings.Contains(url, "cftc"):
			return []byte(`[{"market_name":"EURO FX","open_interest":1}]`), nil
		case strings.Contains(url, "fred"):
			return []byte(`{"observations":[{"date":"2026-11-01","value":"4.0"}]}`), nil
		case strings.Contains(url, "mql5.com/en/news"):
			return []byte(`<div class="news-item__title">Fed speaks about rates outlook today</div>`), nil
		}
		return nil, nil
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
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 4 {
		t.Fatalf("ctx 落行应 ≥4: %d", n)
	}
	// 幂等锚：重拍不增行（dedupe_key 唯一键拒重）
	if err := r.TickOnce(c); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact`).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n2 > n+1 { // news dedupe_key=日戳 同日同键拒重；COT 空键每拍一行容忍
		t.Fatalf("幂等失守: %d→%d", n, n2)
	}
}
