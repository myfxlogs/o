// scan_test.go —— T-0401 锚①②桩测：parseScanTickers 纯解析断言（USDT
// 键域闸+毫秒戳还原）+md_tick_scan upsert 最新赢/sub_request UNION 臂
// 入集 DB 桩（O_TEST_PG_DSN 缺=skip，testutil 惯例）。
package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestParseScanTickers 解析断言：杂币键滤除（业主令同律）+ts 毫秒还原+
// last/bid/ask 数值落位。
func TestParseScanTickers(t *testing.T) {
	raw := json.RawMessage(`[
		{"instId":"AAA-USDT-SWAP","last":"1.5","bidPx":"1.4","askPx":"1.6","ts":"1700000000000"},
		{"instId":"BBB-BTC-SWAP","last":"9","bidPx":"9","askPx":"9","ts":"1700000000000"},
		{"instId":"CCC-USDT-SWAP","last":"2","bidPx":"","askPx":"2.1","ts":"1700000060000"}
	]`)
	rows, err := parseScanTickers(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("杂币键应滤除留 2 行: %d", len(rows))
	}
	if rows[0].instID != "AAA-USDT-SWAP" || rows[0].last != 1.5 || rows[0].bid != 1.4 || rows[0].ask != 1.6 {
		t.Fatalf("行①载荷错位: %+v", rows[0])
	}
	if want := time.UnixMilli(1700000060000).UTC(); !rows[1].ts.Equal(want) {
		t.Fatalf("ts 毫秒还原错: %v want %v", rows[1].ts, want)
	}
}

// tickersStub 按 instType 查参分派 tickers 载荷（族感知——非册族回空集）。
func tickersStub(fam map[string]json.RawMessage) ScanGet {
	return func(_ context.Context, path string) (json.RawMessage, error) {
		for f, raw := range fam {
			if path == "/api/v5/market/tickers?instType="+f {
				return raw, nil
			}
		}
		return json.RawMessage(`[]`), nil
	}
}

// staticGate 族谱域闸桩（定集全集;零参=合法空集）。
func staticGate(keys ...string) ScanGate {
	return func(context.Context) (map[string]bool, error) {
		m := make(map[string]bool, len(keys))
		for _, k := range keys {
			m[k] = true
		}
		return m, nil
	}
}

