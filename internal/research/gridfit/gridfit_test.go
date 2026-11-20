package gridfit

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// migrateBoth 测试面跨域（md_klines/md 域表+fin 判据键+direction 词表）——
// 两链并集态（md 先 fin 后=历史重放方向，dirmom 测试同款）。
func migrateBoth(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
}

// testParams 判据=SSOT⑦钦值同形（lookback 缩至 20 日窗便于种子——
// 判据结构同 prod，值随测试窗缩）。
func testParams() Params {
	return Params{ScanIv: 4 * time.Hour, Lookback: 20, BandPct: 0.15,
		StepPct: 0.01, LineNotional: 50, FeeRT: 0.001,
		MinNetAPR: 0.05, BlownTol: 0.05}
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for k, v := range map[string]string{
		"research.gridfit.scan_interval_min": "240",
		"research.gridfit.lookback_days":     "20",
		"research.gridfit.band_pct":          "0.15",
		"research.gridfit.step_pct":          "0.01",
		"research.gridfit.line_notional_usd": "50",
		"research.gridfit.fee_rt":            "0.001",
		"research.gridfit.min_net_apr":       "0.05",
		"research.gridfit.blown_tol":         "0.05",
		"research.gridfit.stale_h":           "56",
	} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
}

// klineSpec 显式 OHLC 棒（o/h/l/c 全给——网格仿真吃 h/l 触发档）。
type klineSpec struct{ o, h, l, c float64 }

