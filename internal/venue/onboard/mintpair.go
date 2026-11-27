// mintpair.go —— O-41378 件3 配对纯函数面（census 行×registry 索引——
// 零 venue IO、零池读；判词全在表上字段，拉数与判数分拍钉死）。
package onboard

import (
	"fmt"
	"sort"
	"strings"

	"o/internal/mt5"
)

// censusMintRow 普查行配对面（SELECT 投影——判词全在表上字段）。
type censusMintRow struct {
	VenueSymbol string
	BaseGuess   string
	SuffixGuess string
	FamilyGuess string
	CtVal       *float64
	TickSz      *float64
	MinSz       *float64
	LotStep     *float64
	Digits      *int32
	TradeMode   *int32
	ProfitCcy   *string
	SpecHash    *string
}

// registryAliasIndex 别名→登记行索引（aliases 大写归一——配对键=表上纯
// 函数料；同别名撞多 canonical=合法并存，歧义由 pairCensusRegistry 判）。
func registryAliasIndex(reg []mt5.RegistryRow) map[string][]mt5.RegistryRow {
	idx := map[string][]mt5.RegistryRow{}
	for _, row := range reg {
		for _, a := range row.Aliases {
			k := strings.ToUpper(strings.TrimSpace(a))
			if k == "" {
				continue
			}
			idx[k] = append(idx[k], row)
		}
	}
	return idx
}

// pairCensusRegistry 配对纯函数（census 行×registry 索引——零 venue IO；
// 返回 (nil,reason)=flag 判词 / (row,"")=命中待参数对拍）。
// 候选键=base∪base+suffix∪base+suffix(m尾剥) 三型——别名显式字典对拍，
// 禁模糊匹配；多 canonical 命中=alias_ambiguous 如实 flag 不猜。
func pairCensusRegistry(m censusMintRow, idx map[string][]mt5.RegistryRow) (*mt5.RegistryRow, string) {
	keys := matchKeys(m.BaseGuess, m.SuffixGuess)
	if len(keys) == 0 {
		return nil, "no_base"
	}
	hits := map[string]mt5.RegistryRow{}
	var order []string
	for _, k := range keys {
		for _, row := range idx[k] {
			if _, ok := hits[row.Canonical]; !ok {
				hits[row.Canonical] = row
				order = append(order, row.Canonical)
			}
		}
	}
	if len(hits) == 0 {
		return nil, "unregistered:" + strings.Join(keys, "/")
	}
	if len(hits) > 1 {
		sort.Strings(order)
		return nil, "alias_ambiguous:" + strings.Join(order, ",")
	}
	row := hits[order[0]]
	// family 同域收窄（family_guess 只收窄匹配域非落闸——other/空=不收窄）。
	fg := strings.ToLower(strings.TrimSpace(m.FamilyGuess))
	if fg != "" && fg != "other" && fg != strings.ToLower(row.Family) && row.Family != "*" {
		return nil, fmt.Sprintf("family_mismatch:%s_vs_%s", fg, row.Family)
	}
	return &row, ""
}

// matchKeys 配对候选键集（大写归一去重——base 单名形∪base+quote 对形；
// suffix m尾剥=mini 方言标（USDM→USD→XAUUSDm 命中 XAUUSD 别名））。
func matchKeys(base, suffix string) []string {
	b := strings.ToUpper(strings.TrimSpace(base))
	s := strings.ToUpper(strings.TrimSpace(suffix))
	if b == "" {
		return nil
	}
	keys := []string{b}
	if s != "" {
		keys = append(keys, b+s)
		if strings.HasSuffix(s, "M") && len(s) > 1 {
			keys = append(keys, b+s[:len(s)-1])
		}
	}
	// 去重保序
	seen := map[string]bool{}
	out := keys[:0]
	for _, k := range keys {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	return out
}

// censusParamCheck census 存量参数 vs registry spec 对拍（表上纯函数——
// 判据 mt5.* 单源同手工 verifySymbolMapping 尺：ctVal approxEq/
// profit_ccy 美元锚定族/digits>0/trade_mode=4/spec_hash 在案；
// 返回 ""=过 / reason=flag）。
func censusParamCheck(m censusMintRow, reg mt5.RegistryRow) string {
	if m.CtVal == nil {
		return "params_missing:ct_val"
	}
	if !mt5.ApproxEq(*m.CtVal, reg.CtVal) {
		return fmt.Sprintf("ctval_mismatch:%g_vs_%g", *m.CtVal, reg.CtVal)
	}
	if m.ProfitCcy == nil || !mt5.DollarPegged(*m.ProfitCcy) {
		pc := ""
		if m.ProfitCcy != nil {
			pc = *m.ProfitCcy
		}
		return "profit_ccy_reject:" + pc
	}
	if m.Digits == nil || *m.Digits <= 0 {
		return "digits_invalid"
	}
	if m.TradeMode == nil || *m.TradeMode != 4 {
		return "trade_mode_reject"
	}
	if m.SpecHash == nil || *m.SpecHash == "" {
		return "params_missing:spec_hash" // 漂移探针⑤对拍基线缺——落行即产漂移误报
	}
	return ""
}
