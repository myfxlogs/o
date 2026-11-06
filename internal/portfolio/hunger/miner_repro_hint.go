// miner_repro_hint.go —— T-0388 载荷扩：可机判病灶类→pending 载荷
// repro_hint 桩建议文（修复票复现桩前置同族——矿工草案直带「怎么红怎么绿」
// 骨架，工人据以写真桩；建议文非可编译码防 LLM 自产未审桩直跑）。
package hunger

import "strings"

// minerReproLex 可机判病灶类词表（票面钦三类：种键缺/鲜度闸/域拒）——
// note/esc 词面等值命中即附类级桩建议文；多类同窗并见全携以「；」并。
var minerReproLex = []struct {
	cls  string
	keys []string
	hint string
}{
	{"种键缺", []string{"种键", "漏种", "键缺"},
		"桩建议[种键缺]:迁移/装配后不预埋该键直查断言红——补种后同断言绿(0382案形)"},
	{"鲜度闸", []string{"鲜度", "陈旧", "stale"},
		"桩建议[鲜度闸]:注入陈行/缺行断言闸拒红——fresh态同断言修后绿"},
	{"域拒", []string{"域拒", "写域", "域闸"},
		"桩建议[域拒]:未登记/owner键DMSet断言拒红——dm域键放行修后绿"},
}

// minerReproHint 簇行证据扫类词→建议文串（行序保类序，类内首中即定；
// 零命中=空串——omitempty 非机判类零载不硬塞）。
func minerReproHint(c *mineCluster) string {
	var out []string
	for _, e := range minerReproLex {
		hit := false
		for _, r := range c.rows {
			for _, k := range e.keys {
				if strings.Contains(r.note, k) ||
					strings.Contains(r.escClass, k) ||
					strings.Contains(r.escScope, k) {
					hit = true
					break
				}
			}
			if hit {
				break
			}
		}
		if hit {
			out = append(out, e.hint)
		}
	}
	return strings.Join(out, "；")
}
