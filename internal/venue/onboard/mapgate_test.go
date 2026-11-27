// mapgate_test.go —— O-41377 刀3 锚：map 闸承认 fin_symbol_dialect 确认行
// （CFD venue 零配对 stall 根修——草案零配对不再拦，dialect 行在即跃迁；
// bridge.xvenue.map 径保留不动）。
package onboard

import (
	"context"
	"errors"
	"testing"
)

// noProbeDial 探针拨号缝桩——probePass 漂移探针不拨真网关（dialect 行在册
// 即走探针面，桩注败=探针拍让位不阻状态机断言）。
func noProbeDial(r *Runner) {
	r.dialProbeFn = func(string) (probeSession, error) {
		return nil, errors.New("桩注：探针拨号不走真网关")
	}
}

// 锚① dialect 单图源全链：草案有对+无 xvenue.map+dialect 确认行在→
// mapped→map闸（src=fin_symbol_dialect)→paper 一扫到底。
func TestDialectGateSatisfies(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	noProbeDial(r)
	r.dial = func(string) (Session, error) {
		return &fakeSession{syms: []string{"BTCUSDm", "ETHUSDm"}}, nil
	}
	seedCreds(t, ctx, pool, "df1", map[string]string{
		"login": "3001157", "password": "pw", "server": "Fake-MT5"})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_symbol_dialect
		(venue,account,canonical,venue_symbol,ct_val,tick_sz,min_sz,lot_unit,lot_step,digits,trade_mode,spec_hash)
		VALUES('df1','3001157','BTC-USDT-SWAP','BTCUSDm',1,0.01,0.01,'lots',0.01,2,4,'h1')`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "df1"); got != "paper" {
		t.Fatalf("dialect 确认行在应 paper got=%s", got)
	}
	var src string
	if err := pool.QueryRow(ctx, `SELECT evidence->'onboard'->>'map_source'
		FROM fin_venue_candidate WHERE venue='df1'`).Scan(&src); err != nil {
		t.Fatal(err)
	}
	if src != "fin_symbol_dialect" {
		t.Fatalf("map_source=%q 应 fin_symbol_dialect", src)
	}
}

// 锚② 草案零配对+dialect 行在→不 stall（mapped→paper 全链）；真零图源
// （零配对∧零 dialect）才如实记零命中不跃迁。
func TestZeroPairDialectBypass(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	noProbeDial(r)
	// 符号表与 okx 宇宙（BTC/ETH-USDT-SWAP）零交集——MatchSymbol 全 miss。
	r.dial = func(string) (Session, error) {
		return &fakeSession{syms: []string{"ZZZIDX#", "QQQQQQ#"}}, nil
	}
	seedCreds(t, ctx, pool, "zp1", map[string]string{
		"login": "3001158", "password": "pw", "server": "Fake-MT5"})
	if _, err := pool.Exec(ctx, `INSERT INTO fin_symbol_dialect
		(venue,account,canonical,venue_symbol,ct_val,tick_sz,min_sz,lot_unit,lot_step,digits,trade_mode,spec_hash)
		VALUES('zp1','3001158','BTC-USDT-SWAP','BTCUSD#',1,0.01,0.01,'lots',0.01,2,4,'h1')`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "zp1"); got != "paper" {
		t.Fatalf("零配对+dialect 在应 paper（不 stall）got=%s", got)
	}
	// 零命中不再落「接入败」草案行——dialect 径如实记 mapped 跃迁。
	var failN int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_class='接入败' AND esc_scope='zp1:3001158'`).Scan(&failN); err != nil {
		t.Fatal(err)
	}
	if failN != 0 {
		t.Fatalf("dialect 在径应零接入败行 got=%d", failN)
	}
	// 对照户：零配对∧零 dialect=真零图源——stay probed+零命中如实记。
	seedCreds(t, ctx, pool, "zp2", map[string]string{
		"login": "3001159", "password": "pw", "server": "Fake-MT5"})
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stateOf(t, ctx, pool, "zp2"); got != "probed" {
		t.Fatalf("真零图源应 stay probed got=%s", got)
	}
	var zfail int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='venue接入' AND esc_class='接入败' AND esc_scope='zp2:3001159'
		AND note LIKE '%零命中%'`).Scan(&zfail); err != nil || zfail == 0 {
		t.Fatalf("真零图源零命中败行 zfail=%d err=%v（应如实记）", zfail, err)
	}
}
