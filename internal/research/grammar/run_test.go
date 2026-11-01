// run_test.go —— 文法枚举锚（T-0120）：幂等结构键不重生+面关=族不生+
// 缺 executor 族假想卡自动落。
package grammar

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

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
	if nH < 2 {
		t.Fatalf("缺 executor 假想卡应 ≥2 族（lending/earn——T-0215/0216 executor 落位后 vol_sell/quarter_basis 族不再产卡；basis_short 面关不枚举）: %d", nH)
	}
}

// TestProposedAprGate T-0205 桩锚（重飞·列级口径）：proposed 无 APR 估计→
// 写径降级 unevaluable+apr_est_missing；估计在场→net_apr 列同拍非 NULL
// （首飞病灶=闸键面在场而列恒 NULL——闸与不变量同维）；键在值非法
// （nil/类型不识）=缺席同降级（残留窄洞并堵）。
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
	if err := r.writeEval(ctx, "ETH-USDT-261225", "quarter_basis", "short_perp_long_spot",
		"proposed", "", map[string]any{"net_apr": 0.12, "apr": 0.12}); err != nil {
		t.Fatal(err)
	}
	var colApr *float64
	if err := r.pool.QueryRow(ctx, `SELECT net_apr::float8 FROM fin_opp_eval
		WHERE inst_id='ETH-USDT-261225' AND template='quarter_basis'`).Scan(&colApr); err != nil {
		t.Fatal(err)
	}
	if colApr == nil || *colApr != 0.12 {
		t.Fatalf("proposed 行 net_apr 列应同拍落 0.12: %v", colApr)
	}
	if err := r.writeEval(ctx, "SOL-USDT-261225", "quarter_basis", "short_perp_long_spot",
		"proposed", "", map[string]any{"apr": nil}); err != nil {
		t.Fatal(err)
	}
	if err := r.writeEval(ctx, "XRP-USDT-261225", "quarter_basis", "short_perp_long_spot",
		"proposed", "", map[string]any{"net_apr": "0.3"}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE template='quarter_basis' AND verdict='unevaluable'
		AND evidence->>'reason'='apr_est_missing'
		AND inst_id IN ('SOL-USDT-261225','XRP-USDT-261225')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("非法值 proposed 应双双降级 unevaluable+apr_est_missing: %d", n)
	}
}

// TestProposedNullAprDBGate T-0205 重飞 DB 闸锚（fin_0094）：proposed∧
// net_apr NULL 直插=机器拒——写径旁路（gapscan 类）结构性封死。
func TestProposedNullAprDBGate(t *testing.T) {
	ctx, r := setup(t)
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('GATE-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','[]','{}')`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.ConstraintName != "fin_opp_eval_proposed_net_apr" {
		t.Fatalf("proposed∧NULL 直插应被 fin_0094 CHECK 拒: %v", err)
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
	var colApr *float64
	if err := r.pool.QueryRow(ctx, `SELECT verdict, (evidence->>'net_apr')::float8,
		(evidence->>'basis_pct')::float8, net_apr::float8 FROM fin_opp_eval
		WHERE inst_id='BTC-USD-270225' ORDER BY id DESC LIMIT 1`).Scan(&verdict, &apr, &basisPct, &colApr); err != nil {
		t.Fatal("proposed 行未落:", err)
	}
	if verdict != "proposed" || apr <= 0.05 {
		t.Fatalf("大额基差短窗应 proposed 且 net_apr>5%%: %s apr=%v", verdict, apr)
	}
	if colApr == nil || *colApr != apr { // T-0205 重飞：列与 evidence 同拍同值
		t.Fatalf("proposed 行 net_apr 列应同拍落列: %v", colApr)
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

// 锚⑤：vol-sell 实评（T-0215）——single_call 单腿 proposed 提案落+strangle
// C×P 同 uly/expiry 配对复合键提案（executor_missing 钉已撤）。
func TestEvalVolSellReal(t *testing.T) {
	ctx, r := setup(t)
	r.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"BTC": 2.0, "USDT": 100000}, nil
	})
	exp := time.Now().Add(20 * 24 * time.Hour)
	ins := func(pid, dir string, strike, apr float64, qc string) {
		raw := []byte(`{"quoteCcy":"` + qc + `","notionalCcy":"` + qc + `"}`)
		if _, err := r.pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,
			direction,strike,expiry,annualized_yield,min_size,max_size,raw)
			VALUES (now(),'okx','dcd','BTC-USD',$1,$2,$3,$4,$5,0.01,10,$6)`,
			pid, dir, strike, exp, apr, raw); err != nil {
			t.Fatal(err)
		}
	}
	ins("BTC-C-150", "C", 150000, 0.20, "BTC")
	ins("BTC-P-80", "P", 80000, 0.18, "USDT")
	ins("BTC-P-60", "P", 60000, 0.30, "USDT") // 同 uly/expiry 第二 P 腿——取高 APR
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(inst_id,last,ts) VALUES ('BTC-USDT',100000,now())`); err != nil {
		t.Fatal(err)
	}
	for _, f := range Families() {
		if f.Template == "vol_sell_single_call" || f.Template == "vol_sell_strangle" {
			if err := r.evalVolSell(ctx, f); err != nil {
				t.Fatal(err)
			}
		}
	}
	var verdict, reason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='BTC-C-150' AND template='vol_sell_single_call'`).Scan(&verdict, &reason); err != nil {
		t.Fatalf("single_call eval 缺: %v", err)
	}
	if verdict != "proposed" {
		t.Fatalf("single_call 应 proposed: %s/%s", verdict, reason)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='BTC-C-150' AND state='fin_proposed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("single_call 提案应 1 行: %d %v", n, err)
	}
	var sverdict, sreason string
	if err := r.pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id LIKE 'BTC-C-150+BTC-P-%' AND template='vol_sell_strangle'`).Scan(&sverdict, &sreason); err != nil {
		t.Fatalf("strangle eval 缺: %v", err)
	}
	if sverdict != "proposed" {
		t.Fatalf("strangle 应 proposed: %s/%s", sverdict, sreason)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='BTC-C-150+BTC-P-60' AND state='fin_proposed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("strangle 提案应复合键高 APR P 腿: %d %v", n, err)
	}
}
