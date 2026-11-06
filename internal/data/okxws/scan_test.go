// scan_test.go —— T-0401 锚①②桩测：parseScanTickers 纯解析断言（USDT
// 键域闸+毫秒戳还原）+md_tick_scan upsert 最新赢/sub_request UNION 臂
// 入集 DB 桩（O_TEST_PG_DSN 缺=skip，testutil 惯例）。
package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

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
	p := NewScanPoller(pool, nil, func(context.Context, string) (json.RawMessage, error) {
		return json.RawMessage(`[
			{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"},
			{"instId":"YYY-USDT-SWAP","last":"5","bidPx":"5","askPx":"5","ts":"1700000000000"}]`), nil
	})
	if err := p.ScanOnce(ctx); err != nil {
		t.Fatal(err)
	}
	var last float64
	if err := pool.QueryRow(ctx, `SELECT last::float8 FROM md_tick_scan
		WHERE inst_id='ZZZ-USDT-SWAP'`).Scan(&last); err != nil || last != 3 {
		t.Fatalf("粗扫行未落: %v last=%v", err, last)
	}
	// 回戳帧不覆新（EXCLUDED.ts>现 ts 守卫）。
	p.get = func(context.Context, string) (json.RawMessage, error) {
		return json.RawMessage(`[{"instId":"ZZZ-USDT-SWAP","last":"99","bidPx":"9","askPx":"9","ts":"1600000000000"}]`), nil
	}
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

// TestScanCatalogGate T-0416 谱域闸桩：Catalog 挂线后 tickers 行须∈目录
// 全集才落表（目录外滤除）；目录读败=fail-closed 本拍让位零行落表。
func TestScanCatalogGate(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	p := NewScanPoller(pool, nil, func(context.Context, string) (json.RawMessage, error) {
		return json.RawMessage(`[
			{"instId":"ZZZ-USDT-SWAP","last":"3","bidPx":"2.9","askPx":"3.1","ts":"1700000000000"},
			{"instId":"YYY-USDT-SWAP","last":"5","bidPx":"5","askPx":"5","ts":"1700000000000"}]`), nil
	})
	// 目录全集={ZZZ}——YYY 目录外滤除不落表。
	p.Catalog = func(context.Context) (map[string]bool, error) {
		return map[string]bool{"ZZZ-USDT-SWAP": true}, nil
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
	// 目录读败=fail-closed——tickers 有行也零落表。
	p.Catalog = func(context.Context) (map[string]bool, error) {
		return nil, errors.New("instruments 面断")
	}
	if err := p.ScanOnce(ctx); err == nil {
		t.Fatal("目录读败应 fail-closed 让位")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM md_tick_scan`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("目录败拍应零落表（存量不动）: %v n=%d", err, n)
	}
}
