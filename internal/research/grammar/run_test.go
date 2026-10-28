// run_test.go —— 文法枚举锚（T-0120）：幂等结构键不重生+面关=族不生+
// 缺 executor 族假想卡自动落。
package grammar

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	// T-0167 分链：本包测试面跨域（研究域表+判据键）——两链并集态（md 先 fin 后
	// =历史重放方向：md 链瞬态超表层先落，fin 链状态化/守卫后到）。
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

// 锚①：文法枚举幂等——双拍同结果不重生行（UNIQUE 结构键撞行幂等）。
func TestEnumerateIdempotent(t *testing.T) {
	ctx, r := setup(t)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n1 int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE state='candidate'`).Scan(&n1); err != nil {
		t.Fatal(err)
	}
	if n1 == 0 {
		t.Fatal("文法枚举零候选——面在族不生")
	}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n2 int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE state='candidate'`).Scan(&n2); err != nil {
		t.Fatal(err)
	}
	if n1 != n2 {
		t.Fatalf("幂等违例: 首拍 %d 行次拍 %d 行", n1, n2)
	}
}

// 锚②：面关=族不生——mt5.catalog enabled=false 时 mt5_swap_carry 不落。
func TestDisabledFaceNoFamily(t *testing.T) {
	ctx, r := setup(t)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_template
		WHERE template='mt5_swap_carry' AND state='candidate'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("mt5 catalog 关态——mt5_swap_carry 候选不应落")
	}
	// 假想卡锚：缺 executor 族自动落 requirements.executor。
	var nH int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_hypothesis
		WHERE requirements->>'executor' IS NOT NULL`).Scan(&nH); err != nil {
		t.Fatal(err)
	}
	if nH < 3 {
		t.Fatalf("缺 executor 假想卡应 ≥3 族（lending/earn/quarter_basis/vol_sell）: %d", nH)
	}
}

// TestProposedAprGate T-0205 桩锚：proposed 无 APR 估计→写径降级
// unevaluable+apr_est_missing（NULL-proposed 不可能径断言）。
func TestProposedAprGate(t *testing.T) {
	ctx, r := setup(t)
	if err := r.writeEval(ctx, "BTC-USD-261225", "quarter_basis", "short_perp_long_spot",
		"proposed", "", map[string]any{"basis_pct": 0.9}); err != nil {
		t.Fatal(err)
	}
	var v, reason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason'
		FROM fin_opp_eval WHERE inst_id='BTC-USD-261225' AND template='quarter_basis'`).Scan(&v, &reason); err != nil {
		t.Fatal(err)
	}
	if v != "unevaluable" || reason != "apr_est_missing" {
		t.Fatalf("无 APR proposed 应降级 unevaluable+apr_est_missing: %v/%s", v, reason)
	}
}

// T-0213 桩锚：evalBasis net_apr 补发——fut/spot 行齐+基差超阈→proposed
// +evidence 带 net_apr；基差不足→rejected/basis_below_min。
func TestEvalBasisNetApr(t *testing.T) {
	ctx, r := setup(t)
	// fut=105 spot=100 inst 尾 361231（≈3y 到期→年化小；大额基差补短窗造高
	// APR：取 26-02-25 尾=~120d，基差 5%/120d×365≈15%APR 过阈）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		(now(),'BTC-USD-270225','mark-price',105),
		(now(),'BTC-USD','index-tickers',100),
		(now(),'ETH-USD-270225','mark-price',101),
		(now(),'ETH-USD','index-tickers',100)`); err != nil {
		t.Fatal(err)
	}
	f := Family{Template: "basis_ann", Direction: "gap_carry", Kind: "basis"}
	if err := r.evalBasis(ctx, f); err != nil {
		t.Fatal(err)
	}
	var verdict string
	var apr, basisPct float64
	if err := r.pool.QueryRow(ctx, `SELECT verdict, (evidence->>'net_apr')::float8,
		(evidence->>'basis_pct')::float8 FROM fin_opp_eval
		WHERE inst_id='BTC-USD-270225' ORDER BY id DESC LIMIT 1`).Scan(&verdict, &apr, &basisPct); err != nil {
		t.Fatal("proposed 行未落:", err)
	}
	if verdict != "proposed" || apr <= 0.05 {
		t.Fatalf("大额基差短窗应 proposed 且 net_apr>5%%: %s apr=%v", verdict, apr)
	}
	var verdict2, reason2 string
	var apr2 *float64
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason',
		(evidence->>'net_apr')::float8 FROM fin_opp_eval
		WHERE inst_id='ETH-USD-270225' ORDER BY id DESC LIMIT 1`).Scan(&verdict2, &reason2, &apr2); err != nil {
		t.Fatal("rejected 行未落:", err)
	}
	if verdict2 != "rejected" || reason2 != "basis_below_min" {
		t.Fatalf("小基差应 rejected/basis_below_min: %s/%s", verdict2, reason2)
	}
}

// T-0214 桩锚：md_earn_rate 有行→evalCarry 不落 savings_rate_missing。
func TestEvalCarryEarnRate(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_earn_rate(ts,venue,ccy,rate)
		VALUES(now(),'okx','USDT',0.035)`); err != nil {
		t.Fatal(err)
	}
	f := Family{Template: "lending_spread", Direction: "gap_carry", Kind: "carry"}
	if err := r.evalCarry(ctx, f); err != nil {
		t.Fatal(err)
	}
	var verdict, reason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE template='lending_spread' ORDER BY id DESC LIMIT 1`).Scan(&verdict, &reason); err != nil {
		t.Fatal("eval 行未落:", err)
	}
	if verdict == "unevaluable" || reason == "savings_rate_missing" {
		t.Fatalf("有行仍 missing: %s/%s", verdict, reason)
	}
}
