package funding

import (
	"context"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "5", "research.funding.slippage_bps": "3",
		"research.funding.fail_cost_bps": "2", "research.funding.window_loss_bps": "4",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "3"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_funding WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_proposal WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE inst_id LIKE 'TEST%'"); err != nil {
		t.Fatal(err)
	}
	// 鲜 funding 行：高率（应提案）+负率（应拒）
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TEST-USDT-SWAP',0.01),(now(),'TEST2-USDT-SWAP',-0.02)`); err != nil {
		t.Fatal(err)
	}
	// 陈行不评估
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now()-interval '40 hours','TEST3-USDT-SWAP',0.05)`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var verdict string
	var propN int
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='TEST-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("高率卡 verdict=%s err=%v", verdict, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id='TEST-USDT-SWAP'").Scan(&propN); err != nil || propN != 1 {
		t.Fatalf("提案行=%d err=%v", propN, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='TEST2-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&verdict); err != nil || verdict != "rejected" {
		t.Fatalf("负率卡 verdict=%s err=%v", verdict, err)
	}
	var n int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM fin_opp_eval WHERE inst_id='TEST3-USDT-SWAP'").Scan(&n)
	if n != 0 {
		t.Fatal("陈 funding 行被评估——新鲜度门失效")
	}
	// 判据缺位锚：删一键→scan 应 fail-closed
	if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key='research.funding.min_net_apr'"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err == nil {
		t.Fatal("判据缺位应 fail-closed")
	}
	if got := r.rej.Snapshot()["research/missing_params"]; got == 0 {
		t.Fatal("missing_params 门未计")
	}
}

// T-0023 动态底三态锚：below_earn 拒/高过 earn 过/缺失降级。
func TestDynamicFloor(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "5", "research.funding.slippage_bps": "3",
		"research.funding.fail_cost_bps": "2", "research.funding.window_loss_bps": "4",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "3",
		"research.earn.stale_hours": "24"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, "DELETE FROM md_funding WHERE inst_id LIKE 'DF%'"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	// earn=0.50 高于静态底 0.05→DF1 net 0.10 应 below_earn 拒
	if _, err := pool.Exec(ctx, "INSERT INTO md_earn_rate(venue,ccy,rate) VALUES('okx','USDT',0.5)"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'DF1-USDT-SWAP',0.00015),(now(),'DF2-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var v string
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='DF1-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&v); err != nil || v != "rejected" {
		t.Fatalf("below_earn 拒 verdict=%s", v)
	}
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='DF2-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&v); err != nil || v != "proposed" {
		t.Fatalf("高过 earn 应过 verdict=%s", v)
	}
	// 缺失→静态底降级+计数锚
	if _, err := pool.Exec(ctx, "DELETE FROM md_earn_rate WHERE ccy='USDT'"); err != nil {
		t.Fatal(err)
	}
	before := r.rej.Snapshot()["research/earn_stale"]
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'DF3-USDT-SWAP',0.00015)`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["research/earn_stale"] == before {
		t.Fatal("earn 缺失降级未计数")
	}
	if err := pool.QueryRow(ctx,
		"SELECT verdict FROM fin_opp_eval WHERE inst_id='DF3-USDT-SWAP' ORDER BY id DESC LIMIT 1").Scan(&v); err != nil {
		t.Fatal(err)
	}
	// 降级后静态底判：net 0.10>0.05→proposed（缺失降级不阻断）
	if v != "proposed" {
		t.Fatalf("降级静态底判 verdict=%s", v)
	}
}
