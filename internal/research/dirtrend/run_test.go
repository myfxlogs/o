package dirtrend

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

// migrateBoth 测试面跨域（md_klines/md 域表+fin 判据键）——两链并集态
// （md 先 fin 后=历史重放方向，dirmom/funding 测试同款）。
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
		"research.dirtrend.scan_interval_min":  "240",
		"research.dirtrend.lookback_days":      "20",
		"research.dirtrend.replay_days":        "60",
		"research.dirtrend.winrate_lb":         "0.5",
		"research.dirtrend.min_signal_samples": "10",
		"research.dirtrend.hold_days":          "10",
		"research.dirtrend.friction_rt":        "0.001",
		"research.dirtrend.min_net_apr":        "0.05",
	} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
}

// seedKlines 单 inst 日棒种子（confirm=true 收线棒；drift=日对数漂移，
// vol=日对数收益幅交替±——单调漂移面=每日破前 lookback 通道全样本；
// 末棒 ts=now()-24h 鲜于 staleBars 56h 界）。
func seedKlines(t *testing.T, ctx context.Context, pool *pgxpool.Pool,
	inst string, n int, drift, vol float64) {
	t.Helper()
	px := 100.0
	base := time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	for i := 0; i < n; i++ {
		r := drift + vol
		if i%2 == 1 {
			r = drift - vol
		}
		px *= math.Exp(r)
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES('okx',$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
			inst, ts, px); err != nil {
			t.Fatal(err)
		}
	}
}

// seedWave 先升后降波形种子——cl 100→140→101 再末棒收 115 严格居
// 末 20 棒通道（101,120）内：no_breakout 确定性案（几何漂移±vol 交替
// 面在浮点边界上 last≈hi 掷币不取——锯齿期值全等=边界命中非信号）。
func seedWave(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inst string, n int) {
	t.Helper()
	base := time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	for i := 0; i < n; i++ {
		var px float64
		switch {
		case i < n/2:
			px = 100 + float64(i) // 100→139 升段
		case i < n-1:
			px = 100 + float64(n-1-i) // 140→101 降段
		default:
			px = 115 // 末棒收中轨——通道内无突破
		}
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES('okx',$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
			inst, ts, px); err != nil {
			t.Fatal(err)
		}
	}
}

