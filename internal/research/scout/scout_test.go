package scout

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/research/funding"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

// 锚①采集：桩 fetcher 三 venue 响应→md_venue_funding 落行。
func TestCollectStub(t *testing.T) {
	ctx, r := setup(t)
	r.fetch = func(ctx context.Context, url, body string) ([]byte, error) {
		switch {
		case url == "https://fapi.binance.com/fapi/v1/premiumIndex":
			return []byte(`[{"symbol":"BTCUSDT","markPrice":"100","indexPrice":"99.9","lastFundingRate":"0.0001"},{"symbol":"ETHUSDT","markPrice":"50","indexPrice":"50","lastFundingRate":"0.0003"}]`), nil
		case url == "https://fapi.binance.com/fapi/v1/fundingInfo":
			return []byte(`[{"symbol":"BTCUSDT","fundingIntervalHours":8},{"symbol":"ETHUSDT","fundingIntervalHours":4}]`), nil
		case url == "https://api.hyperliquid.xyz/info":
			return []byte(`[{"universe":[{"name":"BTC"}]},[{"funding":"0.0002","markPx":"100","oraclePx":"100"}]]`), nil
		default:
			return []byte(`{"result":{"list":[{"symbol":"BTCUSDT","fundingRate":"0.0001","markPrice":"100","indexPrice":"100","fundingIntervalHour":"8"}]}}`), nil
		}
	}
	if err := r.collectAll(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM md_venue_funding
		WHERE venue IN ('binance-usdm','bybit','hyperliquid')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 3 {
		t.Fatalf("三 venue 采集落行应 ≥3: %d", n)
	}
	// fundingInfo join 归真：ETHUSDT interval_h=4（非 premiumIndex 携带）
	var iv float64
	if err := r.pool.QueryRow(ctx, `SELECT interval_h::float8 FROM md_venue_funding
		WHERE venue='binance-usdm' AND inst='ETHUSDT' ORDER BY ts DESC LIMIT 1`).Scan(&iv); err != nil {
		t.Fatal(err)
	}
	if iv != 4 {
		t.Fatalf("fundingInfo join 应归真 interval_h=4: %v", iv)
	}
}

// 锚②推荐闸：达阈 venue→state=recommended+last_request_at 戳；30d 内不重发。
func TestRecommendOnce(t *testing.T) {
	ctx, r := setup(t)
	base := time.Now()
	r.now = func() time.Time { return base }
	// binance 行：高 funding（净 APR 远超 Earn+margin）
	for i := 0; i < 40; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_venue_funding(ts,venue,inst,funding_rate)
			VALUES($1,'binance-usdm','BTCUSDT',0.01)`,
			time.Now().Add(-time.Duration(i*8)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	// margin 调低触发（persist=采集行天数占比——40 行 8h 间隔≈13.3 天 <30d 占比 0.44；
	// persist_ratio 改 0.4 触发；margin 键 0 → baseline+0 即可达阈）
	for k, v := range map[string]string{"scout.persist_ratio": "0.4", "scout.recommend_margin_apr": "0.0001"} {
		if _, err := r.pool.Exec(ctx,
			"UPDATE app_config SET value=to_jsonb($2::text) WHERE key=$1", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.measure(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	var lr *time.Time
	if err := r.pool.QueryRow(ctx, `SELECT state,last_request_at FROM fin_venue_candidate
		WHERE venue='binance-usdm'`).Scan(&state, &lr); err != nil {
		t.Fatal(err)
	}
	if state != "recommended" || lr == nil {
		t.Fatalf("达阈应 recommended+请求戳: %s/%v", state, lr)
	}
	// 30d 内重拍不重置 last_request_at（不重发闸）
	if _, err := r.pool.Exec(ctx, `UPDATE fin_venue_candidate SET state='measuring'
		WHERE venue='binance-usdm'`); err != nil {
		t.Fatal(err)
	}
	if err := r.measure(ctx); err != nil {
		t.Fatal(err)
	}
	var lr2 *time.Time
	var st2 string
	if err := r.pool.QueryRow(ctx, `SELECT state,last_request_at FROM fin_venue_candidate
		WHERE venue='binance-usdm'`).Scan(&st2, &lr2); err != nil {
		t.Fatal(err)
	}
	if st2 == "recommended" || !lr2.Equal(*lr) {
		t.Fatalf("30d 内不得重发（state=%s ts=%v→%v）", st2, lr, lr2)
	}
}

// 锚③KYC 不可及 venue 永不 recommended（业主地区禁线）。
func TestKycBlocked(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `UPDATE fin_venue_candidate SET kyc_ok=false
		WHERE venue='bybit'`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_venue_funding(ts,venue,inst,funding_rate)
			VALUES($1,'bybit','BTCUSDT',0.05)`,
			time.Now().Add(-time.Duration(i*8)*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.measure(ctx); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_venue_candidate
		WHERE venue='bybit'`).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state == "recommended" {
		t.Fatal("KYC 不可及 venue 不得 recommended")
	}
}

// 锚④canon 归一（返修①）：剥方言→base ccy——1000PEPE 类数量前缀同归。
func TestCanonNormalize(t *testing.T) {
	cases := map[string]string{
		"BTCUSDT": "BTC", "BTC-USDT-SWAP": "BTC", "BTCUSD": "BTC",
		"ETH-USDT-SWAP": "ETH", "SOLUSDT": "SOL",
		"1000PEPEUSDT": "PEPE", "1000PEPE-USDT-SWAP": "PEPE",
		"BTC_USDT_PERP": "BTC", "ETHUSDC": "ETH",
	}
	for in, want := range cases {
		if got := canon(in); got != want {
			t.Fatalf("canon(%q)=%q want %q", in, got, want)
		}
	}
}

// 锚⑤settles 分所归真（T-0304）：HL interval=1→24 次/日；binance interval=0→
// config 回退=3。同 rate 下 HL carry 净应≈8×binance（24/3）。
func TestSettlesPerVenue(t *testing.T) {
	ctx, r := setup(t)
	for _, inst := range []struct {
		v  string
		iv *float64
	}{{"hyperliquid", fp(1)}, {"binance-usdm", nil}} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_venue_funding(ts,venue,inst,funding_rate,interval_h)
			VALUES(now(),$1,'BTCUSDT',0.001,$2)`, inst.v, inst.iv); err != nil {
			t.Fatal(err)
		}
	}
	p, err := r.scoutParams()
	if err != nil {
		t.Fatal(err)
	}
	mid := func(venue string) float64 {
		rows, err := loadVenue(ctx, r, venue)
		if err != nil {
			t.Fatal(err)
		}
		settles := func(iv float64) float64 {
			if iv > 0 {
				return 24 / iv
			}
			return p.SettlesPerDay
		}
		ser := dailySeries(rows, 5, func(rate, iv float64) float64 {
			return abs(rate)*settles(iv)*365 - funding.CostAPR(p)
		})
		return med(ser)
	}
	hl, bn := mid("hyperliquid"), mid("binance-usdm")
	if !(hl > 7*bn && bn > 0) {
		t.Fatalf("HL(24x) 应≈8×binance(3x): hl=%v bn=%v", hl, bn)
	}
}
func fp(v float64) *float64 { return &v }