// TestScanUpsertAndSubArm 锚②桩：ScanOnce upsert md_tick_scan 最新赢
// （回戳帧不覆新）+fin_sub_request open 行经 ActiveSetUnion 入集、
// closed 行出集（晋级径 UNIÓN 臂谓词语义）。
func TestScanUpsertAndSubArm(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM md_tick_scan").Scan(&n); err != nil {
		t.Fatalf("md_tick_scan 应随 fin_0124 落表: %v", err)
	}
	p := NewScanPoller(pool, nil, tickersStub(map[string]json.RawMessage{
		"SWAP": json.RawMessage(`[
			{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"},
			{"instId":"YYY-USDT-SWAP","last":"5","bidPx":"5","askPx":"5","ts":"1700000000000"}]`),
	}))
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var last float64
	if err := pool.QueryRow(ctx, `SELECT last::float8 FROM md_tick_scan
		WHERE inst_id='ZZZ-USDT-SWAP'`).Scan(&last); err != nil || last != 3 {
		t.Fatalf("粗扫行未落: %v last=%v", err, last)
	}
	// 回戳帧不覆新（EXCLUDED.ts>现 ts 守卫）。
	p.get = tickersStub(map[string]json.RawMessage{
		"SWAP": json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"99","bidPx":"9","askPx":"9","ts":"1600000000000"}]`),
	})
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT last::float8 FROM md_tick_scan
		WHERE inst_id='ZZZ-USDT-SWAP'`).Scan(&last); err != nil || last != 3 {
		t.Fatalf("回戳帧覆新=守卫破: %v last=%v", err, last)
	}
	// 晋级臂：open 请求行入 activeSet；closed 出集。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_sub_request(venue,inst_id,requester,state)
		VALUES('okx','QQQ-USDT-SWAP','xvenue-carry','open'),
		      ('okx','RRR-USDT-SWAP','xvenue-carry','closed')`); err != nil {
		t.Fatal(err)
	}
	c := &Collector{pool: pool}
	set, err := c.activeSet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !set["QQQ-USDT-SWAP"] {
		t.Fatal("open 晋级请求行未入 activeSet——SubRequestArm 断")
	}
	if set["RRR-USDT-SWAP"] {
		t.Fatal("closed 请求行误留集——退订语义断")
	}
}

// TestScanCatalogGate T-0416 谱域闸桩（T-0427 改族制）：Gates 挂线后各族
// tickers 行须∈该族目录全集才落表（目录外滤除）；族目录读败=该族让位
// 零行落表、他族照常（fail-closed 粒度到族）；合法空族集=零行非病。
func TestScanCatalogGate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	p := NewScanPoller(pool, nil, tickersStub(map[string]json.RawMessage{
		"SWAP": json.RawMessage(`[
			{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"},
			{"instId":"YYY-USDT-SWAP","last":"5","bidPx":"5","askPx":"5","ts":"1700000000000"}]`),
		"SPOT":    json.RawMessage(`[]`),
		"FUTURES": json.RawMessage(`[]`),
	}))
	// SWAP 目录全集={ZZZ}——YYY 目录外滤除不落表;空族集=合法零行。
	p.Gates = map[string]ScanGate{
		"SWAP":    staticGate("ZZZ-USDT-SWAP"),
		"SPOT":    staticGate(),
		"FUTURES": staticGate(),
	}
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("目录外行应滤除留 1 行: %d", n)
	}
	// SWAP 目录读败=该族让位零落表（存量不动）;他族照常（FUTURES 行落）。
	p.Gates["SWAP"] = func(context.Context) (map[string]bool, error) {
		return nil, errors.New("instruments 面断")
	}
	p.Gates["FUTURES"] = staticGate("QQQ-USD-260327")
	p.get = tickersStub(map[string]json.RawMessage{
		"SWAP": json.RawMessage(`[
			{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000001000"}`),
		"SPOT":    json.RawMessage(`[]`),
		"FUTURES": json.RawMessage(`[{"instId":"QQQ-USD-260327","last":"7","bidPx":"7","askPx":"7","ts":"1700000001000"}]`),
	})
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatalf("单族读败不应全拍炸: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan WHERE inst_id LIKE 'ZZZ%'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("族败拍 SWAP 应零落表（存量不动）: %v n=%d", err, n)
	}
	var fam string
	if err := pool.QueryRow(ctx, `SELECT inst_type FROM md_tick_scan
		WHERE inst_id='QQQ-USD-260327'`).Scan(&fam); err != nil || fam != "futures" {
		t.Fatalf("他族应照常落行: %v fam=%q", err, fam)
	}
}

// TestScanFamiliesLand T-0427 红桩①：三族 tickers fixture→md_tick_scan
// 携 inst_type 落行（显式族标>inst_id 键内隐式——防族判散落正则复制件）。
func TestScanFamiliesLand(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	p := NewScanPoller(pool, nil, tickersStub(map[string]json.RawMessage{
		"SWAP":    json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"}]`),
		"SPOT":    json.RawMessage(`[{"instId":"ZZZ-USDT","last":"2.9","bidPx":"2.8","askPx":"3.0","ts":"1700000001000"}]`),
		"FUTURES": json.RawMessage(`[{"instId":"ZZZ-USD-260327","last":"3.1","bidPx":"3.0","askPx":"3.2","ts":"1700000002000"}]`),
	}))
	p.Gates = map[string]ScanGate{
		"SWAP":    staticGate("ZZZ-USDT-SWAP"),
		"SPOT":    staticGate("ZZZ-USDT"),
		"FUTURES": staticGate("ZZZ-USD-260327"),
	}
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ inst, fam string }{
		{"ZZZ-USDT-SWAP", "swap"}, {"ZZZ-USDT", "spot"}, {"ZZZ-USD-260327", "futures"},
	} {
		var fam string
		if err := pool.QueryRow(ctx, `SELECT inst_type FROM md_tick_scan
			WHERE inst_id=$1`, tc.inst).Scan(&fam); err != nil || fam != tc.fam {
			t.Fatalf("族标错位 %s: %v fam=%q want %q", tc.inst, err, fam, tc.fam)
		}
	}
}

