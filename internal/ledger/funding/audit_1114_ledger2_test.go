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
