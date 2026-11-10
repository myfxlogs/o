// shape_align_test.go —— 执行族册 vs 演习腿形映射对齐机检：execfam
// 册内每模板须有已知腿形（legShapeOf 命中），或落在显式「演习不支持」
// 白名单内——新增族忘登映射=矩阵面缺口静默漏（本件把缺口前置到测试期）。
package simdrill

import (
	"testing"

	"o/internal/platform/execfam"
)

// drillUnsupported 演习不支持白名单——册内模板经显式裁定不入矩阵腿形
// 面者在此登记（空=零豁免：全册须有形）。新增豁免须同 commit 写明裁处。
var drillUnsupported = []string{}

func TestExecfamLegShapeAlign(t *testing.T) {
	exempt := map[string]bool{}
	for _, tmpl := range drillUnsupported {
		exempt[tmpl] = true
	}
	var unshaped []string
	for _, tmpl := range execfam.All() {
		if _, ok := legShapeOf(tmpl); !ok && !exempt[tmpl] {
			unshaped = append(unshaped, tmpl)
		}
	}
	if len(unshaped) > 0 {
		t.Fatalf("execfam 模板腿形未登记且不在演习不支持白名单: %v", unshaped)
	}
}
