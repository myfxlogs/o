// run_test.go —— dcd 臂桩测（T-0102 判据锚）：夹具 products→评估行→提案落库+
// 阈拒因；读败 fail-closed 不造零快照；「注册即被枚举」扩张自证（关态面可见
// 不真探）；单 uly 方向不叠。
package dcd

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Runner) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewRunner(pool, cfg)
}

// fixtureProducts 合格品两（C/P）+越界品四（dist_thin/apr_thin/term_over/
// insufficient_balance）——spot=80000 锚（md_ticks 夹具）。
func fixtureProducts() []Product {
	d7 := time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second)
	d45 := time.Now().UTC().Add(45 * 24 * time.Hour).Truncate(time.Second)
	return []Product{
		{ProductID: "BTC-USDT-261101-92000-C", Uly: "BTC-USDT", Direction: "C",
			Strike: 92000, Expiry: d7, AnnualizedYield: 0.13, MinSize: 0.01},
		{ProductID: "BTC-USDT-261101-68000-P", Uly: "BTC-USDT", Direction: "P",
			Strike: 68000, Expiry: d7, AnnualizedYield: 0.11, MinSize: 100},
		{ProductID: "BTC-USDT-261101-82000-C", Uly: "BTC-USDT", Direction: "C",
			Strike: 82000, Expiry: d7, AnnualizedYield: 0.30, MinSize: 0.01},
		{ProductID: "BTC-USDT-261101-68000-P2", Uly: "BTC-USDT", Direction: "P",
			Strike: 68000, Expiry: d7, AnnualizedYield: 0.03, MinSize: 100},
		{ProductID: "BTC-USDT-261208-92000-C", Uly: "BTC-USDT", Direction: "C",
			Strike: 92000, Expiry: d45, AnnualizedYield: 0.20, MinSize: 0.01},
		{ProductID: "BTC-USDT-261101-92000-C2", Uly: "BTC-USDT", Direction: "C",
			Strike: 92000, Expiry: d7, AnnualizedYield: 0.13, MinSize: 5},
	}
}

