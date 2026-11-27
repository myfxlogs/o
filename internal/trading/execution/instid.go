// instid 乐器键族判据（O-41383 棘轮拆件自 sizing.go——instID 形态谓词
// 单源簇：SWAP 尾判=legset.IsSwapInst 收编面，datedFUT=-YYMMDD 期货键）。
package execution

import (
	"fmt"
	"regexp"
	"strings"

	"o/internal/platform/legset"
)

// DeriveSpotInst 派生现货乐器（去 "-SWAP" 尾——派生败 fail-closed 拒构）。
func DeriveSpotInst(instID string) (string, error) {
	if !legset.IsSwapInst(instID) {
		return "", fmt.Errorf("非 SWAP 乐器不可派生现货: %s", instID)
	}
	spot := strings.TrimSuffix(instID, "-SWAP")
	if spot == "" || spot == instID {
		return "", fmt.Errorf("派生现货败: %s", instID)
	}
	return spot, nil
}

// datedFUTRe 到期日期货键（-YYMMDD 尾——okxws/simdrill 同形）。
var datedFUTRe = regexp.MustCompile(`-\d{6}$`)

// contractInst 合约域乐器判据（SWAP 永续+datedFUT 到期期货——腿 sz 语义=
// 张数域，T-0483：legCash/legPx 旧判只排 -SWAP，datedFUT 误落 base 域产
// phantom spot_* 现金行+qty_base 张数误标币量）。
func contractInst(instID string) bool {
	return legset.IsSwapInst(instID) || datedFUTRe.MatchString(instID)
}
