// symmatch.go —— broker 符号方言剥后缀比基（venue 自报表内寻禁猜名）。
// T-0408 自 internal/venue/onboard 归置本包：符号方言=MT5 venue 域知识单真源
// （符号名一律 SymbolList 实读禁猜——Exness=m 尾实拍），onboard 探测草案与
// sigdist 影子候选提名同用一匹配器（防冗余断言——禁平行第二套）。
package mt5

import "strings"

// quoteSuffixes broker 符号报价后缀谱（全大写面——u 已 ToUpper 比基；m 尾
// 方言含——长先剥）。尾段=XM '#' 族（O-41377——自报表实读集
// docs/能力账/xmglobal-survey-2026-11-26.json：168 hash 符=<BASE><QUOTE>#，
// quote=fiat 码+BTC；裸 '#' 尾=GOLD#/SILVER#/*Cash# 单名符——剥后须全等
// base 才命中，GOLD24-7# 剥 '#' 余 GOLD24-7≠GOLD 不误配）。
var quoteSuffixes = []string{
	"USDTM", "USDM", "USDT", "USDCM", "USDC", "USDZ", "USDD", "USD", "PERP", ".P",
	// XM '#' 族 quote 码裸尾（<BASE><QUOTE>#——BTCUSD# 剥 USD#→BTC、
	// ETHBTC# 剥 BTC#→ETH、GAUCNH# 剥 CNH#→GAU；码谱=自报表实读集
	// quote 全谱非穷举）；裸 '#' 殿后=单名符兜底剥（GOLD#→GOLD、
	// GOLD24-7#→GOLD24-7≠GOLD 不误配——剥后全等 base 才命中）。
	"AUD#", "CAD#", "CHF#", "CNH#", "DKK#", "EUR#", "GBP#", "HKD#",
	"HUF#", "JPY#", "MXN#", "NOK#", "NZD#", "PLN#", "SEK#", "SGD#",
	"TRY#", "USD#", "ZAR#", "BTC#",
	"#",
}

// QuoteSuffixes 后缀谱只读出件（T-0432 候选机造剥后缀复用——禁平行第二套
// 后缀表漂移）。
func QuoteSuffixes() []string { return quoteSuffixes }

// MatchSymbol broker 符号表寻 base 对应名（方言剥后缀比基——venue 自报表内寻，
// 禁猜名）。多命中取首（自报表序）。
func MatchSymbol(syms []string, base string) string {
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

// MatchedSymbols universe 各 base 命中的 broker 符号集（探测 SymbolParams 域）。
func MatchedSymbols(syms []string, universe []string) []string {
	var out []string
	for _, inst := range universe {
		base := strings.SplitN(inst, "-", 2)[0]
		if s := MatchSymbol(syms, base); s != "" {
			out = append(out, s)
		}
	}
	return out
}
