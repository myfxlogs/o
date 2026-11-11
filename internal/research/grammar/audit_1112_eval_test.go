// audit_1112_eval_test.go —— 审计/评估枚举 RED 锚（audit/1112-eval）：
// 断言正确语义——修复落库后自然转绿，门已撤转正为回归锚（升件#10793 修）。
package grammar

import (
	"os"
	"strings"
	"testing"
)

// writeEval 去重判据败情静默成 nil（swallow-sql-err-dead-guard 判例第三例——
// ledger run.go:164/184 升件#9962 同族）：`err != nil || n > 0 → return nil`
// 把「去重查询库败」与「同窗同因已报」混判全吞——证据列漂移/瞬断时
// fin_opp_eval 枚举面静默断更：评估行停写而全链零痕零计数（INSERT 败反而
// 上抛——同函数内两查询败情口径不一）。枚举面断更=评估→提案链失明、
// 族沉默探针被下游噪声遮蔽真因。正确语义=去重判据败情上抛（调用方
// rej.Inc 留痕），仅 n>0 判重。
func TestAuditEvalwriteDedupSwallowsErr(t *testing.T) {
	src, err := os.ReadFile("evalwrite.go")
	if err != nil {
		t.Fatal(err)
	}
	// 正语义形：n>0 独立判重；查询败 err 必须上行非并入静默支。
	// 病灶形=单行 || 合并判（err!=nil || n>0 → return nil）。
	if strings.Contains(string(src), "err != nil || n > 0") {
		t.Fatalf("writeEval 去重判据吞错在案（err!=nil||n>0→nil 同支）——库败与判重混吞=枚举面静默断更零痕")
	}
}
