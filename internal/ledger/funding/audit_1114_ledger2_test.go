// audit_1114_ledger2_test.go —— 审计/账本对账(二巡) RED 锚：record() 未映射
// 子型 bill 现金守恒断——册载 dust_in/dust_out/supply_income/airdrop 等
// 5 kind 零写者=该域漏账。默认 skip；显跑：
// AUDIT_RED=1 go test ./internal/ledger/funding -run TestAuditUnmappedBillConservation
package funding

import (
	"os"
	"testing"
	"time"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1")
	}
}

// TestAuditUnmappedBillConservation 不变量（现金守恒——ingest.go:167 钦定
// 「balChg=PnL+fee」同律应及全域）：凡落 fin_venue_bill 的 bill 行
// balChg≠0 必有 cashflow 行覆盖其现金移动。现 default 支只出 fee+衍生品
// trade_pnl——balChg≠0 而 fee=pnl=0 的未映射子型（dust 归集/理财收付/
// 空投奖励类——fin_0075 册载 dust_in/dust_out/supply_income/airdrop 正为
// 此域生而零写者=账册预备无产者）整额漏账→cashflow 净额 vs venue 权益
// 漂差=对账 phantom diff 源。正确语义=未映射子型 balChg≠0 须落账（映射
// 入册载 kind）或显式升件（不可静默零覆盖）——修好本测试自然转绿。
func TestAuditUnmappedBillConservation(t *testing.T) {
	auditRedGate(t)
	ctx, pool := setup(t)
	i := NewIngester(pool, fakeSrc{})
	b := Bill{BillID: "unmapped-dust-1", SubType: "200", Type: "8",
		Ccy: "USDT", TsMs: time.Now().Add(-time.Minute).UnixMilli(),
		BalChg: 5.0, Raw: []byte(`{"billId":"unmapped-dust-1"}`)}
	n, err := i.record(ctx, b)
	if err != nil {
		t.Fatalf("record 上抛: %v", err)
	}
	var cashRows int
	var covered float64
	if err := pool.QueryRow(ctx,
		`SELECT count(*), COALESCE(sum(amount),0) FROM fin_cashflow c
		 JOIN fin_venue_bill v ON c.source_table='fin_venue_bill' AND c.source_ref=v.id
		 WHERE v.bill_id=$1`, b.BillID).Scan(&cashRows, &covered); err != nil {
		t.Fatal(err)
	}
	if n == 0 || covered != b.BalChg {
		t.Fatalf("未映射子型 bill(balChg=%.2f) cashflow 覆盖=%.2f 行数=%d——现金守恒断"+
			"（源行落 fin_venue_bill 而 cashflow 零/欠覆盖=对账 phantom diff；册载 dust/supply/airdrop kind 无产者即此洞）",
			b.BalChg, covered, cashRows)
	}
}

// TestAuditSpotBillFeeDoubleBook 不变量（单源——升件#10063① 钦定
// 「SPOT/MARGIN 现货腿走 legCash 毛额对称径，bill 补记=双计」同律须及
// fee）：flow_leg.go legCash 已对 USDT 费现货腿落 kind='fee'；ingest
// default 支 `if b.Fee != 0` 无 InstType 闸——同一笔现货交易费被
// fin_order_leg 源+bill 源各记一次→fee kind 双倍→realizedKinds 实现盈亏
// 系统性低估+对账净额常偏 -fee。正确语义=SPOT/MARGIN bill 费支同
// trade_pnl 显排（现货域归 legCash 单源）——修好本测试自然转绿。
func TestAuditSpotBillFeeDoubleBook(t *testing.T) {
	auditRedGate(t)
	ctx, pool := setup(t)
	i := NewIngester(pool, fakeSrc{})
	b := Bill{BillID: "spot-fee-1", SubType: "1", Type: "2", InstType: "SPOT",
		InstID: "BTC-USDT", Ccy: "USDT", TsMs: time.Now().Add(-time.Minute).UnixMilli(),
		BalChg: 99.5, Fee: -0.5, Raw: []byte(`{"billId":"spot-fee-1"}`)}
	if _, err := i.record(ctx, b); err != nil {
		t.Fatalf("record 上抛: %v", err)
	}
	var feeRows int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_cashflow c
		 JOIN fin_venue_bill v ON c.source_table='fin_venue_bill' AND c.source_ref=v.id
		 WHERE v.bill_id=$1 AND c.kind='fee'`, b.BillID).Scan(&feeRows); err != nil {
		t.Fatal(err)
	}
	if feeRows != 0 {
		t.Fatalf("SPOT bill fee 落 fee kind %d 行——现货费域归 legCash 单源，bill 补记=同事双计"+
			"（trade_pnl 已按升件#10063① 排 SPOT，fee 支漏同 InstType 闸）", feeRows)
	}
}
