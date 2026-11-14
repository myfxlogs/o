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

	"o/internal/platform/excell"
)

func auditRedGateLife2(t *testing.T) {
	t.Helper()
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit RED 锚——修复期显跑: AUDIT_RED=1")
	}
}

// TestAuditDrillDomainCoverage 不变量：凡可经 paper→probe→live 毕业链抵
// execDrillPass 闸的模板，必须有策略级演习凭据产能径（LegShapeOf 映到
// 非拒载腿形）。包向禁 import execfam——同源消费面=源文解析 execfam.set
// 册（注册语义=paper 结算域贯通，paper→probe→live 链路同域）。
// 破面原案：xvenue-carry 在册可 paper→probe，而 legShapeOf→excell.ShapeXvenue→
// Drill.Run 显式拒载——probe 滞留 exec_drill_missing 至 armed_ttl 黑洞；
// matrix 格凭据走 cell_key 键域与 strategy_ref 键域不接=产能域两键空。
// 修复形（升件#15291 消费域收窄——canon 钦定径①）：毕业闸在
// execDrillPass 消费前以 excell.Drillable 谓词拦不可钻模板
// （xvenue 产能只在格级键域，strategy_ref 凭据永不可得——谓词与
// Run 派路同出 LegShapeOf 单源）。断言三件：
// ①paper.go probe→live 段含 Drillable 收窄谓词（消费域收窄在案——
//
//	谓词缺席=闸回退全谱消费，任何在册不可钻模板即成黑洞）；
//
// ②execfam 册内每键 LegShapeOf 有映射（矩阵 fail-closed 覆盖锚——
//
//	未登记=格凭据 unshaped fail+闸谓词挡=册缺口死径仍抓）；
//
// ③Drillable 与 LegShapeOf 派路一致（谓词漂移=消费域/产能域再裂）。
func TestAuditDrillDomainCoverage(t *testing.T) {
	auditRedGateLife2(t)
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(file))))
	src, err := os.ReadFile(filepath.Join(root, "internal/platform/execfam/execfam.go"))
	if err != nil {
		t.Fatal(err)
	}
	// ① 收窄锚：毕业闸消费凭据前必以 Drillable 谓词拦拒载族（源文断言
	// 同 execfam 册解析律——谓词失位=消费域无界回退）。
	psrc, err := os.ReadFile(filepath.Join(root, "internal/portfolio/lifecycle/paper.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(psrc), "excell.Drillable(") {
		t.Fatal("execDrillPass 消费域无界——paper.go probe→live 段未见 excell.Drillable 收窄谓词" +
			"（升件#15291 修复件失位：消费域>产能域=probe 滞留黑洞回退）")
	}
	// 册解析：var set 字面量内 "tmpl": true 键集（注册域=execfam 唯一真源）。
	keys := regexp.MustCompile(`"([a-z_0-9-]+)"\s*:\s*true`).FindAllStringSubmatch(string(src), -1)
	if len(keys) == 0 {
		t.Fatal("execfam 册解析空——源文形态变更须同步本锚")
	}
	seen := map[string]bool{}
	for _, k := range keys {
		tmpl := k[1]
		if seen[tmpl] {
			continue
		}
		seen[tmpl] = true
		// ② 册覆盖：在册模板须腿形登记——未登记则矩阵 unshaped fail+
		// 闸谓词恒挡=死径占位（新增执行族注册须同 commit 补演习格）。
		shape, ok := excell.LegShapeOf(tmpl)
		if !ok {
			t.Errorf("execfam 在册模板腿形未登记（矩阵 unshaped fail+毕业闸外=死径占位）: %s", tmpl)
			continue
		}
		// ③ 谓词同出：Drillable⇔映到 Drill.Run 可产凭据腿形（excell.ShapeXvenue
		// =格级键域产能——策略级谓词恒 false 消费域外）。
		if excell.Drillable(tmpl) != (shape != excell.ShapeXvenue) {
			t.Errorf("Drillable 与 LegShapeOf 派路分歧: %s shape=%s（消费域收窄谓词须与 Run 派路同出）", tmpl, shape)
		}
	}
}
