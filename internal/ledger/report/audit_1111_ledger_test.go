// audit_1111_ledger_test.go —— 审计/账本对账 回归锚（audit/1111-ledger 修复转绿落库）：
// 断言正确语义常驻——三锚即修复面回归。/report
package report

import (
	"context"
	"testing"
	"time"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// ② 实现盈亏面吞 venue='sim' 夹具行：realizedSQL/RealizedDaily 按 kind 求和
// 无 venue 谓词——simdrill 以真实 product/inst 播 venue='sim' 的 fee 行
// （drill.go:249——产线演习每拍实发），演习成本落业主盈亏面=跨域污染。
// 对拍件商业探针 meta_esc.go:134 已显滤 venue<>'sim'=探针知拒夹具域、
// 账面正本反吞=同字段读口径漂移。正确语义=实现盈亏面只含产线域行。
// 共享库并发稳健：按插入前后增量断言。
func TestAuditRealizedSimVenuePollution(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	end := time.Now().Add(time.Hour)
	start := end.Add(-24 * time.Hour)
	before, err := Realized(ctx, pool, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('fee','sim','USDT',-7.00,NULL,0,'simvenue',now())`); err != nil {
		t.Fatal(err)
	}
	after, err := Realized(ctx, pool, start, end)
	if err != nil {
		t.Fatal(err)
	}
	if after-before != 0 {
		t.Fatalf("实现盈亏面吞 venue='sim' 夹具行——插 -7 fee 后 Realized 漂 %.2f（当 0：夹具域与业主盈亏隔离）", after-before)
	}
}
