// verify.go —— 符号硬验判据单源（O-41378：手工 verifySymbolMapping 与
// onboard 自动 mint 拍共用同一把尺子——判据字面继承禁平行第二套；
// 上收自 cmd/core/dialect_{face,submit}.go 私有件）。
package mt5

import "strings"

// SpreadCapBPS 点差合理带上限（硬验拒收带——bps；超带=数据病/死市快照，
// 拒收非警告）。保守宽带上限——正常主流 CFD 全谱远低于此。
const SpreadCapBPS = 5000

// DollarPegged 美元锚定计价族（计价币对拍判据——CFD 计价方言：broker 自报
// ProfitCurrency USD/USDT/USDC 经济同族（美元锚定），官名 quote 直等禁
// （BTCUSDm profit=USD vs 官名 quote=USDT 字面异经济同）；族外=拒）。
func DollarPegged(ccy string) bool {
	switch strings.ToUpper(strings.TrimSpace(ccy)) {
	case "USD", "USDT", "USDC":
		return true
	}
	return false
}

// ApproxEq 相对容差等（ctVal 对拍——浮点直等禁，1e-9 相对容差；
// 单位错配病防线原点：XAU-USDT-SWAP=0.001oz vs XM GOLD#=100oz 同名不同券）。
func ApproxEq(a, b float64) bool {
	if a == b {
		return true
	}
	d := a - b
	if d < 0 {
		d = -d
	}
	m := a
	if b > m {
		m = b
	}
	return d <= m*1e-9
}