// seedBars 单 inst 日棒种子（confirm=true 收线棒；末棒 ts=now()-24h
// 鲜于 stale_h=56h 界）。
func seedBars(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst string, specs []klineSpec) {
	t.Helper()
	base := time.Now().Add(-time.Duration(len(specs)) * 24 * time.Hour)
	for i, s := range specs {
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES('okx',$1,'1D',$2,$3,$4,$5,$6,1,1,true)`,
			inst, ts, s.o, s.h, s.l, s.c); err != nil {
			t.Fatal(err)
		}
	}
}

// sinusoidBars 正弦价路（周期 8 日、幅 5%——带内震荡触发多轮买卖）。
// 每棒 o=前收、h/l=本棒区间±0.4% 影线。
func sinusoidBars(n int) []klineSpec {
	out := make([]klineSpec, n)
	for i := 0; i < n; i++ {
		c := 100 * (1 + 0.05*math.Sin(2*math.Pi*float64(i)/8))
		pc := 100 * (1 + 0.05*math.Sin(2*math.Pi*float64(i-1)/8))
		if i == 0 {
			pc = c
		}
		hi, lo := pc, pc
		if c > hi {
			hi = c
		}
		if c < lo {
			lo = c
		}
		out[i] = klineSpec{o: pc, h: hi * 1.004, l: lo * 0.996, c: c}
	}
	return out
}

// breakdownBars 单边跌穿带（每日 -3%——买线逐档接飞刀、末收<band_lo·0.95）。
func breakdownBars(n int) []klineSpec {
	out := make([]klineSpec, n)
	for i := 0; i < n; i++ {
		c := 100 * math.Pow(0.97, float64(i))
		pc := 100 * math.Pow(0.97, float64(i-1))
		if i == 0 {
			pc = c
		}
		out[i] = klineSpec{o: pc, h: math.Max(pc, c) * 1.002, l: math.Min(pc, c) * 0.998, c: c}
	}
	return out
}

func toKlines(specs []klineSpec) []kline {
	out := make([]kline, len(specs))
	for i, s := range specs {
		out[i] = kline{h: s.h, l: s.l, c: s.c}
	}
	return out
}

// TestEvalSinusoid 正弦锚①：带内震荡→多轮买卖 realized>0∧proposed
// （纯函数面——E2E 影子闸翻转另案）。
func TestEvalSinusoid(t *testing.T) {
	p := testParams()
	c := evalGrid("TESTGRID-USDT", toKlines(sinusoidBars(p.Lookback+1)),
		time.Now(), p)
	if c.verdict != "proposed" {
		t.Fatalf("正弦判词=%s reason=%s net_apr=%.4f 期 proposed", c.verdict, c.reason, c.netAPR)
	}
	if c.realized <= 0 {
		t.Fatalf("正弦 realized=%.4f 期 >0（n_sells=%d）", c.realized, c.nSells)
	}
	if c.nSells == 0 {
		t.Fatal("正弦 n_sells=0——仿真未发生卖出")
	}
}

// TestEvalBreakdown 单边跌穿带锚②：末收<band_lo(1-blown_tol)→
// rejected/band_blown（net_apr 深负同向证伪）。
func TestEvalBreakdown(t *testing.T) {
	p := testParams()
	c := evalGrid("TESTDOWN-USDT", toKlines(breakdownBars(p.Lookback+1)),
		time.Now(), p)
	if c.verdict != "rejected" {
		t.Fatalf("跌穿带判词=%s 期 rejected", c.verdict)
	}
	if c.reason != "band_blown" && c.reason != "net_thin" {
		t.Fatalf("跌穿带 reason=%s 期 band_blown|net_thin", c.reason)
	}
}

// TestScanE2E 三锚合流：正弦 inst→评估行落 rejected/exec_family_missing
// （影子闸）∧evidence.sim_verdict=proposed∧realized>0；跌穿带 inst→
// rejected 卡；薄棒 inst（<lookback+1）→skip 零行。
func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	for _, inst := range []string{"TESTGRID-USDT", "TESTDOWN-USDT", "TESTTHIN-USDT"} {
		for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
				t.Fatal(err)
			}
		}
	}
	p := testParams()
	seedBars(t, ctx, pool, "TESTGRID-USDT", sinusoidBars(p.Lookback+1))
	seedBars(t, ctx, pool, "TESTDOWN-USDT", breakdownBars(p.Lookback+1))
	seedBars(t, ctx, pool, "TESTTHIN-USDT", sinusoidBars(10)) // 10<21=lookback+1 薄棒
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := NewRunner(pool, cfg).ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// 正弦：影子闸后 verdict=rejected/exec_family_missing，evidence 留
	// sim_verdict=proposed+realized>0（卡证据面全可见执行零径）。
	var v, reason string
	var realized float64
	var simVerdict string
	if err := pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason',
		(evidence->>'realized')::float8, evidence->>'sim_verdict'
		FROM fin_opp_eval WHERE inst_id='TESTGRID-USDT'
		AND template='grid_spot' AND direction='grid_spot'`).
		Scan(&v, &reason, &realized, &simVerdict); err != nil {
		t.Fatalf("TESTGRID-USDT 评估行缺失: %v", err)
	}
	if v != "rejected" || reason != "exec_family_missing" {
		t.Fatalf("TESTGRID-USDT 判词=%s/%s 期 rejected/exec_family_missing（影子闸）", v, reason)
	}
	if simVerdict != "proposed" || realized <= 0 {
		t.Fatalf("TESTGRID-USDT sim_verdict=%s realized=%.4f 期 proposed∧>0", simVerdict, realized)
	}
	// 跌穿带：rejected 卡（band_blown|net_thin）。
	if err := pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='TESTDOWN-USDT' AND template='grid_spot' AND direction='grid_spot'`).
		Scan(&v, &reason); err != nil {
		t.Fatalf("TESTDOWN-USDT 评估行缺失: %v", err)
	}
	if v != "rejected" {
		t.Fatalf("TESTDOWN-USDT 判词=%s 期 rejected", v)
	}
	// 薄棒：skip 零评估行零提案。
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id='TESTTHIN-USDT'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("TESTTHIN-USDT 评估行数=%v err=%v 期 0（thin_bars skip）", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id IN ('TESTGRID-USDT','TESTDOWN-USDT','TESTTHIN-USDT')`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("提案行数=%v err=%v 期 0（影子族零提案径）", n, err)
	}
}
