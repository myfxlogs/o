// symmatch.go —— broker 符号方言剥后缀比基（venue 自报表内寻禁猜名）。
package onboard

import "strings"

// ── 符号匹配 ─────────────────────────────────────────────────────────

// quoteSuffixes broker 符号报价后缀谱（全大写面——u 已 ToUpper 比基；m 尾
// 方言含——长先剥）。
var quoteSuffixes = []string{"USDTM", "USDM", "USDT", "USDCM", "USDC", "USDZ", "USDD", "USD", "PERP", ".P"}

// matchSymbol broker 符号表寻 base 对应名（方言剥后缀比基——venue 自报表内寻，
// 禁猜名）。多命中取首（自报表序）。
func matchSymbol(syms []string, base string) string {
	for _, s := range syms {
		u := strings.ToUpper(s)
		for _, q := range quoteSuffixes {
			if strings.HasSuffix(u, q) && strings.EqualFold(u[:len(u)-len(q)], base) {
				return s
			}
		}
	}
	return ""
}

// matchedSymbols universe 各 base 命中的 broker 符号集（探测 SymbolParams 域）。
func matchedSymbols(syms []string, universe []string) []string {
	var out []string
	for _, inst := range universe {
		base := strings.SplitN(inst, "-", 2)[0]
		if s := matchSymbol(syms, base); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func strSlice(v any) []string {
	xs, _ := v.([]any)
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}
