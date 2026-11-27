// dialectname.go —— 方言名剥件单源（自 cmd/core/dialect_face.go 上收
// internal/mt5——onboard 普查件 base_guess/suffix_guess 与 cmd/core 候选
// 机造共用同一剥法，禁平行第二套）。判据锚不变：markers 档位标先剥、
// #尾 tradable 变体标次剥（O-35250 XM 实拍）、quote 层长先剥单源
// QuoteSuffixes；剥后余串=base、命中 quote=小写 suffix。
package mt5

import "strings"

// DialectMarkers 档位/型后缀（非报价——Exness .pro 实拍在案/期货 .P·PERP；
// 剥后续剥 quote——XAUUSD.pro=base XAUUSD+quote USD+标 .pro 两层方言）。
var DialectMarkers = []string{".PRO", ".P", "PERP"}

// SplitDialect 方言名两层剥后缀→(base, quote 后缀)。
func SplitDialect(sym string) (base, suffix string) {
	rest := strings.ToUpper(sym)
	for _, m := range DialectMarkers {
		if strings.HasSuffix(rest, m) {
			rest = rest[:len(rest)-len(m)]
			break
		}
	}
	rest = strings.TrimSuffix(rest, "#") // XM 方言：#尾=tradable 变体标
	for _, q := range QuoteSuffixes() {
		if q == "PERP" || q == ".P" {
			continue // 型后缀归 marker 层（本件两遍剥）
		}
		if strings.HasSuffix(rest, q) {
			return sym[:len(rest)-len(q)], strings.ToLower(q)
		}
	}
	return sym[:len(rest)], ""
}

// IsPerpSuffix perp 型后缀判（PERP/.P——期货永续方言面）。
func IsPerpSuffix(sym string) bool {
	u := strings.ToUpper(sym)
	return strings.HasSuffix(u, "PERP") || strings.HasSuffix(u, ".P")
}
