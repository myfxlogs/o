// audit_1114_life2_test.go —— 审计/生命周期(二巡) RED 锚②：execDrillPass
// 库败静默映 drilled=false（注释钦定「读败=err 上抛」不实——swallow-
// sql-err-dead-guard 判例同型）。默认 skip；显跑：
// AUDIT_RED=1 go test ./internal/portfolio/lifecycle -run TestAuditExecDrillPassErrProp
package lifecycle

import (
	"context"
	"os"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func auditRedGateLife2(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1")
	}
}

// TestAuditExecDrillPassErrProp 不变量（paper.go:192 钦定「读败=err 上抛
// fail-closed」）：fin_exec_drill 查询败（undefined_table 型库败）须
// err 透传——断凭 vs 断库必须可辨。现实：QueryRow+Scan 的任何 err 一律
// return(false,false,nil)——签名有 err 返回位而实现永 nil=死返回值；
// 库败态被映成「凭据缺」且零痕（exec_drill_missing 计数不辨）。正确语义
// =ErrNoRows→drilled=false（真缺件），余 err→上抛。测试法：表更名注
// 库败——查询必挂 undefined_table。
func TestAuditExecDrillPassErrProp(t *testing.T) {
	auditRedGateLife2(t)
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`ALTER TABLE fin_exec_drill RENAME TO fin_exec_drill_audit_bak`); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(),
			`ALTER TABLE fin_exec_drill_audit_bak RENAME TO fin_exec_drill`)
	}()
	r := NewRunner(pool, config.New(pool))
	_, _, err := r.execDrillPass(ctx, 1)
	if err == nil {
		t.Fatal("fin_exec_drill 表缺席（undefined_table 库败）execDrillPass 仍返 nil err——" +
			"全类库错静默映 drilled=false：断凭 vs 断库不可辨，注释钦定 err 上抛不实")
	}
}
