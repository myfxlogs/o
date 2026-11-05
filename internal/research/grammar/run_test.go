// run_test.go —— 文法枚举锚（T-0120）：幂等结构键不重生+面关=族不生+
// 缺 executor 族假想卡自动落。
package grammar

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/research/dcd"
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

func TestDCDDirectoryFaceControlsVolSellEnumeration(t *testing.T) {
	for _, tc := range []struct {
		enabled bool
		want    int
	}{{false, 0}, {true, 2}} {
		families := Enumerate(map[string]bool{"okx/dcd": tc.enabled})
		got := 0
		for _, f := range families {
			if f.Template == "vol_sell_single_call" || f.Template == "vol_sell_strangle" {
				got++
			}
		}
		if got != tc.want {
			t.Fatalf("okx/dcd enabled=%t volsell families=%d want=%d", tc.enabled, got, tc.want)
		}
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

// T-0213 桩锚（T-0298 新口径重锚）：evalBasis 基差−往返摊销 vs 动态底——
// 高 APR 过阈→proposed；薄基差→rejected/basis_below_min；到期<7d→
// basis_expiry_soon 早退；深负基差有符号判→rejected（旧 math.Abs 病灶）。
func TestEvalBasisNetApr(t *testing.T) {
	ctx, r := setup(t)
	inst := func(d int) string { return "BTC-USD-" + time.Now().AddDate(0, 0, d).Format("060102") }
	inst10 := inst(10)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		(now(),'BTC-USD','index-tickers',100),
		(now(),$1,'mark-price',101),
		(now(),'ETH-USD','index-tickers',100),
		(now(),$2,'mark-price',101)`, inst10, "ETH-USD-"+time.Now().AddDate(0, 0, 30).Format("060102")); err != nil {
		t.Fatal(err)
	}
	f := Family{Template: "quarter_basis_long", Direction: "basis_quarter", Kind: "basis"}
	if err := r.evalBasis(ctx, f); err != nil {
		t.Fatal(err)
	}
	// basis=1%@~30d：gross≈0.1217−cost(32bps×365/30≈0.0396)≈0.082≥0.06→proposed
	var verdict string
	var apr, costApr float64
	var colApr *float64
	if err := r.pool.QueryRow(ctx, `SELECT verdict, (evidence->>'net_apr')::float8,
		(evidence->>'cost_apr')::float8, net_apr::float8 FROM fin_opp_eval
		WHERE inst_id=$1 ORDER BY id DESC LIMIT 1`,
		"ETH-USD-"+time.Now().AddDate(0, 0, 30).Format("060102")).Scan(&verdict, &apr, &costApr, &colApr); err != nil {
		t.Fatal("proposed 行未落:", err)
	}
	if verdict != "proposed" {
		t.Fatalf("1%%基差30d应 proposed: %s apr=%v", verdict, apr)
	}
	if colApr == nil || *colApr != apr {
		t.Fatalf("proposed 行 net_apr 列应同拍落列: %v", colApr)
	}
	wantCost := 32.0 / 1e4 * 365 / 30
	if math.Abs(costApr-wantCost) > 0.01 {
		t.Fatalf("cost_apr=%v want≈%v（32bps×365/30d 摊）", costApr, wantCost)
	}
	var v10, reason2 string
	if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id=$1 ORDER BY id DESC LIMIT 1`, inst10).Scan(&v10, &reason2); err != nil {
		t.Fatal("10d 行未落:", err)
	}
	if v10 != "proposed" {
		t.Fatalf("10d 1%%基差净≈0.24 应 proposed: %s/%q", v10, reason2)
	}
}

// T-0353 件①锚：4.84% 近确定基差落在 4% 中间档，应过阈；旧 6% 一刀切会错杀。
func TestEvalBasisMiddleBand(t *testing.T) {
	ctx, r := setup(t)
	inst := "BTC-USD-" + time.Now().AddDate(0, 0, 30).Format("060102")
	// 30d 基差取值使 net_apr≈4.84%：basis=(0.0032 + .0484*30/365)。
	fut := 100 * (1 + 0.0032 + 0.0484*30/365)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		(now(),'BTC-USD','index-tickers',100),(now(),$1,'mark-price',$2)`, inst, fut); err != nil {
		t.Fatal(err)
	}
	f := Family{Template: "quarter_basis_long", Direction: "basis_quarter", Kind: "basis"}
	if err := r.evalBasis(ctx, f); err != nil {
		t.Fatal(err)
	}
	var verdict, reason string
	var net float64
	if err := r.pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason',
		(evidence->>'net_apr')::float8 FROM fin_opp_eval WHERE inst_id=$1`, inst).
		Scan(&verdict, &reason, &net); err != nil {
		t.Fatal(err)
	}
	if net < 0.048 || net > 0.049 || verdict != "proposed" {
		t.Fatalf("4.84%% basis 应在 4%% 档 proposed；got verdict=%s reason=%s net=%v", verdict, reason, net)
	}
}