func count(t *testing.T, ctx context.Context, r *Runner, q string, args ...any) int {
	t.Helper()
	var n int
	if err := r.pool.QueryRow(ctx, q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// 锚链：夹具 products→快照 6 行→评估行 verdict/证据→合格品提案 2 行→
// 重扫不叠（单 uly 方向闸+提案幂等）。
func TestDCDEnumerateEvaluatePropose(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'BTC-USDT',80000)`); err != nil {
		t.Fatal(err)
	}
	r.AttachFeed(func(context.Context, Surface) ([]Product, error) { return fixtureProducts(), nil })
	r.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"BTC": 1, "USDT": 500}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM md_dcd_product WHERE surface='dcd'`); n != 6 {
		t.Fatalf("快照应 6 行: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_eval WHERE template LIKE 'dcd_%'
		AND verdict='proposed'`); n != 2 {
		t.Fatalf("合格品评估应 2: %d", n)
	}
	var strike, dist, spot, term float64
	if err := r.pool.QueryRow(ctx, `SELECT (evidence->>'strike')::numeric,
		(evidence->>'dist_pct')::numeric, (evidence->>'spot')::numeric,
		(evidence->>'term')::numeric FROM fin_opp_eval
		WHERE inst_id='BTC-USDT-261101-92000-C' LIMIT 1`).Scan(&strike, &dist, &spot, &term); err != nil {
		t.Fatal(err)
	}
	if spot != 80000 || strike != 92000 || dist < 14.9 || dist > 15.1 || term < 6 || term > 8 {
		t.Fatalf("证据不齐: spot=%v strike=%v dist=%v term=%v", spot, strike, dist, term)
	}
	for _, reason := range []string{"dist_thin", "apr_thin", "term_over", "insufficient_balance"} {
		if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_eval
			WHERE evidence->>'reason'=$1 AND verdict='rejected'`, reason); n < 1 {
			t.Fatalf("阈拒因 %s 缺行", reason)
		}
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id LIKE 'BTC-USDT%' AND state='fin_proposed'`); n != 2 {
		t.Fatalf("提案应恰 2（C+P）: %d", n)
	}
	// 重扫：单 uly 方向不叠+提案幂等——提案仍 2，重评行 rejected。
	// lastAt 重置越节流窗（生产=事件风暴节流，测试=逐拍显式放行）。
	r.lastAt = time.Time{}
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_proposal
		WHERE inst_id LIKE 'BTC-USDT%' AND state='fin_proposed'`); n != 2 {
		t.Fatalf("重扫提案应仍 2（不叠）: %d", n)
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM fin_opp_eval
		WHERE evidence->>'reason'='duplicate_direction'`); n < 2 {
		t.Fatalf("重扫应出方向叠拒因: %d", n)
	}
}

// 锚：注册即被枚举（关态测试面在全集可见）+scan 只真探 enabled 面。
func TestRegistryExpansionZeroCode(t *testing.T) {
	ctx, r := setup(t)
	ss, err := r.surfaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var seenTest, seenDcd bool
	for _, s := range ss {
		if s.Surface == "dcd-test" && !s.Enabled {
			seenTest = true
		}
		if s.Surface == "dcd" && s.Enabled {
			seenDcd = true
		}
	}
	if !seenTest || !seenDcd {
		t.Fatalf("注册表全集缺面（dcd-test 关态应可见）: %+v", ss)
	}
	var called []string
	r.AttachFeed(func(_ context.Context, s Surface) ([]Product, error) {
		called = append(called, s.Surface)
		return nil, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if len(called) != 1 || called[0] != "dcd" {
		t.Fatalf("scan 应只真探 enabled 面: %v", called)
	}
}

// 锚：读败 fail-closed 不造零快照。
func TestDCDReadFailClosed(t *testing.T) {
	ctx, r := setup(t)
	r.AttachFeed(func(context.Context, Surface) ([]Product, error) {
		return nil, errors.New("dcd products 500")
	})
	if err := r.ScanOnce(ctx); err == nil {
		t.Fatal("feed 读败应上抛")
	}
	if n := count(t, ctx, r, `SELECT count(*) FROM md_dcd_product`); n != 0 {
		t.Fatalf("读败禁造零快照: %d", n)
	}
}

// T-0117 返修锚：臂静默走业主通道（notify.Alerter.Alert 非 log 前缀——
// kindSpec research_arm_silent=humanize 飞书可达）；超 2×节拍零产出置标。
func TestArmSilentFlagOwnerChannel(t *testing.T) {
	ctx, r := setup(t)
	fs := &dcdCapSender{}
	r.AttachAlerter(notify.NewAlerter(fs, func() string { return "c1" }, func() bool { return true }))
	iv := 30 * time.Minute
	r.watch.SetBeat(iv)
	now := time.Now()
	r.watch.SetNow(func() time.Time { return now })
	r.watch.OK() // 上次产出 t0
	now = now.Add(3 * time.Hour)
	_ = r.scan(ctx) // params 缺径=fail-closed 让位；静默判定仍须发
	if len(fs.sent) == 0 {
		t.Fatal("业主通道应收到研究臂静默告警")
	}
	if !strings.Contains(fs.sent[0], "研究臂静默") {
		t.Fatalf("载荷应 humanize 标题: %q", fs.sent[0])
	}
}

type dcdCapSender struct{ sent []string }

func (f *dcdCapSender) SendText(ctx context.Context, chatID, text string) (string, error) {
	f.sent = append(f.sent, text)
	return "m1", nil
}

// T-0117 返修锚③：uly 方言归一——DCD uly=指数名 BTC-USD，行情采 BTC-USDT
// （归一序 uly→+T→+T-SWAP→+-SWAP 首命中）——spot_missing 假拒不得再演，
// 评估出 evaluable dcd 行（proposed 或带数值 reason 的 rejected）。
func TestUlyDialectNormalizedEvaluable(t *testing.T) {
	ctx, r := setup(t)
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),'BTC-USDT',80000)`); err != nil {
		t.Fatal(err)
	}
	// 全 uly=保底族指数名 BTC-USDG（SG 实拍返修③方言形）；call+put 各一合格品。
	d7 := time.Now().UTC().Add(7 * 24 * time.Hour).Truncate(time.Second)
	r.AttachFeed(func(_ context.Context, s Surface) ([]Product, error) {
		return []Product{
			{ProductID: "BTC-USDG-261101-92000-C", Uly: "BTC-USDG", Direction: "C",
				Strike: 92000, Expiry: d7, AnnualizedYield: 0.13, MinSize: 0.01},
			{ProductID: "BTC-USDG-261101-68000-P", Uly: "BTC-USDG", QuoteCcy: "USDG",
				Direction: "P", Strike: 68000, Expiry: d7, AnnualizedYield: 0.11,
				MinSize: 100},
		}, nil
	})
	r.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"BTC": 1, "USDT": 420}, nil
	})
	if err := r.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var nEval int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_opp_eval
		WHERE template IN ('dcd_call','dcd_put')
		AND (verdict='proposed' OR (verdict='rejected' AND evidence->>'reason'
		NOT IN ('spot_missing','expiry_missing','strike_missing','coins_absent')))`).Scan(&nEval); err != nil {
		t.Fatal(err)
	}
	if nEval != 2 {
		t.Fatalf("uly 归一后应两 evaluable dcd 行: %d", nEval)
	}
	var spot float64
	if err := r.pool.QueryRow(ctx, `SELECT (evidence->>'spot')::numeric FROM fin_opp_eval
		WHERE inst_id='BTC-USDG-261101-92000-C' LIMIT 1`).Scan(&spot); err != nil || spot != 80000 {
		t.Fatalf("spot 应归一取 BTC-USDT 价 80000: %v %v", spot, err)
	}
	// USDG 计价 put 锁仓币=USDG 非 USDT——余量错币过闸=钱径方言坑。
	var locked string
	if err := r.pool.QueryRow(ctx, `SELECT evidence->>'locked_ccy' FROM fin_opp_eval
		WHERE inst_id='BTC-USDG-261101-68000-P' LIMIT 1`).Scan(&locked); err != nil || locked != "USDG" {
		t.Fatalf("put 锁仓币应=USDG: %v %v", locked, err)
	}
}
