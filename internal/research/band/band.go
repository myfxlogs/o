// band.go —— 族级阈两级档注册表（T-0353 件①——业主裁 B「中间档只放近确定
// 族」附钉①的码面承载）：阈存址随现有 per-family config 键机制（basis/dcd/
// funding/volsell 各族一键——评估器原读径零改），本件只做两件事：
//  1. 族→阈键→档位的显式可枚举映射（禁止散点硬编码——审计即对表）；
//  2. boot 审计闸（cmd/research 装配调用）：白名单外族键值 < Default 档
//     =红线漂移（directional 族溜进中间档）——Alert 显形不静默。
package band

import "sort"

// 两级档位（业主裁 B 锚——4.84% 期差基差错杀根修；config 值=唯一活值，
// 本件常量=审计基准非评估判据，评估器照旧吃 config）。
const (
	Middle  = 0.04 // 近确定 carry 族档（quarter_basis_long/perp_quarter_spread/dcd_call/dcd_put）
	Default = 0.06 // 纯阈档（funding-hedge 及未列族不变）
)

// NearCertain 近确定族白名单（收敛型 carry——到期/结算收敛是事件非方向
// 赌注；票面钦定闭集，新增族须业主裁随票扩册，禁施工自扩）。
var NearCertain = map[string]bool{
	"quarter_basis_long":  true,
	"perp_quarter_spread": true,
	"dcd_call":            true,
	"dcd_put":             true,
}

// FamilyKeys 族→阈键枚举（同族共键如实同址——basis 双族共 research.basis.
// min_net_apr、dcd 双族共 research.dcd.min_apr；vol-sell 两族 T-0353 起与
// dcd 解耦自持键——min_apr 降档不再连带卖权族）。
var FamilyKeys = map[string]string{
	"funding-hedge":        "research.funding.min_net_apr",
	"quarter_basis_long":   "research.basis.min_net_apr",
	"perp_quarter_spread":  "research.basis.min_net_apr",
	"dcd_call":             "research.dcd.min_apr",
	"dcd_put":              "research.dcd.min_apr",
	"vol_sell_single_call": "research.volsell.min_apr",
	"vol_sell_strangle":    "research.volsell.min_apr",
}

// Audit 两级档红线审计：返回违规族列表（空=净）。
//   - 白名单族键值 < Middle = 低于钦定档（配置手滑或副本收割回退）；
//   - 白名单外族键值 < Default = directional 族溜进中间档（附钉①红线）。
//
// 键值读败不计违规（缺键=fail-closed 域，评估器侧已挡——审计只盯「有值但
// 越档」的漂移面）。
func Audit(get func(string) (float64, error)) []string {
	type seen = struct {
		key string
		val float64
	}
	byKey := map[string]seen{}
	for _, key := range FamilyKeys {
		v, err := get(key)
		if err != nil {
			continue
		}
		byKey[key] = seen{key, v}
	}
	var bad []string
	for fam, key := range FamilyKeys {
		s, ok := byKey[key]
		if !ok {
			continue
		}
		floor := Default
		if NearCertain[fam] {
			floor = Middle
		}
		if s.val < floor {
			bad = append(bad, fam)
		}
	}
	sort.Strings(bad)
	return bad
}
