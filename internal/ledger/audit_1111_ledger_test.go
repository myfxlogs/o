// audit_1111_ledger_test.go —— 审计/账本对账 RED 锚（audit/1111-ledger）：
// 断言正确语义——修复落库后自然转绿，门转正为回归锚。
// 显跑：AUDIT_RED=1 go test ./internal/ledger
package ledger

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func auditRedGate(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1 go test -run " + t.Name() + " ./internal/ledger；修好后去门转正为回归锚")
	}
}

// ③ 判据喂数吞错回潮（swallow-sql-err-dead-guard 判例）：reconcile 的
// 账本净额（fin_cashflow 求和）与豁免额（fin_known_residual 求和）两处
// 判词喂数查询被 `_ =` 吞错——列漂移/权限缺/连接断时 net=0→venue 全额
// phantom diff 连刷、exempt=0→豁免静默失效过警，两方向皆无计数零痕。
// 判例钦定=判定面探测错误必须 rej 计数+WARN+本轮让位（fail-closed
// 可观测），禁裸吞。源码级锚：两查询不得 `_ =` 吞错（含分行写法——
// sig 行回溯 3 行内查裸吞赋值）。
func TestAuditJudgmentFeedSwallow(t *testing.T) {
	auditRedGate(t)
	_, file, _, _ := runtime.Caller(0)
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), "run.go"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(src), "\n")
	for _, sig := range []string{
		"sum(amount),0) FROM fin_cashflow",
		"sum(amount_usd),0) FROM fin_known_residual",
	} {
		for i, line := range lines {
			if !strings.Contains(line, sig) {
				continue
			}
			for j := i; j >= 0 && j >= i-3; j-- { // 回溯至语句起点查裸吞
				if strings.Contains(lines[j], "QueryRow") {
					if strings.Contains(lines[j], "_ =") {
						t.Fatalf("判据喂数吞错: run.go:%d %s——错误须 rej 计数+让位非静默零值（判例回潮）", j+1, strings.TrimSpace(lines[j]))
					}
					break
				}
			}
		}
	}
}
