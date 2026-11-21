// audit_1121_drill_test.go —— 审计场（2026-11-21 演习面）:drillUnsupported
// 白名单键唯一+册内登记——重复条目=合并工件冗余,豁免键不在 execfam 册=
// 死豁免（拼错名豁免静默失效,缺口面无人挡）。
// 显跑：AUDIT_RED=1 go test ./internal/trading/simdrill -run TestDrillUnsupported
package simdrill

import (
	"os"
	"testing"

	"o/internal/platform/execfam"
)

func TestDrillUnsupportedUniqueAndRegistered(t *testing.T) {
	if os.Getenv("AUDIT_RED") != "1" {
		t.Skip("audit 等价锚显跑")
	}
	seen := map[string]bool{}
	reg := map[string]bool{}
	for _, f := range execfam.All() {
		reg[f] = true
	}
	var dup, orphan []string
	for _, tmpl := range drillUnsupported {
		if seen[tmpl] {
			dup = append(dup, tmpl)
		}
		seen[tmpl] = true
		if !reg[tmpl] {
			orphan = append(orphan, tmpl)
		}
	}
	if len(dup) > 0 {
		t.Fatalf("drillUnsupported 白名单重复条目: %v", dup)
	}
	if len(orphan) > 0 {
		t.Fatalf("drillUnsupported 豁免键不在 execfam 册（死豁免）: %v", orphan)
	}
}
