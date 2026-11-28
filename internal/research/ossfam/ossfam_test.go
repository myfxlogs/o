package ossfam

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// migrateBoth 测试面跨域（md_funding/md_ticks/md_mark_index/md_klines
// md 域表+fin 判据键+direction 词表+fin_opp_template md 对偶册行）——
// 两链并集态（md 先 fin 后=历史重放方向，gridcfd 测试同款）。
func migrateBoth(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
}

func loadCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) *config.Client {
	t.Helper()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func evalCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, template string) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_opp_eval WHERE template=$1", template).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// ── funding-extreme-fade ─────────────────────────────────────────

func TestFundingFadeE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool, map[string]string{
		"research.funding_fade.scan_interval_min":      "360",
		"research.funding_fade.stale_h":                "26",
		"research.funding_fade.funding_pctile_gate":    "0.95",
		"research.funding_fade.friction_rt":            "0.00072",
		"research.funding_fade.est_trades_per_year":    "30",
		"research.funding_fade.target_net_annual":      "0.07",
		"research.funding_fade.periods_per_year":       "1095",
		"research.funding_fade.falsify_winrate_min":    "0.55",
		"research.funding_fade.falsify_net_annual_min": "0.05",
	})
	if _, err := pool.Exec(ctx, "DELETE FROM md_funding WHERE inst_id LIKE 'TESTFUND-%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE template='funding-extreme-fade'"); err != nil {
		t.Fatal(err)
	}
	// 宇宙：40 常态 inst（|rate|=1e-5）+2 极值（±1e-3）——p95 阈=1e-5，
	// 严格>阈只有两极值 inst 成信号；极值 gross=1e-3×1095≈1.07
	// net≈1.05≥0.07→proposed；常态 net=0.011−0.022<0→rejected。
	for i := 0; i < 40; i++ {
		inst := fmt.Sprintf("TESTFUND-N%02d-SWAP", i)
		if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
			VALUES(now(),'okx',$1,0.00001)`, inst); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		inst string
		rate float64
	}{
		{"TESTFUND-HI-SWAP", 0.001}, {"TESTFUND-LO-SWAP", -0.001},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,venue,inst_id,funding_rate)
			VALUES(now(),'okx',$1,$2)`, tc.inst, tc.rate); err != nil {
			t.Fatal(err)
		}
	}
	r := NewFundingRunner(pool, loadCfg(t, ctx, pool))
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := evalCount(t, ctx, pool, "funding-extreme-fade"); n != 42 {
		t.Fatalf("eval 行数=%d 期望 42（全宇宙覆盖）", n)
	}
	var verdict, direction, reason string
	if err := pool.QueryRow(ctx, `SELECT verdict,direction,
		COALESCE(evidence->>'reason','') FROM fin_opp_eval
		WHERE template='funding-extreme-fade' AND inst_id='TESTFUND-HI-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict, &direction, &reason); err != nil {
		t.Fatal(err)
	}
	if verdict != "proposed" || direction != "perp_fade" {
		t.Fatalf("极值多付租 inst verdict=%s direction=%s 期望 proposed/perp_fade", verdict, direction)
	}
	var fadeSide string
	if err := pool.QueryRow(ctx, `SELECT evidence->>'fade_side' FROM fin_opp_eval
		WHERE template='funding-extreme-fade' AND inst_id='TESTFUND-HI-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&fadeSide); err != nil {
		t.Fatal(err)
	}
	if fadeSide != "short_perp" {
		t.Fatalf("rate>0 fade_side=%s 期望 short_perp", fadeSide)
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE template='funding-extreme-fade' AND inst_id='TESTFUND-N00-SWAP'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict); err != nil {
		t.Fatal(err)
	}
	if verdict != "rejected" {
		t.Fatalf("常态 inst verdict=%s 期望 rejected", verdict)
	}
	// 指纹去重：同拍重扫 verdict+net 未迁→零新行。
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := evalCount(t, ctx, pool, "funding-extreme-fade"); n != 42 {
		t.Fatalf("重扫后 eval 行数=%d 期望 42（指纹闸去重）", n)
	}
	// 零提案零执行钦定径：fin_proposal direction='perp_fade' 恒 0。
	var propN int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE direction='perp_fade'").Scan(&propN); err != nil {
		t.Fatal(err)
	}
	if propN != 0 {
		t.Fatalf("fin_proposal perp_fade 行=%d 期望 0（影子零提案）", propN)
	}
}

