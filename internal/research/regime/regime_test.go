package regime

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// migrateBoth 测试面跨域（md_klines/fin_opp_*/ops_event/app_config——
// 两链并集态 md 先 fin 后=历史重放方向，gridfit 测试同款）。
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
	return Params{ScanIv: 6 * time.Hour, Lookback: 20, ZThr: 2.0,
		VrThr: 0.9, ContThr: 0.6, QuietThr: 0.01, VolThr: 0.03, ThinDays: 7}
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool, regMap string) {
	t.Helper()
	for k, v := range map[string]string{
		"research.regime.scan_interval_min": "360",
		"research.regime.lookback_days":     "20",
		"research.regime.z_thr":             "2.0",
		"research.regime.vr_thr":            "0.9",
		"research.regime.cont_thr":          "0.6",
		"research.regime.quiet_thr":         "0.01",
		"research.regime.vol_thr":           "0.03",
		"research.regime.thin_days":         "7",
	} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO app_config(key,value) VALUES('research.regime.map',$1::jsonb) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value",
		regMap); err != nil {
		t.Fatal(err)
	}
}

// klineSpec 显式 OHLC 棒（网格同款——分类实吃 c，h/l 备真形）。
type klineSpec struct{ o, h, l, c float64 }

// seedBars 单 inst 日棒种子（confirm=true；末棒 ts=now()-24h 鲜于
// stale_h=56h 懒读界）。
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

// chopBars 高波快震荡正弦（周期 3 日、幅 4%——vr<vr_thr∧含带率≥cont_thr
// 双条件实证形：慢正弦 vr>1 不入 chop，快周期收益交替→均值回复）。
func chopBars(n int) []klineSpec {
	out := make([]klineSpec, n)
	for i := 0; i < n; i++ {
		c := 100 * (1 + 0.04*math.Sin(2*math.Pi*float64(i)/3))
		pc := 100 * (1 + 0.04*math.Sin(2*math.Pi*float64(i-1)/3))
		if i == 0 {
			pc = c
		}
		out[i] = klineSpec{o: pc, h: math.Max(pc, c) * 1.005, l: math.Min(pc, c) * 0.995, c: c}
	}
	return out
}

// trendBars 单边上行+小噪（日漂移 +2%+±0.3% 摆动——|z|≫z_thr 实证形）。
func trendBars(n int) []klineSpec {
	out := make([]klineSpec, n)
	for i := 0; i < n; i++ {
		c := 100 * math.Pow(1.02, float64(i)) * (1 + 0.003*math.Sin(float64(i)*1.7))
		pc := 100 * math.Pow(1.02, float64(i-1)) * (1 + 0.003*math.Sin(float64(i-1)*1.7))
		if i == 0 {
			pc = c
		}
		out[i] = klineSpec{o: pc, h: math.Max(pc, c) * 1.005, l: math.Min(pc, c) * 0.995, c: c}
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

// TestClassifyChop 锚①：快震荡正弦→chop（vr∧cont 双条件+判词序先中）。
func TestClassifyChop(t *testing.T) {
	p := testParams()
	label, f := classify(toKlines(chopBars(p.Lookback+1)), p)
	if label != "chop" {
		t.Fatalf("快正弦判词=%s 期 chop（z=%.2f vr=%.3f cont=%.2f sigma=%.4f）",
			label, f.z, f.vr, f.cont, f.sigma)
	}
}

// TestClassifyTrend 锚②：单边上行→trend（|z|≥z_thr 首中）。
func TestClassifyTrend(t *testing.T) {
	p := testParams()
	label, f := classify(toKlines(trendBars(p.Lookback+1)), p)
	if label != "trend" {
		t.Fatalf("上行判词=%s 期 trend（z=%.2f）", label, f.z)
	}
}

// TestScanProposeOnceSuppress 锚③：全宇宙 chop dominant+映射缺族
// （ghost 不在册）→族提案升件一次；二拍销抑在案仍一行（未销不重复）。
// 态账行每拍落占比证据。
func TestScanProposeOnceSuppress(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool, `{"chop":["ghost_regime_family"],"trend":[],"quiet":[],"volatile":[]}`)
	for _, inst := range []string{"CHOP-A-USDT", "CHOP-B-USDT", "CHOP-C-USDT"} {
		for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
				t.Fatal(err)
			}
		}
	}
	// ops_event 无 inst_id 列——清域按 seat/event（测试 schema 隔离已
	// 包级，残留行不撞 scope 键：esc_scope='regime_map_gap' 须清否则
	// 前测残留族提案触发销抑）。
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='regime_map_gap'`); err != nil {
		t.Fatal(err)
	}
	p := testParams()
	for _, inst := range []string{"CHOP-A-USDT", "CHOP-B-USDT", "CHOP-C-USDT"} {
		seedBars(t, ctx, pool, "okx", inst, chopBars(p.Lookback+1))
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg).AttachEscPool(pool)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// 二拍：未销族提案在案→销抑不重复升件。
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='族提案' AND esc_scope='regime_map_gap'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("族提案行数=%v err=%v 期 1（一次升件二拍销抑）", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='态账' AND seat='态评估'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("态账行数=%v err=%v 期 2（每拍落证据）", n, err)
	}
	// 态账证据面：chop dominant 占比在案。
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='态账' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(note, "dominant=chop") || !strings.Contains(note, "chop=1.00") {
		t.Fatalf("态账 note=%q 期 dominant=chop+chop=1.00", note)
	}
}

// TestScanCoveredNoPropose 锚④：trend dominant+映射族在册且近窗卡数
// 足（≥thin_days）→覆盖足零提案；态账照落。
func TestScanCoveredNoPropose(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool, `{"chop":[],"trend":["dir_perp_mom"],"quiet":[],"volatile":[]}`)
	for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id LIKE 'TREND-%'"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `DELETE FROM ops_event WHERE esc_scope='regime_map_gap'`); err != nil {
		t.Fatal(err)
	}
	// dir_perp_mom 在册（md_1095）——补近窗 7 卡达稀薄线。
	for i := 0; i < 7; i++ {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,gross_apr,net_apr,verdict,legs,evidence)
			VALUES('TREND-SEED','dir_perp_mom','long_perp',0.1,0.08,'proposed','["perp_long"]','{}')`); err != nil {
			t.Fatal(err)
		}
	}
	p := testParams()
	for _, inst := range []string{"TREND-A-USDT", "TREND-B-USDT"} {
		seedBars(t, ctx, pool, "okx", inst, trendBars(p.Lookback+1))
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := NewRunner(pool, cfg).AttachEscPool(pool).ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='族提案' AND esc_scope='regime_map_gap'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("族提案行数=%v err=%v 期 0（覆盖足零提案）", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='态账' AND seat='态评估'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("态账行数=%v err=%v 期 1", n, err)
	}
}