// 锚⑥跨所差式分列：同符号各带本所结算频次（HL 小时率 vs OKX 8h 率不比同×3）。
func TestXvenueSplitSettles(t *testing.T) {
	ctx, r := setup(t)
	// HL 小时结 rate=0.001×24=0.024/日；OKX 8h rate=0.002×3=0.006/日
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_venue_funding(ts,venue,inst,funding_rate,interval_h)
		VALUES(now(),'hyperliquid','BTCUSDT',0.001,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
		VALUES(now(),'okx','BTC-USDT-SWAP',0.002)`); err != nil {
		t.Fatal(err)
	}
	p, err := r.scoutParams()
	if err != nil {
		t.Fatal(err)
	}
	boards, err := r.boardFor(ctx, "hyperliquid", 5, p, 0)
	if err != nil {
		t.Fatal(err)
	}
	var xb *Board
	for i := range boards {
		if boards[i].Family == "xvenue_spread" {
			xb = &boards[i]
		}
	}
	if xb == nil {
		t.Fatal("xvenue_spread 榜行缺")
	}
	// 手算：|0.001×24−0.002×3|×365=|0.024−0.006|×365=6.57−双腿费年化
	// takerA(HL 台账)=4.5 + takerB(OKX 台账)=5 →costRt=2*(4.5+5+2*slip)+fail+window
	want := 0.018 * 365
	if xb.NetAprMid <= 0 || xb.NetAprMid > want {
		t.Fatalf("差式分列净 APR 应∈(0,%.1f]: %v", want, xb.NetAprMid)
	}
	if xb.BaselineSource != "earn" {
		t.Fatalf("xvenue 基线应 earn: %s", xb.BaselineSource)
	}
}

// 锚 T-0313：六 adapter 桩（url-switch stub 照三先例）+canon 新方言用例。
func adapterFor(t *testing.T, venue string) func(context.Context, Fetcher) ([]row, error) {
	t.Helper()
	fn, ok := adapters[venue]
	if !ok {
		t.Fatalf("adapter %s 未注册", venue)
	}
	return fn
}

func TestSixAdapters(t *testing.T) {
	ctx := context.Background()
	mk := func(_ string) Fetcher {
		return func(ctx context.Context, u string, _ string) ([]byte, error) {
			switch {
			case strings.Contains(u, "bitget"):
				return []byte(`{"code":"00000","data":[{"symbol":"BTCUSDT","markPrice":"77000","indexPrice":"77010","fundingRate":"0.0001"},{"symbol":"ETHUSDT","markPrice":"2300","indexPrice":"2301","fundingRate":"-0.0001"}]}`), nil
			case strings.Contains(u, "gateio"):
				return []byte(`[{"contract":"BTC_USDT","funding_rate":"0.0002","mark_price":"77000","index_price":"77010"}]`), nil
			case strings.Contains(u, "kucoin"):
				return []byte(`{"code":"200000","data":[{"symbol":"XBTUSDTM","markPrice":77000,"indexPrice":77010,"fundingFeeRate":0.0001,"currentFundingRateGranularity":28800000}]}`), nil
			case strings.Contains(u, "mexc"):
				return []byte(`{"data":[{"symbol":"BTC_USDT","fundingRate":0.0003,"fairPrice":77000,"indexPrice":77010}]}`), nil
			case strings.Contains(u, "dydx"):
				return []byte(`{"markets":{"BTC-USD":{"ticker":"BTC-USD","nextFundingRate":"0.000007","oraclePrice":"77000"}}}`), nil
			case strings.Contains(u, "get_instruments"):
				return []byte(`{"result":[{"instrument_name":"BTC-PERPETUAL"},{"instrument_name":"BTC-4NOV26"}]}`), nil
			case strings.Contains(u, "deribit.com/api/v2/public/ticker"):
				return []byte(`{"result":{"funding_8h":0.00008,"mark_price":77000,"index_price":77010}}`), nil
			}
			return nil, fmt.Errorf("未桩端点: %s", u)
		}
	}
	for _, v := range []struct {
		venue string
		want  int
	}{
		{"bitget", 2}, {"gate", 1}, {"kucoin-futures", 1},
		{"mexc", 1}, {"dydx-v4", 1}, {"deribit", 6},
	} {
		got, err := adapterFor(t, v.venue)(ctx, mk(v.venue))
		if err != nil {
			t.Fatalf("%s: %v", v.venue, err)
		}
		if len(got) != v.want {
			t.Fatalf("%s 行数: %d", v.venue, len(got))
		}
	}
	// canon 新方言用例（PERPETUAL 尾/M 尾/XBT 别名——票面钉）。
	for _, c := range [][2]string{
		{"BTC-PERPETUAL", "BTC"}, {"XBTUSDTM", "BTC"}, {"ETH-PERPETUAL", "ETH"},
		// 返修① B2 四回归：复合尾剥+裸 M 保（ATOM/DYM 不误伤）
		{"ATOM", "ATOM"}, {"DYM", "DYM"},
	} {
		if g := canon(c[0]); g != c[1] {
			t.Fatalf("canon %s → %s（want %s）", c[0], g, c[1])
		}
	}
}