// ── mark-last-disloc-fade ────────────────────────────────────────

func TestDislocFadeE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool, map[string]string{
		"research.disloc_fade.scan_interval_min": "240",
		"research.disloc_fade.stale_min":         "30",
		"research.disloc_fade.disloc_gate_bp":    "10",
		"research.disloc_fade.edge_bp_min":       "2",
		"research.disloc_fade.edge_bp_max":       "5",
		"research.disloc_fade.friction_rt":       "0.00072",
	})
	// 逐表精确清（TESTDISC- 前缀隔离——他包测试行不触）。
	for _, sql := range []string{
		"DELETE FROM md_ticks WHERE inst_id LIKE 'TESTDISC-%'",
		"DELETE FROM md_mark_index WHERE inst_id LIKE 'TESTDISC-%'",
		"DELETE FROM fin_opp_eval WHERE template='mark-last-disloc-fade'",
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	// TESTDISC-A：last=100 mark=100.15→disloc=15bp>10 闸；per_event_net=
	// 3.5−7.2=−3.7bp<0→rejected/edge_sub_cost（卡载结构判词）。
	// TESTDISC-B：last=100 mark=100.05→5bp<10→rejected/disloc_thin。
	// TESTDISC-C：仅 tick 无 mark→join 不入卡。
	for _, tc := range []struct {
		inst, last, mark string
		hasMark          bool
	}{
		{"TESTDISC-A-SWAP", "100", "100.15", true},
		{"TESTDISC-B-SWAP", "100", "100.05", true},
		{"TESTDISC-C-SWAP", "100", "", false},
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask)
			VALUES(now(),'okx',$1,$2::numeric,$2::numeric,$2::numeric)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=now(),last=$2::numeric`,
			tc.inst, tc.last); err != nil {
			t.Fatal(err)
		}
		if tc.hasMark {
			if _, err := pool.Exec(ctx, `INSERT INTO md_mark_index(ts,venue,inst_id,channel,px)
				VALUES(now(),'okx',$1,'mark-price',$2::numeric)
				ON CONFLICT (venue,inst_id,channel) DO UPDATE SET ts=now(),px=$2::numeric`,
				tc.inst, tc.mark); err != nil {
				t.Fatal(err)
			}
		}
	}
	r := NewDislocRunner(pool, loadCfg(t, ctx, pool))
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := evalCount(t, ctx, pool, "mark-last-disloc-fade"); n != 2 {
		t.Fatalf("eval 行数=%d 期望 2（mark 缺席 C 不入卡）", n)
	}
	var verdict, reason, fadeSide string
	var net float64
	if err := pool.QueryRow(ctx, `SELECT verdict,
		COALESCE(evidence->>'reason',''), evidence->>'fade_side', net_apr
		FROM fin_opp_eval WHERE template='mark-last-disloc-fade'
		AND inst_id='TESTDISC-A-SWAP' ORDER BY id DESC LIMIT 1`).
		Scan(&verdict, &reason, &fadeSide, &net); err != nil {
		t.Fatal(err)
	}
	if verdict != "rejected" || reason != "edge_sub_cost" {
		t.Fatalf("错位>闸 verdict=%s reason=%s 期望 rejected/edge_sub_cost", verdict, reason)
	}
	if fadeSide != "long_perp" {
		t.Fatalf("last<mark fade_side=%s 期望 long_perp（向 mark 收敛）", fadeSide)
	}
	if net != 0 {
		t.Fatalf("net_apr=%v 期望 0（event_rate 不可测结构判词如实空）", net)
	}
	if err := pool.QueryRow(ctx, `SELECT COALESCE(evidence->>'reason','')
		FROM fin_opp_eval WHERE template='mark-last-disloc-fade'
		AND inst_id='TESTDISC-B-SWAP' ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != "disloc_thin" {
		t.Fatalf("小错位 reason=%s 期望 disloc_thin", reason)
	}
}

// ── xsec-perp-mom-basket ─────────────────────────────────────────

// seedXsecBars 单 inst n 根 1D 确认棒——收盘从 c0 线性漂到 c1
// （末棒 ts=now-24h 在 stale 界内，旧棒逐日回推）。
func seedXsecBars(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst string, n int, c0, c1 float64) {
	t.Helper()
	base := time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	for i := 0; i < n; i++ {
		c := c0 + (c1-c0)*float64(i)/float64(n-1)
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES('okx',$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
			inst, ts, c); err != nil {
			t.Fatal(err)
		}
	}
}

