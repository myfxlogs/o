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
		case strings.Contains(url, "frankfurter"):
			return []byte(`{"amount":1.0,"base":"USD","date":"2026-11-06","rates":{"EUR":0.89}}`), nil
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
	// kind 各≥1 锚（注记⑨产证锚对应件；T-0429 扩面 fx/equity/commodity 补维）
	for _, k := range []string{"macro", "oi", "cot", "fx", "equity", "commodity"} {
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

// T-0429 红桩①：ffcal 周历 XML 声明 windows-1252（ForexFactory 实测头）+
// 高位字节 0xE9（é）——Go xml 无 CharsetReader 拒读=病灶断言（修前红：
// parse_err 静默零行；修后绿：High→md_macro_event、Medium→ctx_fact 落行）。
func TestFFCharset1252(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool)
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		if strings.Contains(url, "ff_calendar_thisweek") {
			// 0xE9=windows-1252 "é"——解码败则标题形变/整拒。
			return []byte("<?xml version=\"1.0\" encoding=\"windows-1252\"?>\n<weeklyevents><event>\n" +
				"\t<title><![CDATA[CPI y/y]]></title><country>USD</country><date>11-05-2026</date>\n" +
				"\t<time>8:30am</time><impact>High</impact><url>/e/cpi</url>\n" +
				"\t</event><event>\n" +
				"\t<title><![CDATA[PMI r\xe9vis\xe9]]></title><country>USD</country><date>11-06-2026</date>\n" +
				"\t<time>10:00am</time><impact>Medium</impact><url>/e/pmi-revise</url>\n" +
				"\t</event></weeklyevents>"), nil
		}
		return nil, nil
	}
	if err := r.collectCalendar(c); err != nil {
		t.Fatal(err)
	}
	var src string
	if err := pool.QueryRow(c, `SELECT source FROM md_macro_event WHERE kind='cpi' AND title='CPI y/y'`).Scan(&src); err != nil {
		t.Fatal("windows-1252 周历 High 事件未落 md_macro_event（charset 拒读形）:", err)
	}
	if src != "ffcal" {
		t.Fatalf("source 应 ffcal: %s", src)
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='ffcal' AND dedupe_key='pmi-revise:11-06-2026'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("windows-1252 Medium 事件未落 ctx_fact（n=%d err=%v）", n, err)
	}
}

// T-0444 红桩：ffcal dedupe 键=slug:事件日期——FF url slug 无日期位
// （11-us-unemployment-claims 类周/月循环事件同键恒存），修前红：次周
// 同 slug 新发生撞 ON CONFLICT DO NOTHING 静默吞行（「断更」病灶——
// 生产实证 11-us-unemployment-claims 冻结在 11-05-2026 行）；修后绿：
// 新日期=新键=新行，同日重拍仍幂等。
func TestFFCalDedupeNewDate(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool)
	feedDate := "11-05-2026"
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		if strings.Contains(url, "ff_calendar_thisweek") {
			return []byte(`<?xml version="1.0" encoding="windows-1252"?>
<weeklyevents><event>
	<title><![CDATA[Unemployment Claims]]></title><country>USD</country><date>` + feedDate + `</date>
	<time>1:30pm</time><impact>Medium</impact><url>/e/claims</url>
	</event></weeklyevents>`), nil
		}
		return nil, nil
	}
	if err := r.collectCalendar(c); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='ffcal' AND dedupe_key='claims:11-05-2026'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("dedupe_key 应含事件日期（claims:11-05-2026）: n=%d err=%v", n, err)
	}
	feedDate = "11-12-2026" // 次周同 slug 新发生——周循环事件
	if err := r.collectCalendar(c); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='ffcal' AND dedupe_key='claims:11-12-2026'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("新日期应落新行: n=%d err=%v", n, err)
	}
	feedDate = "11-05-2026" // 回本周重拍——同键幂等不增行
	if err := r.collectCalendar(c); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='ffcal' AND dedupe_key LIKE 'claims:%'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("同键重拍幂等失守: n=%d err=%v", n, err)
	}
}

// T-0429 红桩②：FRED dedupe 键=序列:最新观察日——新观察日=新键=新行。
// 修前红：键=序列名，二拍同序列 ON CONFLICT DO NOTHING 恒静默（「断供」主
// 病灶——成功也零新行）；修后绿：换观察日即落新行。
func TestFredDedupeNewObsDate(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool).AttachFredKey("k")
	obsDate := "2026-11-05"
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		if strings.Contains(url, "fred") {
			return []byte(`{"observations":[{"date":"` + obsDate + `","value":"4.0"}]}`), nil
		}
		return nil, nil
	}
	if err := r.collectFred(c); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='fred' AND dedupe_key=$1`, "DGS10:"+obsDate).Scan(&n); err != nil || n != 1 {
		t.Fatalf("dedupe_key 应含观察日（DGS10:%s）: n=%d err=%v", obsDate, n, err)
	}
	obsDate = "2026-11-06" // FRED 新观察日发布——下一拍
	if err := r.collectFred(c); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='fred' AND dedupe_key=$1`, "DGS10:2026-11-06").Scan(&n); err != nil || n != 1 {
		t.Fatalf("新观察日应落新行: n=%d err=%v", n, err)
	}
}

// T-0429 锚③：frankfurter fx 源——日率行 dedupe=base:date 幂等。
func TestFxCollect(t *testing.T) {
	pool := testutil.Pool(t)
	c := context.Background()
	if err := pgstore.Migrate(c, pool, "md"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool)
	r.fetch = func(ctx context.Context, url string) ([]byte, error) {
		if strings.Contains(url, "frankfurter") {
			return []byte(`{"amount":1.0,"base":"USD","date":"2026-11-06","rates":{"EUR":0.89,"JPY":158.3}}`), nil
		}
		return nil, nil
	}
	if err := r.collectFx(c); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='frankfurter' AND kind='fx' AND dedupe_key='USD:2026-11-06'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("fx 行缺: n=%d err=%v", n, err)
	}
	if err := r.collectFx(c); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(c, `SELECT count(*) FROM md_ctx_fact WHERE source='frankfurter'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("fx 重拍幂等失守: n=%d", n)
	}
}
