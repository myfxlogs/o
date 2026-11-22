// instfamily.go —— inst 键族路由（工单O-33377④——287249 幻影退役根修）：
// 「venue 不存在」判词的前提=核验打对了族。DCD 产品键（base-quote-YYMMDD-
// strike-C/P，如 BTC-USDT-261204-66000-P）在 sfp/dcd 产品册不在
// /public/instruments 衍生品仪器册——跨族查空=结构性假阴性非不存在。
// 族判据钉死：键形→族单射，判词行必须携 CheckedFamily 标识。
package okxexec

import (
	"fmt"
	"regexp"
)

// dcdProductKeyRe DCD 产品键形（T-0101 实证族——BTC-USDT-261204-66000-P：
// base-quote-YYMMDD 行权价-C/P；SIM- 前缀夹具键同形同族——夹具泄漏是
// 另一病灶 #22844 立案，本判据只管族归属不管真伪）。
var dcdProductKeyRe = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+-[0-9]{6}-[0-9]+-[CP]$`)

// optionKeyRe 期权合约键形（OKX OPTION 族——BTC-USD_UM-261124-68000-C：
// instFamily 实键=BASE-QUOTE_SETTLE-YYMMDD-行权价-C/P，settle 段带
// _UM/_U 本位标记；无 settle 段的 USD 计价期权同族判 OPTION）。
var optionKeyRe = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+(_[A-Z]+)?-[0-9]{6}-[0-9]+-[CP]$`)

// datedFutKeyRe 到期日期货键形（-YYMMDD 尾六键——FUTURES 族）。
var datedFutKeyRe = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+-[0-9]{6}$`)

// spotKeyRe 现货键形（BASE-QUOTE 两段裸币对——带日期戳/-SWAP/-C/-P 尾
// 的键已被上支拦截，此处只认严格两段形防 SIM-/多段畸形键落 spot 兜底）。
var spotKeyRe = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+$`)

// InstFamily inst 键→venue 产品族判（族错=核验前提错——287249 案
// /public/instruments 查 DCD 键=族错查空判不存在）。族域闭集：
// dcd=sfp/dcd 产品册（order-history/order-status 面）；option=OPTION
// 仪器册（instFamily 分族）；futures=datedFUT；swap=perp；spot=现货；
// unknown=键形不入册（判词须标 unknown 不许猜族）。
func InstFamily(instID string) string {
	switch {
	case dcdProductKeyRe.MatchString(instID):
		return "dcd"
	case optionKeyRe.MatchString(instID):
		return "option"
	case datedFutKeyRe.MatchString(instID):
		return "futures"
	case len(instID) > 5 && instID[len(instID)-5:] == "-SWAP":
		return "swap"
	case spotKeyRe.MatchString(instID):
		return "spot"
	default:
		return "unknown"
	}
}

// DcdProductKey DCD 产品键判（族路由入口件——判词/核验径先过此闸再选面）。
func DcdProductKey(instID string) bool { return InstFamily(instID) == "dcd" }

// FamilyCheckNote 「venue 不存在」判词族标识后缀（O-33377④ 钦定——
// 任何不存在判词行必须携所查族；判词写手拼 note 时调用，族=核验实查面
// 非键形猜——InstFamily 给键形族，实查面族由核验径回填）。
func FamilyCheckNote(family string) string {
	return fmt.Sprintf("[核验族=%s]", family)
}