// seedSpike 末棒跳变种子——前 n-1 棒平直 100、末棒跳至 spike：今棒
// Donchian 突破成立而回放窗零样本（薄样本案）。
func seedSpike(t *testing.T, ctx context.Context, pool *pgxpool.Pool, inst string, n int, spike float64) {
	t.Helper()
	base := time.Now().Add(-time.Duration(n) * 24 * time.Hour)
	for i := 0; i < n; i++ {
		px := 100.0
		if i == n-1 {
			px = spike
		}
		ts := base.Add(time.Duration(i) * 24 * time.Hour)
		if _, err := pool.Exec(ctx, `INSERT INTO md_klines(venue,inst_id,bar,ts,o,h,l,c,vol,vol_ccy,confirm)
			VALUES('okx',$1,'1D',$2,$3,$3,$3,$3,1,1,true)`,
			inst, ts, px); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanE2E(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	for _, inst := range []string{"TUP-USDT-SWAP", "TDN-USDT-SWAP", "TSW-USDT-SWAP", "TSP-USDT-SWAP"} {
		for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
			if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
				t.Fatal(err)
			}
		}
	}
	// W=lookback+replay+1=81 确认棒/inst：UP=日均+2%单调升（每棒破前
	// 20 通道→60 回放样本全胜→proposed long）；DN=日均-2%单调降
	// （空侧同构→proposed short）；SW=升降波形末棒收通道内（零突破
	// →no_breakout 双向拒）；SP=80 平+末棒跳 10%（今棒突破成立回放零
	// 样本→thin_samples）。
	seedKlines(t, ctx, pool, "TUP-USDT-SWAP", 81, 0.02, 0.005)
	seedKlines(t, ctx, pool, "TDN-USDT-SWAP", 81, -0.02, 0.005)
	seedWave(t, ctx, pool, "TSW-USDT-SWAP", 81)
	seedSpike(t, ctx, pool, "TSP-USDT-SWAP", 81, 110)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := NewRunner(pool, cfg).ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	// UP：long_perp proposed+提案行；short_perp 反信号拒。
	var v, reason string
	if err := pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='TUP-USDT-SWAP' AND template='perp-trend-daily' AND direction='long_perp'`).Scan(&v, &reason); err != nil || v != "proposed" {
		t.Fatalf("UP long_perp 判词=%v/%q err=%v 期 proposed", v, reason, err)
	}
	if err := pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='TUP-USDT-SWAP' AND template='perp-trend-daily' AND direction='short_perp'`).Scan(&v, &reason); err != nil || v != "rejected" || reason != "counter_signal" {
		t.Fatalf("UP short_perp 判词=%v/%q err=%v 期 rejected(counter_signal)", v, reason, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TUP-USDT-SWAP' AND direction='long_perp' AND state='fin_proposed'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("UP long_perp 提案数=%v err=%v 期 1", n, err)
	}
	// DN：short_perp proposed。
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id='TDN-USDT-SWAP' AND template='perp-trend-daily' AND direction='short_perp'`).Scan(&v); err != nil || v != "proposed" {
		t.Fatalf("DN short_perp 判词=%v err=%v 期 proposed", v, err)
	}
	// SW：no_breakout——双向各一行 rejected（无信号向可挂如实记）。
	var allRej bool
	if err := pool.QueryRow(ctx, `SELECT count(*),bool_and(verdict='rejected' AND evidence->>'reason'='no_breakout')
		FROM fin_opp_eval WHERE inst_id='TSW-USDT-SWAP' AND template='perp-trend-daily'`).Scan(&n, &allRej); err != nil || n != 2 || !allRej {
		t.Fatalf("SW 行数=%v 全拒no_breakout=%v err=%v 期 2 行 rejected(no_breakout)", n, allRej, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TSW-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("SW 提案数=%v 期 0", n)
	}
	// SP：今棒突破成立但回放零样本→thin_samples（信号向单行）。
	if err := pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='TSP-USDT-SWAP' AND template='perp-trend-daily' AND direction='long_perp'`).Scan(&v, &reason); err != nil || v != "rejected" || reason != "thin_samples" {
		t.Fatalf("SP long_perp 判词=%v/%q err=%v 期 rejected(thin_samples)", v, reason, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id='TSP-USDT-SWAP' AND template='perp-trend-daily'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("SP eval 行数=%v err=%v 期 1（信号向单行）", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_proposal
		WHERE inst_id='TSP-USDT-SWAP'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("SP 提案数=%v 期 0", n)
	}
}

// TestEvalInsertOnlyVersions 工单#19778 验收锚：fin_opp_eval 版本化
// insert-only——同键重评=新增版行不覆写（upsert 灭史病灶根修：入场信念
// 成史可查），读面取最新=id DESC。
func TestEvalInsertOnlyVersions(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	migrateBoth(t, ctx, pool)
	seedCfg(t, ctx, pool)
	const inst = "TVER-USDT-SWAP"
	for _, tb := range []string{"md_klines", "fin_opp_eval", "fin_proposal"} {
		if _, err := pool.Exec(ctx, "DELETE FROM "+tb+" WHERE inst_id=$1", inst); err != nil {
			t.Fatal(err)
		}
	}
	seedKlines(t, ctx, pool, inst, 81, 0.02, 0.005)
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	for i := 0; i < 2; i++ {
		if err := r.ScanOnce(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE inst_id=$1 AND template='perp-trend-daily' AND direction='long_perp'`, inst).Scan(&n); err != nil || n != 2 {
		t.Fatalf("同键重评版数=%v err=%v 期 2（insert-only 版本史）", n, err)
	}
	var v string
	if err := pool.QueryRow(ctx, `SELECT verdict FROM fin_opp_eval
		WHERE inst_id=$1 AND template='perp-trend-daily' AND direction='long_perp'
		ORDER BY id DESC LIMIT 1`, inst).Scan(&v); err != nil || v != "proposed" {
		t.Fatalf("最新版判词=%v err=%v 期 proposed", v, err)
	}
}
