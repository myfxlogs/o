// symmatch_test.go —— O-41377 刀2 锚：XM '#' 族后缀对拍（册源=
// docs/能力账/xmglobal-survey-2026-11-26.json 自报表实读集——BTCUSD#→BTC
// 命中；GOLD24-7# 不误配 GOLD=24-7 变体不剥；裸 '#' 单名符 GOLD#→GOLD）。
package mt5

import "testing"

func TestMatchSymbolHashFamily(t *testing.T) {
	syms := []string{
		"BTCUSD#", "ETHBTC#", "GAUCNH#", "GOLD#", "GOLD24-7#",
		"UK100Cash#", "ChinaInternet#", "OIL-JAN27", "US100-DEC26",
		"BTCUSDm", "EURUSD",
	}
	for _, c := range []struct {
		base, want string
	}{
		{"BTC", "BTCUSD#"}, // <BASE>USD# 剥 quote 码→base 命中（spec 锚①）
		{"ETH", "ETHBTC#"}, // quote=BTC 族同剥
		{"GAU", "GAUCNH#"}, // quote=CNH 族
		{"GOLD", "GOLD#"},  // 裸 '#' 单名符剥尾→GOLD（表序 GOLD# 先于 GOLD24-7#）
		{"XAU", ""},        // GOLD# 剥 '#' 余 GOLD≠XAU——不误配别名
		{"UK100", ""},      // UK100Cash# 剥 '#' 余 UK100Cash≠UK100——Cash 不剥
		{"OIL", ""},        // -MONYY 期货名不在 '#' 族——不剥不猜
		{"US100", ""},      // 同上
		{"EUR", "EURUSD"},  // 存量 USD 尾方言回归（无 # 形不受影响）
	} {
		if got := MatchSymbol(syms, c.base); got != c.want {
			t.Fatalf("MatchSymbol(%v,%q)=%q 应 %q", syms, c.base, got, c.want)
		}
	}
	// spec 锚②独立面：GOLD24-7# 对 base GOLD 必空（24-7 变体非 GOLD 本体）。
	if got := MatchSymbol([]string{"GOLD24-7#"}, "GOLD"); got != "" {
		t.Fatalf("GOLD24-7# 误配 GOLD=%q（24-7 后缀不剥钉死）", got)
	}
	// 长后缀优先钉：quote 码尾先于裸 '#' 尾尝试——BTCUSD# 剥 USD# 得 BTC
	// （若裸 '#' 先吃余串 BTCUSD≠BTC 自然落空续试，双序均正确；此锚锁语义）。
	if got := MatchSymbol([]string{"BTCUSD#"}, "BTCUSD"); got != "BTCUSD#" {
		t.Fatalf("base=BTCUSD 全等剥 '#' 应命中 got=%q", got)
	}
}
