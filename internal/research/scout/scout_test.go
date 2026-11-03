package scout

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
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
			return []byte(`[{"symbol":"BTCUSDT","markPrice":"100","indexPrice":"99.9","lastFundingRate":"0.0001","fundingIntervalHours":8},{"symbol":"ETHUSDT","markPrice":"50","indexPrice":"50","lastFundingRate":"0.0003","fundingIntervalHours":8}]`), nil
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
