// shape_whitelist_test.go —— O-41387 陈废豁免回归哨（audit/1127-drill
// 钦定验收式正名常名落地）：drillUnsupported 册内任一条目若已获
// excell.LegShapeOf 腿形映射=豁免陈废——白名单语义=「显式裁定无可
// 演习腿形」，陈废不除对拍闸弱化（后撤映射者被残留豁免静默再放行）。
// 凡新增腿形忘清豁免=本件自红。
package simdrill

import (
	"testing"

	"o/internal/platform/excell"
)

func TestDrillUnsupportedNoStaleExempt(t *testing.T) {
	var stale []string
	for _, tmpl := range drillUnsupported {
		if shape, ok := excell.LegShapeOf(tmpl); ok {
			stale = append(stale, tmpl+"("+string(shape)+")")
		}
	}
	if len(stale) > 0 {
		t.Fatalf("drillUnsupported 陈废豁免——已获腿形须撤出: %v", stale)
	}
}
