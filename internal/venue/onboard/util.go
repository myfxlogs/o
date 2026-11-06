// util.go —— onboard 杂件（evidence any→型安全取）。
// 符号方言匹配器已归置 internal/mt5（T-0408 单真源——mt5.MatchSymbol/
// MatchedSymbols，影子候选提名同用一匹配器）。
package onboard

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