// T-0298 验收锚：days=10/30/90 三档 net_apr 手算对齐+days<7 expiry_soon 早退
// +深负基差有符号拒（旧 math.Abs 误判面）。
func TestEvalBasisCostAmortization(t *testing.T) {
	ctx, r := setup(t)
	day := func(d int) (string, float64) {
		s := time.Now().AddDate(0, 0, d).Format("060102")
		e, _ := time.Parse("060102", s)
		return s, time.Until(e).Hours() / 24
	}
	i10, _ := day(10)
	i30, _ := day(30)
	i90, d90 := day(90)
	i3, _ := day(3)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_mark_index(ts,inst_id,channel,px) VALUES
		(now(),'BTC-USD','index-tickers',100),
		(now(),'SOL-USD','index-tickers',100),
		(now(),'BTC-USD-'||$1,'mark-price',101),
		(now(),'BTC-USD-'||$2,'mark-price',101),
		(now(),'BTC-USD-'||$3,'mark-price',101),
		(now(),'BTC-USD-'||$4,'mark-price',101),
		(now(),'SOL-USD-'||$2,'mark-price',99)`, i10, i30, i90, i3); err != nil {
		t.Fatal(err)
	}
	f := Family{Template: "quarter_basis_long", Direction: "basis_quarter", Kind: "basis"}
	if err := r.evalBasis(ctx, f); err != nil {
		t.Fatal(err)
	}
	get := func(inst string) (string, string, float64, float64, float64) {
		var v, rs string
		var net, cost, gross float64
		if err := r.pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason',
			(evidence->>'net_apr')::float8, (evidence->>'cost_apr')::float8,
			(evidence->>'gross_apr')::float8 FROM fin_opp_eval
			WHERE inst_id=$1 ORDER BY id DESC LIMIT 1`, inst).Scan(&v, &rs, &net, &cost, &gross); err != nil {
			t.Fatalf("%s eval 行未落: %v", inst, err)
		}
		return v, rs, net, cost, gross
	}
	// 手算对齐：90d basis=1% → gross=1/d90*365/100；cost=32bps×365/d90；net=gross−cost
	v, rs, net, cost, gross := get("BTC-USD-" + i90)
	wantGross := 1.0 / d90 * 365 / 100
	wantCost := 32.0 / 1e4 * 365 / d90
	if math.Abs(gross-wantGross) > 0.001 || math.Abs(cost-wantCost) > 0.001 ||
		math.Abs(net-(gross-cost)) > 1e-6 {
		t.Fatalf("90d 手算错位: gross=%v(want %v) cost=%v(want %v) net=%v", gross, wantGross, cost, wantCost, net)
	}
	if v != "rejected" || rs != "basis_below_min" {
		t.Fatalf("90d 净≈0.028<0.06 应 basis_below_min: %s/%s", v, rs)
	}
	v, _, n30, _, _ := get("BTC-USD-" + i30)
	if v != "proposed" {
		t.Fatalf("30d 净≈0.082 应 proposed: %s", v)
	}
	v, _, n10, _, _ := get("BTC-USD-" + i10)
	if v != "proposed" {
		t.Fatalf("10d 净≈0.24 应 proposed: %s", v)
	}
	if !(n10 > n30 && n30 > net) {
		t.Fatalf("net_apr 非递减: 10d=%v 30d=%v 90d=%v", n10, n30, net)
	}
	// days<7=expiry_soon 早退（cost 摊销前）
	v, rs, _, _, _ = get("BTC-USD-" + i3)
	if v != "rejected" || rs != "basis_expiry_soon" {
		t.Fatalf("3d<7d 应 basis_expiry_soon: %s/%s", v, rs)
	}
	// 深负基差（fut<spot）有符号判→rejected（旧 math.Abs 会误判 proposed）
	v, rs, netN, _, _ := get("SOL-USD-" + i30)
	if v != "rejected" || netN >= 0 {
		t.Fatalf("深负基差应有符号 rejected: %s/%s net=%v", v, rs, netN)
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

// vol-sell 实评：非 USDT 担保 call/strangle 只落条件盒；USDT 担保 put 走提案常径。
func TestEvalVolSellReal(t *testing.T) {
	ctx, r := setup(t)
	r.AttachFin(r.pool)
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
		WHERE inst_id='BTC-C-150' AND state='fin_proposed'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("BTC 担保 call 不得 armed: %d %v", n, err)
	}
	var path string
	if err := r.pool.QueryRow(ctx, `SELECT evidence->>'proposal_path' FROM fin_opp_eval
		WHERE inst_id='BTC-C-150' AND template='vol_sell_single_call'`).Scan(&path); err != nil || path != "condition_gap_owner_domain" {
		t.Fatalf("BTC call eval 应标条件盒路径: %s %v", path, err)
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
		WHERE inst_id='BTC-C-150+BTC-P-60' AND state='fin_proposed'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("BTC 担保 strangle 不得 armed: %d %v", n, err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_condition_gap
		WHERE kind='holding' AND venue='okx' AND surface='dcd' AND state='pending'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("owner-domain 提案应去重落一个条件请求: %d %v", n, err)
	}
	// USDT 担保腿留基金常径：不落业主资产域条件请求，正常生成提案。
	put := dcd.Product{ProductID: "USDT-P-90000", Uly: "USDT-USD", Direction: "P", QuoteCcy: "USDT",
		Strike: 90000, Expiry: exp, AnnualizedYield: .18, MinSize: .01}
	pe := dcd.Evaluate(put, 100000, 100000, dcd.Params{MinDistPct: 8, MaxTermDays: 30, MinAPR: .06, NotionalPct: 20})
	if pe.Verdict != "proposed" {
		t.Fatalf("USDT put fixture should propose: %s/%s", pe.Verdict, pe.Reason)
	}
	if err := r.persistLegEval(ctx, pe, Family{Template: "vol_sell_strangle", Direction: "vol_sell"}, put.ProductID); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal WHERE inst_id='USDT-P-90000' AND state='fin_proposed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("USDT 担保腿应走基金提案常径: %d %v", n, err)
	}
}