// TestScanFamilyFailClosed T-0427 红桩②：某族目录读败=该族零行他族不受
// 影响（fail-closed 粒度到族——拍不炸不虚构）;全族败=本拍 loud 岗终。
func TestScanFamilyFailClosed(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	p := NewScanPoller(pool, nil, tickersStub(map[string]json.RawMessage{
		"SWAP":    json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"}]`),
		"SPOT":    json.RawMessage(`[{"instId":"ZZZ-USDT","last":"2.9","bidPx":"2.8","askPx":"3.0","ts":"1700000001000"}]`),
		"FUTURES": json.RawMessage(`[{"instId":"ZZZ-USD-260327","last":"3.1","bidPx":"3.0","askPx":"3.2","ts":"1700000002000"}]`),
	}))
	p.Gates = map[string]ScanGate{
		"SWAP":    staticGate("ZZZ-USDT-SWAP"),
		"SPOT":    func(context.Context) (map[string]bool, error) { return nil, errors.New("spot 目录断") },
		"FUTURES": staticGate("ZZZ-USD-260327"),
	}
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatalf("SPOT 族读败不应全拍炸: %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan WHERE inst_type='spot'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("族读败该族应零行: %v n=%d", err, n)
	}
	for _, keep := range []string{"ZZZ-USDT-SWAP", "ZZZ-USD-260327"} {
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan
			WHERE inst_id=$1`, keep).Scan(&n); err != nil || n != 1 {
			t.Fatalf("他族不受影响 %s: %v n=%d", keep, err, n)
		}
	}
	// 全族败=面病灶——本拍 loud 上抛（同原 fail-closed 形）。
	p.Gates = map[string]ScanGate{
		"SWAP":    func(context.Context) (map[string]bool, error) { return nil, errors.New("swap 断") },
		"SPOT":    func(context.Context) (map[string]bool, error) { return nil, errors.New("spot 断") },
		"FUTURES": func(context.Context) (map[string]bool, error) { return nil, errors.New("futures 断") },
	}
	if err := p.ScanOnce(ctx); err == nil {
		t.Fatal("全族败应 loud 上抛岗终")
	}
}

// TestScanRunBeatFailYields T-0446 病灶本体桩（观测盲:scan:md_tick_scan→
// discovery 产者死径）：ScanOnce 全族败拍=让位下拍续试不拖死采集主
// 进程——Run 拍败 errc 零发+瞬断自愈落行（修前=errc 岗终全臂死——
// 先红断言 errc 空+行落双证）。
func TestScanRunBeatFailYields(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('okxws.tick_scan_s',to_jsonb(0.05))`); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	okGet := tickersStub(map[string]json.RawMessage{
		"SWAP": json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"}]`),
	})
	get := func(ctx context.Context, path string) (json.RawMessage, error) {
		if calls.Add(1) <= 6 { // 前两拍全族瞬断（每拍三族各一发）
			return nil, errors.New("tickers 面瞬断")
		}
		return okGet(ctx, path)
	}
	p := NewScanPoller(pool, cfg, get)
	errc := make(chan error, 4)
	rctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() { p.Run(rctx, errc); close(done) }()
	deadline := time.Now().Add(2500 * time.Millisecond)
	for {
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan`).Scan(&n); err == nil && n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("瞬断拍后未自愈落行——Run 拍败径断")
		}
		time.Sleep(20 * time.Millisecond) // sched-exempt-ok（桩断言轮询期界——等自愈拍落行非调度件）
	}
	cancel()
	select {
	case err := <-errc:
		t.Fatalf("拍败上抛 errc=全臂死病灶复现: %v", err)
	case <-done:
	case <-time.After(2 * time.Second): // sched-exempt-ok（桩收摊期界锚——Run 应随 ctx 即退）
		t.Fatal("Run 未随 ctx 收摊")
	}
}
