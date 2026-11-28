// subscribe_activeset_test.go —— T-0392 锚①桩测三断言：funding-hedge 族
// inst=X-USDT-SWAP→activeSet 含 X-USDT 派生现货腿（修前红钉死病灶）；
// 非 funding 模板零派生；虚构键（宇宙外）照既有 valid/isUSDTInst 闸滤除。
package okxws

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestActiveSetFundingHedgeSpotLeg 锚①：族属模板（fundingHedgeFamily 册）
// inst_id 剥 -SWAP 尾派生现货对入集；非族模板/退役态零派生。
func TestActiveSetFundingHedgeSpotLeg(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct{ inst, tmpl, state string }{
		{"AAA-USDT-SWAP", "funding-hedge", "live"},     // 族属 live→派生
		{"ZZZ-USDT-SWAP", "xvenue-carry", "paper"},     // 族属（册全集）paper→派生
		{"BBB-USDT-SWAP", "vol_sell_strangle", "live"}, // 非族→零派生
		{"CCC-USDT-SWAP", "funding-hedge", "retired"},  // 退役→零派生
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
			VALUES($1,$2,$3,100)`, s.inst, s.tmpl, s.state); err != nil {
			t.Fatal(err)
		}
	}
	c := &Collector{pool: pool}
	set, err := c.activeSet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !set["AAA-USDT"] {
		t.Fatal("funding-hedge 现货腿 AAA-USDT 未派生入集——失明病灶在")
	}
	if !set["ZZZ-USDT"] {
		t.Fatal("xvenue-carry（族册属）现货腿 ZZZ-USDT 未派生入集")
	}
	if set["BBB-USDT"] {
		t.Fatal("非族模板 BBB-USDT 误派生")
	}
	if !set["BBB-USDT-SWAP"] {
		t.Fatal("live 策略 inst 本键 BBB-USDT-SWAP 缺（arm①回退）")
	}
	if set["CCC-USDT"] || set["CCC-USDT-SWAP"] {
		t.Fatal("退役策略键域误入集")
	}
}

// TestActiveSetFictionalSpotFiltered 锚①③：虚构键（无该现货对上市的币）
// 派生入集但照既有 isUSDTInst+valid 闸滤除——不订不炸 fail-closed。
func TestActiveSetFictionalSpotFiltered(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('FAKE9-USDT-SWAP','funding-hedge','live',100)`); err != nil {
		t.Fatal(err)
	}
	c := &Collector{pool: pool}
	set, err := c.activeSet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !set["FAKE9-USDT"] {
		t.Fatal("派生面 FAKE9-USDT 缺——滤除断言行无从对拍")
	}
	// 订阅宇宙（okx.symbols 形）无 FAKE9-USDT——tickGate=syncActiveSet 同款
	// 加订谓词（isUSDTInst∩valid∩非拒集∩未订）。
	valid := map[string]bool{"FAKE9-USDT-SWAP": true, "BTC-USDT": true}
	for inst := range set {
		if tickGate(inst, valid, nil, nil) && inst == "FAKE9-USDT" {
			t.Fatal("虚构键 FAKE9-USDT 穿闸——订阅滤失效")
		}
	}
}

// TestActiveSetValuationCcyArm O-41395 件②锚：估值需求币臂——
// fin_valuation_ccy 在册 ccy→ccy-USDT 派生入集（BTC-USDT 死订根修面）；
// USDT/USD 脏行双闸滤除（发布器+臂兜底）；写谓词同源过闸（订而不写同律）。
func TestActiveSetValuationCcyArm(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fin_valuation_ccy WHERE ccy LIKE 'LWPVAL%'`)
	}()
	for _, ccy := range []string{"LWPVALCCY", "USDT", "USD"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO fin_valuation_ccy(ccy,src) VALUES($1,'test')
			 ON CONFLICT (ccy) DO UPDATE SET src='test'`, ccy); err != nil {
			t.Fatal(err)
		}
	}
	c := &Collector{pool: pool}
	set, err := c.activeSet(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !set["LWPVALCCY-USDT"] {
		t.Fatal("估值需求币 LWPVALCCY-USDT 未派生入集——死订根修面缺")
	}
	if set["USDT-USDT"] || set["USD-USDT"] {
		t.Fatal("直入币 USDT/USD 误派生（双闸滤除失效）")
	}
	pred := LocalWritePredFunc(ctx, pool)
	if !pred("tickers", "LWPVALCCY-USDT") {
		t.Fatal("估值币写谓词未过——订而不写同源闸断")
	}
}
