package ctx

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// 锚：FF 周历 XML High→md_macro_event source='ffcal'（种子兜底+优先级）；
// 中低/FRED/COT→ctx dedupe slug 幂等；FRED 键缺=跳过。
func TestCollectStub(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool).AttachFredKey("k")
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		switch {
		case strings.Contains(url, "ff_calendar_thisweek"):
			return []byte(`<weeklyevents><event>
				<title>CPI y/y</title><country>USD</country><date>11-05-2026</date>
				<time>8:30am</time><impact>High</impact><url>/e/cpi</url>
				</event><event>
				<title>PMI</title><country>USD</country><date>11-06-2026</date>
				<time>10:00am</time><impact>Medium</impact><url>/e/pmi</url>
				</event></weeklyevents>`), nil
		case strings.Contains(url, "cftc"):
			return []byte(`[{"market_name":"EURO FX","open_interest":1}]`), nil
		case strings.Contains(url, "fred"):
			return []byte(`{"observations":[{"date":"2026-11-01","value":"4.0"}]}`), nil
		case strings.Contains(url, "open-interest"):
			return []byte(`{"data":[{"instId":"BTC-USDT-SWAP","oi":"1000","oiCcy":"1000"}]}`), nil
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
	if src != "ffcal" {
		t.Fatalf("source 应 ffcal: %s", src)
	}
	var seedN int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_macro_event WHERE source='seed'`).Scan(&seedN); err != nil || seedN == 0 {
		t.Fatal("种子行不得删")
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 6 {
		t.Fatalf("ctx 落行应 ≥6（PMI+FRED×4+COT+OI）: %d", n)
	}
	// kind 各≥1 锚（注记⑨产证锚对应件）
	for _, k := range []string{"macro", "oi", "cot"} {
		var kn int
		if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE kind=$1`, k).Scan(&kn); err != nil || kn < 1 {
			t.Fatalf("kind %s 行缺: %v", k, err)
		}
	}
	// 幂等锚：重拍不增行（slug dedupe 拒重）
	if err := r.TickOnce(c); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact`).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n2 > n+1 { // COT 行 id 键拒重；OI 日戳同键拒重——容 1 漂移
		t.Fatalf("幂等失守: %d→%d", n, n2)
	}
}

// 锚：ffDate——"All Day"=仅日期；解析败=false。
func TestFFDate(t *testing.T) {
	if _, ok := ffDate("bad", "8:30am"); ok {
		t.Fatal("坏日期应 false")
	}
	if d, ok := ffDate("11-05-2026", "All Day"); !ok || d.Hour() != 0 {
		t.Fatal("All Day 应仅日期 0 时")
	}
	if d, ok := ffDate("11-05-2026", "8:30am"); !ok {
		t.Fatal("8:30am 应解析成功")
	} else if d.Year() != 2026 || d.Month() != 11 || d.Day() != 5 {
		t.Fatal("日期错")
	}
}
