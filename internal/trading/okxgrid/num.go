package okxgrid

import (
	"strconv"
	"strings"
)

// fmtNum venue 数字串方言（tradingBot 族参数全 string——docs-v5 取证钉）。
func fmtNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// parseF 宽松浮点解（venue 数值串/空串→0——okxexec.parseF 同语义本地件，
// 本包零依赖交易域内部）。
func parseF(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
