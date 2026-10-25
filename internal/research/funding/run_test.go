package funding

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
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

// TestScanEarnFloorE2E 动态底扫描径三态：Earn 现率新鲜行→below_earn 拒；行缺/陈→
// 静态底降级+计数（降级不阻断）。
func TestScanEarnFloorE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// net=0.06 锚定形判据（taker 1500+slip 50bps；rate 0.001×1 settle×365）
	for k, v := range map[string]string{
		"research.funding.taker_fee_bps": "1500", "research.funding.slippage_bps": "50",
		"research.funding.fail_cost_bps": "0", "research.funding.window_loss_bps": "0",
		"research.funding.min_net_apr": "0.05", "research.funding.settles_per_day": "1",
		"research.earn.stale_hours": "2"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cleanup := func() {
		for _, q := range []string{
			"DELETE FROM fin_proposal WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM fin_opp_eval WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM md_funding WHERE inst_id LIKE 'TESTE%'",
			"DELETE FROM md_earn_rate WHERE ccy='USDT'"} {
			_, _ = pool.Exec(ctx, q)
		}
	}
	cleanup()
	defer cleanup()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	// ①鲜 Earn 行 0.08（>net 0.06）→TESTE1 below_earn 拒；TESTE2 net≈10.9 过
	if _, err := pool.Exec(ctx, `INSERT INTO md_earn_rate(ts,venue,ccy,rate) VALUES(now(),'okx','USDT',0.08)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE1-USDT-SWAP',0.001),(now(),'TESTE2-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var reason string
	if err := pool.QueryRow(ctx, `SELECT evidence->>'reason' FROM fin_opp_eval WHERE inst_id='TESTE1-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil || len(reason) < 10 || reason[:10] != "below_earn" {
		t.Fatalf("between 态 reason=%q err=%v（应 below_earn）", reason, err)
	}
	var verdict string
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE2-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("高于 Earn 底 verdict=%s err=%v", verdict, err)
	}
	// ②Earn 行缺→静态底降级：TESTE3（同 0.001，net 0.06≥0.05）→proposed+计数
	if _, err := pool.Exec(ctx, "DELETE FROM md_earn_rate WHERE ccy='USDT'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE3-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	before := r.rej.Snapshot()["research/earn_rate_degraded"]
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["research/earn_rate_degraded"] <= before {
		t.Fatal("降级计数门未计")
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE3-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("降级态 verdict=%s err=%v（应静态底放行）", verdict, err)
	}
	// ③陈 Earn 行（5h>2h 门）→同样降级
	if _, err := pool.Exec(ctx, `INSERT INTO md_earn_rate(ts,venue,ccy,rate) VALUES
		(now()-interval '5 hours','okx','USDT',0.08)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate) VALUES
		(now(),'TESTE4-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	before = r.rej.Snapshot()["research/earn_rate_degraded"]
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if r.rej.Snapshot()["research/earn_rate_degraded"] <= before {
		t.Fatal("陈现率降级计数门未计")
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval WHERE inst_id='TESTE4-USDT-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil || verdict != "proposed" {
		t.Fatalf("陈现率降级 verdict=%s err=%v", verdict, err)
	}
}

// T-0029 macro 注解锚：近窗事件→evidence.macro 在；无事件→缺省；判据不受影响。
func TestMacroAnnotate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 种子锚：kind 三值齐+行数≥15
	var kinds, n int
	if err := pool.QueryRow(ctx,
		"SELECT count(DISTINCT kind),count(*) FROM md_macro_event").Scan(&kinds, &n); err != nil || kinds != 3 || n < 15 {
		t.Fatalf("种子行 kinds=%d n=%d", kinds, n)
	}
	// 注一个明日事件→evidence.macro 应在
	if _, err := pool.Exec(ctx,
		"INSERT INTO md_macro_event(event_date,kind,title) VALUES(current_date+1,'fomc','dbg-event') ON CONFLICT DO NOTHING"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	e := &Eval{InstID: "M-USDT-SWAP", Evidence: map[string]any{}}
	r.annotateMacro(ctx, e)
	if _, ok := e.Evidence["macro"]; !ok {
		t.Fatal("近窗事件 macro 注解缺席")
	}
	// 无近窗事件（horizon=0 域内）→不注解
	e2 := &Eval{InstID: "M2", Evidence: map[string]any{}}
	if _, err := pool.Exec(ctx,
		"UPDATE app_config SET value='\"0\"' WHERE key='research.macro.horizon_days'"); err != nil {
		t.Fatal(err)
	}
	_ = cfg.Load(ctx)
	r.annotateMacro(ctx, e2)
	if _, ok := e2.Evidence["macro"]; ok {
		t.Fatal("horizon=0 不应注解")
	}
	// 缺键 fail-open：删键不炸不注解
	if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key='research.macro.horizon_days'"); err != nil {
		t.Fatal(err)
	}
	_ = cfg.Load(ctx)
	e3 := &Eval{InstID: "M3", Evidence: map[string]any{}}
	r.annotateMacro(ctx, e3) // fail-open 无注解不炸
}
