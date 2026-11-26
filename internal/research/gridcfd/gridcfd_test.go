package gridcfd

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

// migrateBoth 测试面跨域（md_klines/md_ticks md 域表+fin 判据键+
// direction 词表+fin_opp_template md 对偶册行）——两链并集态
// （md 先 fin 后=历史重放方向，gridfit 测试同款）。
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
// 判据结构同 prod，值随测试窗缩；FeeRT=仿真注入槽 scan 逐 inst 覆写）。
func testParams() Params {
	return Params{ScanIv: 4 * time.Hour, Lookback: 20, BandPct: 0.15,
		StepPct: 0.01, LineNotional: 50, FeeFloor: 0.002,
		MinNetAPR: 0.05, BlownTol: 0.05}
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for k, v := range map[string]string{
		"research.gridcfd.scan_interval_min": "240",
		"research.gridcfd.lookback_days":     "20",
		"research.gridcfd.band_pct":          "0.15",
		"research.gridcfd.step_pct":          "0.01",
		"research.gridcfd.line_notional_usd": "50",
		"research.gridcfd.fee_floor":         "0.002",
		"research.gridcfd.min_net_apr":       "0.05",
		"research.gridcfd.blown_tol":         "0.05",
		"research.gridcfd.stale_h":           "56",
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
// 鲜于 stale_h=56h 界；venue 参数化——宇宙谓词断言面）。
func seedBars(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	venue, inst string, specs []klineSpec) {
	t.Helper()
	base := time.Now().Add(-time.Duration(len(specs)) * 24 * time.Hour)
	for i, s := range specs {
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES($1,$2,'1D',$3,$4,$5,$6,$7,1,1,true)`,
			venue, inst, ts, s.o, s.h, s.l, s.c); err != nil {
			t.Fatal(err)
		}
	}
}

// seedTick 单拍 md_ticks 行（venue 参数化；bid/ask 给定→spread=
// (ask-bid)/bid 实测面——PK (venue,inst_id) 一 inst 一行）。
func seedTick(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	venue, inst string, bid, ask float64) {
	t.Helper()
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,bid,ask)
		VALUES(now(),$1,$2,$3,$4)
		ON CONFLICT (venue,inst_id) DO UPDATE SET ts=now(),bid=$3,ask=$4`,
		venue, inst, bid, ask); err != nil {
		t.Fatal(err)
	}
}

// sinusoidBars 正弦价路（周期 8 日、幅 5%——带内震荡触发多轮买卖）。
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

// TestEvalSinusoid 纯仿真锚：带内震荡→多轮买卖 realized>0∧proposed
// （FeeRT=0.001 注入=cost_rt=0.002 floor 主导档同形）。
func TestEvalSinusoid(t *testing.T) {
	p := testParams()
	p.FeeRT = 0.001
	c := evalGrid("TESTGRIDm", toKlines(sinusoidBars(p.Lookback+1)),
		time.Now(), p)
	if c.verdict != "proposed" {
		t.Fatalf("正弦判词=%s reason=%s net_apr=%.4f 期 proposed", c.verdict, c.reason, c.netAPR)
	}
	if c.realized <= 0 || c.nSells == 0 {
		t.Fatalf("正弦 realized=%.4f n_sells=%d 期 >0∧>0", c.realized, c.nSells)
	}
}

// TestEvalBreakdown 纯仿真锚②：单边跌穿带→rejected（band_blown|net_thin）。
func TestEvalBreakdown(t *testing.T) {
	p := testParams()
	p.FeeRT = 0.001
	c := evalGrid("TESTDOWNm", toKlines(breakdownBars(p.Lookback+1)),
		time.Now(), p)
	if c.verdict != "rejected" {
		t.Fatalf("跌穿带判词=%s 期 rejected", c.verdict)
	}
	if c.reason != "band_blown" && c.reason != "net_thin" {
		t.Fatalf("跌穿带 reason=%s 期 band_blown|net_thin", c.reason)
	}
}

// TestScanE2E 四锚合流：①宇宙谓词只命中 venue='exness'（okx/xmglobal
// 同名/异名棒零评估）；②cost_rt=价差成本模型实测（floor 主导档
// cost_rt=0.002/fee_side=0.001；spread 主导档 cost_rt=0.02/fee_side
// =0.01→仿真 net_thin 自然拒）；ticks 全缺=fail-closed skip 零行；
// ③eval 行 template/direction='grid_cfd'+legs=["cfd_long"]；
// ④影子族闸=execfam 未册→proposed 翻 rejected/exec_family_missing
// （sim_verdict=proposed 原判留证据面）∧fin_proposal 零行。
func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	for _, tb := range []string{"md_ticks", "fin_opp_eval", "fin_proposal"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+tb); err != nil {
			t.Fatal(err)
		}
	}
	p := testParams()
	// ① 宇宙谓词：exness 三 inst（正弦+floor 主导价差 / 正弦+spread 主导
	// / 薄棒）+okx+xmglobal 各一——venue 谓词外全零评估。
	seedBars(t, ctx, pool, "exness", "TESTGRIDm", sinusoidBars(p.Lookback+1))
	seedBars(t, ctx, pool, "exness", "TESTWIDEm", sinusoidBars(p.Lookback+1))
	seedBars(t, ctx, pool, "exness", "TESTTHINm", sinusoidBars(10))
	seedBars(t, ctx, pool, "exness", "TESTNOTICKm", sinusoidBars(p.Lookback+1))
	seedBars(t, ctx, pool, "okx", "TESTOKX-USDT", sinusoidBars(p.Lookback+1))
	seedBars(t, ctx, pool, "xmglobal", "TESTXMMm", sinusoidBars(p.Lookback+1))
	// ② ticks：TESTGRIDm spread=0.001<floor→cost_rt=0.002；
	// TESTWIDEm spread=0.02>floor→cost_rt=0.02 费吃光网格毛利；
	// TESTOKX-USDT/xmglobal ticks 同种——venue 谓词外不入样本。
	seedTick(t, ctx, pool, "exness", "TESTGRIDm", 100, 100.1)
	seedTick(t, ctx, pool, "exness", "TESTWIDEm", 100, 102)
	seedTick(t, ctx, pool, "okx", "TESTOKX-USDT", 100, 100.1)
	seedTick(t, ctx, pool, "xmglobal", "TESTXMMm", 100, 100.1)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunnerVenue(pool, cfg, "exness", "grid_cfd", "网格评估-cfd-test")
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// ③④ TESTGRIDm：eval 行键形+影子闸翻转+价差成本对拍面。
	var v, reason, template, direction, simVerdict string
	var realized, spreadPct, costRT, feeSide float64
	if err := pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason',
		template, direction, evidence->>'sim_verdict',
		(evidence->>'realized')::float8, (evidence->>'spread_pct')::float8,
		(evidence->>'cost_rt')::float8, (evidence->>'fee_side')::float8
		FROM fin_opp_eval WHERE inst_id='TESTGRIDm'`).
		Scan(&v, &reason, &template, &direction, &simVerdict,
			&realized, &spreadPct, &costRT, &feeSide); err != nil {
		t.Fatalf("TESTGRIDm 评估行缺失: %v", err)
	}
	if template != "grid_cfd" || direction != "grid_cfd" {
		t.Fatalf("TESTGRIDm template/direction=%s/%s 期 grid_cfd/grid_cfd", template, direction)
	}
	if v != "rejected" || reason != "exec_family_missing" {
		t.Fatalf("TESTGRIDm 判词=%s/%s 期 rejected/exec_family_missing（影子族闸）", v, reason)
	}
	if simVerdict != "proposed" || realized <= 0 {
		t.Fatalf("TESTGRIDm sim_verdict=%s realized=%.4f 期 proposed∧>0（闸前原判留证据）", simVerdict, realized)
	}
	if math.Abs(spreadPct-0.001) > 1e-6 || math.Abs(costRT-0.002) > 1e-9 ||
		math.Abs(feeSide-0.001) > 1e-9 {
		t.Fatalf("TESTGRIDm 价差面 spread=%.5f cost_rt=%.5f fee_side=%.5f 期 0.001/0.002/0.001",
			spreadPct, costRT, feeSide)
	}
	// TESTWIDEm：spread 主导档 cost_rt=0.02→仿真自身 net_thin 拒
	// （spread≥step 经济性死经 net_apr 自然落——不另设判词键）。
	var simW string
	if err := pool.QueryRow(ctx, `SELECT verdict, evidence->>'reason',
		(evidence->>'spread_pct')::float8, (evidence->>'cost_rt')::float8,
		evidence->>'sim_verdict'
		FROM fin_opp_eval WHERE inst_id='TESTWIDEm'`).
		Scan(&v, &reason, &spreadPct, &costRT, &simW); err != nil {
		t.Fatalf("TESTWIDEm 评估行缺失: %v", err)
	}
	if v != "rejected" || reason != "net_thin" || simW != "rejected" {
		t.Fatalf("TESTWIDEm 判词=%s/%s sim=%s 期 rejected/net_thin/rejected（spread 主导费死=仿真自身拒非影子闸）", v, reason, simW)
	}
	if math.Abs(spreadPct-0.02) > 1e-6 || math.Abs(costRT-0.02) > 1e-9 {
		t.Fatalf("TESTWIDEm 价差面 spread=%.5f cost_rt=%.5f 期 0.02/0.02", spreadPct, costRT)
	}
	// 宇宙谓词+fail-closed：okx/xmglobal/薄棒/无 ticks 全零评估行。
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id IN ('TESTOKX-USDT','TESTXMMm','TESTTHINm','TESTNOTICKm')`).
		Scan(&n); err != nil || n != 0 {
		t.Fatalf("谓词外/薄棒/无ticks 评估行数=%v err=%v 期 0", n, err)
	}
	// 影子族闸：proposed 翻转后 fin_proposal 全零（执行面零开通）。
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("fin_proposal 行数=%v err=%v 期 0（exec_family_missing 闸后零提案）", n, err)
	}
	// 幂等：再拍 SameFingerprint 指纹同=不产新版史（行数仍 2）。
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id IN ('TESTGRIDm','TESTWIDEm')`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("二拍评估行数=%v err=%v 期 2（指纹闸幂等）", n, err)
	}
}
