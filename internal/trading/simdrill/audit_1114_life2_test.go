// audit_1114_life2_test.go —— 审计/生命周期(二巡) RED 锚：闸凭据消费域
// ⊆ 演习产能域（canon simdrill-gate-domain-coverage-mismatch）。默认
// skip；显跑：AUDIT_RED=1 go test ./internal/trading/simdrill -run TestAuditDrillDomainCoverage
package simdrill

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func auditRedGateLife2(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1")
	}
}

// TestAuditDrillDomainCoverage 不变量：凡可经 paper→probe→live 毕业链抵
// execDrillPass 闸的模板，必须有策略级演习凭据产能径（legShapeOf 映到
// 非拒载腿形）。包向禁 import execfam——同源消费面=源文解析 execfam.set
// 册（注册语义=paper 结算域贯通，paper→probe→live 链路同域）。
// 破面：xvenue-carry 在册可 paper→probe，而 legShapeOf→shapeXvenue→
// Drill.Run 显式拒载（「xvenue 腿形未载…拒载不产凭据」）——probe 滞留
// exec_drill_missing 至 armed_ttl 黑洞；matrix 格凭据走 cell_key 键域
// 与 execDrillPass 的 strategy_ref 键域不接=产能域两键空。
// 正确语义=①execDrillPass 只消费可钻域（册收窄）或②xvenue 凭据改由
// matrix cell 键域接通——修好（册/闸/产能三径任一闭合）本测试转绿。
func TestAuditDrillDomainCoverage(t *testing.T) {
	auditRedGateLife2(t)
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	src, err := os.ReadFile(filepath.Join(root, "internal/platform/execfam/execfam.go"))
	if err != nil {
		t.Fatal(err)
	}
	// 册解析：var set 字面量内 "tmpl": true 键集（注册域=execfam 唯一真源）。
	keys := regexp.MustCompile(`"([a-z_0-9-]+)"\s*:\s*true`).FindAllStringSubmatch(string(src), -1)
	if len(keys) == 0 {
		t.Fatal("execfam 册解析空——源文形态变更须同步本锚")
	}
	seen := map[string]bool{}
	var undrillable []string
	for _, k := range keys {
		tmpl := k[1]
		if seen[tmpl] {
			continue
		}
		seen[tmpl] = true
		shape, ok := legShapeOf(tmpl)
		if !ok || shape == shapeXvenue {
			undrillable = append(undrillable, tmpl)
		}
	}
	if len(undrillable) > 0 {
		t.Fatalf("execfam 在册模板无策略级演习产能径: %s——execDrillPass 全谱消费 strategy_ref 凭据"+
			"而 Drill.Run 对之拒载=probe 滞留 exec_drill_missing 至 armed_ttl 黑洞"+
			"（canon simdrill-gate-domain-coverage-mismatch：闸凭据消费域必须≤演习产能域）",
			strings.Join(undrillable, ","))
	}
}
