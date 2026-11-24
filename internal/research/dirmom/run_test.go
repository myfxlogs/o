package dirmom

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// migrateBoth 测试面跨域（md_klines/md 域表+fin 判据键）——两链并集态
// （md 先 fin 后=历史重放方向，funding 测试同款）。
func migrateBoth(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		t.Fatal(err)
	}
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
}

func seedCfg(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	for k, v := range map[string]string{
		"research.dirmom.scan_interval_min": "240",
		"research.dirmom.lookback_days":     "20",
		"research.dirmom.z_entry":           "1.5",
		"research.dirmom.hold_days":         "10",
		"research.dirmom.persist":           "0.5",
		"research.dirmom.friction_rt":       "0.001",
		"research.dirmom.min_net_apr":       "0.05",
	} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
}

// seedKlines 单 inst 日棒种子（confirm=true 收线棒；drift=日对数漂移，
// vol=日对数收益幅；末棒 ts=now()-24h 鲜于 staleBars 36h 界）。
func seedKlines(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst string, n int, drift, vol float64) {
	t.Helper()
	seedKlinesV(t, ctx, pool, "okx", inst, n, drift, vol)
}

// seedKlinesV venue 参数化形（O-38593 mt5 影子臂 E2E——venue='mt5'
// 宇宙同径产卡断言）。
func seedKlinesV(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue, inst string,
	n int, drift, vol float64) {
	t.Helper()
	px := 100.0
	base := time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	for i := 0; i < n; i++ {
		// 交替 ±vol 叠漂移——sigma≈vol，mom≈drift·n
		r := drift + vol
		if i%2 == 1 {
			r = drift - vol
		}
		px *= math.Exp(r)
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES($4,$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
			inst, ts, px, venue); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	for _, inst := range []string{"TESTUP-USDT-SWAP", "TESTFLAT-USDT-SWAP", "TESTDN-USDT-SWAP"} {
		for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
				t.Fatal(err)
			}
		}
	}
	// 趋势面：UP=日均+1.5%（vol 1%——z 高分多）；DN=日均-1.5%（空侧）；
	// FLAT=零漂纯噪（z≈0 拒）。
	seedKlines(t, ctx, pool, "TESTUP-USDT-SWAP", 21, 0.015, 0.010)
	seedKlines(t, ctx, pool, "TESTDN-USDT-SWAP", 21, -0.015, 0.010)
	seedKlines(t, ctx, pool, "TESTFLAT-USDT-SWAP", 21, 0.0, 0.010)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := NewRunner(pool, cfg).ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// UP：long_perp proposed+提案行；short_perp 反信号拒。
	var v string
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id='TESTUP-USDT-SWAP' AND template='dir_perp_mom' AND direction='long_perp'`).Scan(&v); err != nil || v != "proposed" {
		t.Fatalf("UP long_perp 判词=%v err=%v 期 proposed", v, err)
	}
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id='TESTUP-USDT-SWAP' AND template='dir_perp_mom' AND direction='short_perp'`).Scan(&v); err != nil || v != "rejected" {
		t.Fatalf("UP short_perp 判词=%v err=%v 期 rejected(counter_signal)", v, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TESTUP-USDT-SWAP' AND direction='long_perp' AND state='fin_proposed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("UP long_perp 提案数=%v err=%v 期 1", n, err)
	}
	// DN：short_perp proposed。
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id='TESTDN-USDT-SWAP' AND template='dir_perp_mom' AND direction='short_perp'`).Scan(&v); err != nil || v != "proposed" {
		t.Fatalf("DN short_perp 判词=%v err=%v 期 proposed", v, err)
	}
	// FLAT：零信号判词=rejected(z_weak) 双方向零提案（z=0 浮点尾巴方向
	// 位不定——断言「唯一行存在且 rejected」非钉方向）。
	var allRej bool
	if err := pool.QueryRow(ctx, `SELECT count(*),bool_and(verdict='rejected') FROM fin_opp_eval
		WHERE inst_id='TESTFLAT-USDT-SWAP' AND template='dir_perp_mom'`).Scan(&n, &allRej); err != nil || n != 1 || !allRej {
		t.Fatalf("FLAT 行数=%v 全拒=%v err=%v 期 1 行 rejected(z_weak)", n, allRej, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TESTFLAT-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("FLAT 提案数=%v 期 0", n)
	}
}

// TestScanE2EMt5Venue O-38593 回归锚：mt5 影子臂（venue='mt5' 全目录
// instLike='%'）同判据产卡——史病灶=周棒误标 '1D' 下 56h 陈棒界结构性
// 永不满足零产出；本测钉 venue 泛构造径 eval 行真实落账（判词可见=
// 评估活着，execfam 册内 proposed 可达提案径）。
func TestScanE2EMt5Venue(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	inst := "TESTMT5m"
	for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
			t.Fatal(err)
		}
	}
	seedKlinesV(t, ctx, pool, "mt5", inst, 21, 0.015, 0.010)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunnerVenue(pool, cfg, "mt5", "%", "mt5_dir_mom", "MT5方向评估")
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var v, legs string
	if err := pool.QueryRow(ctx, `SELECT verdict, legs::text FROM fin_opp_eval
		WHERE inst_id=$1 AND template='mt5_dir_mom' AND direction='long_perp'`, inst).Scan(&v, &legs); err != nil {
		t.Fatalf("mt5_dir_mom eval 行缺: %v", err)
	}
	if v != "proposed" {
		t.Fatalf("mt5 UP long_perp 判词=%v 期 proposed", v)
	}
	if legs != `["cfd_long"]` {
		t.Fatalf("mt5 腿原语=%v 期 cfd_long（legName venue 分派锚）", legs)
	}
}

// O-38640 回归锚：宇宙非空+全陈棒→零判词卡→watch 不落 OK→
// 2×beat 静默告警（旧语义拍返回即 OK=永不告警=mt5_dir_mom
// 史零产出病灶形）；对照：宇宙空=合理零产出 Quiet 不误报。
type capSender struct{ sent []string }

func (f *capSender) SendText(_ context.Context, _, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

func silentWatchAl() (*notify.Alerter, *capSender) {
	fs := &capSender{}
	return notify.NewAlerter(fs, func() string { return "c1" }, func() bool { return true }), fs
}

func TestScanStaleUniverseSilent(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	inst := "TESTSTALE-USDT-SWAP"
	seedKlines(t, ctx, pool, inst, 21, 0.015, 0.010)
	// 全棒陈化 10d——staleBars 界外=零卡（宇宙仍在非空）。
	if _, err := pool.Exec(ctx,
		`UPDATE md_klines SET ts=ts-interval '10 days' WHERE inst_id=$1`, inst); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	al, fs := silentWatchAl()
	r := NewRunner(pool, cfg).AttachAlerter(al)
	fakeNow := time.Now()
	r.watch.SetNow(func() time.Time { return fakeNow })
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var evalN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval WHERE inst_id=$1`, inst).Scan(&evalN); err != nil {
		t.Fatal(err)
	}
	if evalN != 0 {
		t.Fatalf("陈棒宇宙不应产判词卡, got %d", evalN)
	}
	fakeNow = fakeNow.Add(49 * time.Hour) // >2×24h beat
	r.watch.Check(ctx)
	if len(fs.sent) != 1 {
		t.Fatalf("宇宙在零卡超 2×beat 应告警恰一次: %v", fs.sent)
	}
}

func TestScanEmptyUniverseQuiet(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// instLike 失配=宇宙空——Quiet 落活性不告警。
	al, fs := silentWatchAl()
	r := NewRunnerVenue(pool, cfg, "okx", "NOSUCHINST%", Template, "方向评估").AttachAlerter(al)
	fakeNow := time.Now()
	r.watch.SetNow(func() time.Time { return fakeNow })
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// 宇宙空 Quiet 每拍续锚——23h 后再拍（<2×beat 窗内）不告警。
	fakeNow = fakeNow.Add(23 * time.Hour)
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	fakeNow = fakeNow.Add(23 * time.Hour)
	r.watch.Check(ctx)
	if len(fs.sent) != 0 {
		t.Fatalf("宇宙空合理零产出不应告警——误报: %v", fs.sent)
	}
}