func TestXsecMomE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool, map[string]string{
		"research.xsec_mom.scan_interval_min":        "360",
		"research.xsec_mom.lookback_days":            "10",
		"research.xsec_mom.stale_h":                  "56",
		"research.xsec_mom.basket_k":                 "2",
		"research.xsec_mom.rebal_per_year":           "12",
		"research.xsec_mom.friction_rt":              "0.00072",
		"research.xsec_mom.target_net_annual":        "0.15",
		"research.xsec_mom.falsify_sharpe_lb":        "1.0",
		"research.xsec_mom.falsify_gross_annual_min": "0.12",
		"research.xsec_mom.paper_window_d":           "90",
	})
	if _, err := pool.Exec(ctx, "DELETE FROM md_klines WHERE inst_id LIKE 'TESTXS-%'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "DELETE FROM fin_opp_eval WHERE template='xsec-perp-mom-basket'"); err != nil {
		t.Fatal(err)
	}
	r := NewXsecRunner(pool, loadCfg(t, ctx, pool))
	// ① 宇宙 3 inst<2k=4→thin_universe 拒行（仍落卡=评估覆盖）。
	for _, inst := range []string{"TESTXS-A-SWAP", "TESTXS-B-SWAP", "TESTXS-C-SWAP"} {
		seedXsecBars(t, ctx, pool, inst, 11, 100, 105)
	}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var verdict, reason string
	if err := pool.QueryRow(ctx, `SELECT verdict,COALESCE(evidence->>'reason','')
		FROM fin_opp_eval WHERE template='xsec-perp-mom-basket'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict, &reason); err != nil {
		t.Fatal(err)
	}
	if verdict != "rejected" || reason != "thin_universe" {
		t.Fatalf("薄宇宙 verdict=%s reason=%s 期望 rejected/thin_universe", verdict, reason)
	}
	// ② 补至 6 inst：TOP×2 +10%、BOT×2 −10%、MID×2 平——spread≈0.2
	// L日→gross≈0.2×36.5=7.3 net≫0.15→proposed。
	for _, inst := range []string{"TESTXS-D-SWAP", "TESTXS-E-SWAP", "TESTXS-F-SWAP"} {
		seedXsecBars(t, ctx, pool, inst, 11, 100, 105)
	}
	// 覆写 D/E/B/C 方向：A,B 已 100→105=top 候选；让 C,F 平、D,E 下跌。
	for _, tc := range []struct {
		inst string
		c1   float64
	}{
		{"TESTXS-C-SWAP", 100}, {"TESTXS-D-SWAP", 90}, {"TESTXS-E-SWAP", 90}, {"TESTXS-F-SWAP", 100},
	} {
		if _, err := pool.Exec(ctx, `DELETE FROM md_klines WHERE inst_id=$1`, tc.inst); err != nil {
			t.Fatal(err)
		}
		seedXsecBars(t, ctx, pool, tc.inst, 11, 100, tc.c1)
	}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var direction, inst string
	if err := pool.QueryRow(ctx, `SELECT verdict,direction,inst_id
		FROM fin_opp_eval WHERE template='xsec-perp-mom-basket'
		ORDER BY id DESC LIMIT 1`).Scan(&verdict, &direction, &inst); err != nil {
		t.Fatal(err)
	}
	if verdict != "proposed" || direction != "xsec_mom_basket" || inst != xsecBasketInst {
		t.Fatalf("满仓 verdict=%s direction=%s inst=%s 期望 proposed/xsec_mom_basket/%s",
			verdict, direction, inst, xsecBasketInst)
	}
	if n := evalCount(t, ctx, pool, "xsec-perp-mom-basket"); n != 2 {
		t.Fatalf("eval 行数=%d 期望 2（thin+proposed 两版史）", n)
	}
	// 零提案零执行：direction='xsec_mom_basket' fin_proposal 恒 0。
	var propN int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE direction='xsec_mom_basket'").Scan(&propN); err != nil {
		t.Fatal(err)
	}
	if propN != 0 {
		t.Fatalf("fin_proposal xsec_mom_basket 行=%d 期望 0", propN)
	}
}
