// audit_1111_ledger_test.go —— 审计/账本对账 回归锚（audit/1111-ledger 修复转绿落库）：
// 断言正确语义常驻——三锚即修复面回归。/funding
package funding

import (
	"testing"
	"time"
)

// ① 合约域 trade bill 实现盈亏漏账：OKX SWAP/FUT 平仓 bill 的 balChg=
// 实现PnL+fee（官义），ingest record() default 支只记 fee——Pnl 分量进
// fin_venue_bill 而止，永不到 fin_cashflow。legCash 钦定「合约腿无名义
// 现金（结算走 bills/fee）」=把合约腿现金托给 bills 径，但 bills 径丢
// Pnl 分量→每平一仓合约腿账本漏记其实现盈亏→venue 权益与账本净额
// 系统性漂移（对账 phantom diff 源）+realizedKinds 'trade_pnl' 有册无
// 产者（仅 dustsweep 手工件写）。现货腿 balChg 已由 legCash 毛额对称记
// ——本锚只钉合约域（instType=SWAP/FUT/OPTION 类）bill 的现金守恒：
// Σcashflow(bill 源行)==balChg。
func TestAuditContractBillPnlUnbooked(t *testing.T) {
	ctx, pool := setup(t)
	mkPos(t, ctx, pool, "BTC-USDT-SWAP")
	nowMs := time.Now().Add(-time.Minute).UnixMilli()
	bills := []Bill{
		{BillID: "acls", SubType: "5", InstID: "BTC-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs, BalChg: -2.55, Fee: -0.05, Pnl: -2.50,
			Raw: []byte(`{}`)}, // 平仓实现损 -2.50 + 费 -0.05 = balChg -2.55
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	if _, err := ing.Ingest(ctx); err != nil {
		t.Fatal(err)
	}
	var sum float64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow c
		JOIN fin_venue_bill b ON b.id=c.source_ref AND c.source_table='fin_venue_bill'
		WHERE b.bill_id='acls'`).Scan(&sum); err != nil {
		t.Fatal(err)
	}
	if sum != -2.55 {
		t.Fatalf("合约平仓 bill 现金守恒破——balChg=-2.55 但账本只记 %.2f（PnL=-2.50 漏账：trade_pnl 有册无产者）", sum)
	}
}
