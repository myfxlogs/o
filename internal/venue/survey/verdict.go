// verdict.go —— 承接面判词+方言对拍机算面（O-41375——venue 符集 ∩ 各评估
// 族宇宙三档判词「能/不能/部分(子集N符)」；决策素材非决策）。族宇宙真源
// =fin_opp_eval 近 90d 实评 inst_id 集（评估族实际覆盖面——非声明册
// exec_venue 绑定列：本臂答「此 venue 能否承接」假想题，绑定与否归业主
// 决定面）。canon 名规范对照=canonKey 双端同函数归一（venue 符与族宇宙
// inst_id 同径剥缀——判据对称才可交）。
package survey

import (
	"context"
	"strings"

	"o/internal/mt5"
	"o/internal/platform/execfam"
)

// canonSuffixes 剥缀谱=mt5.QuoteSuffixes() 全表+单字母 M 尾（Exness 系
// FX 符 m 尾方言——'m' 居表尾最短后剥，长后缀先配不误伤）。
var canonSuffixes = append(append([]string{}, mt5.QuoteSuffixes()...), "M")

// canonKey 符号 canon 归一键（venue 符与族宇宙 inst_id 同函数——判据对称）：
// 大写化→去尾点→含'-'取首段（OKX 形取基底）→迭代剥缀至不动点。
// 判词原料非决策——碰撞/误剥风险落 map_draft 全量对照面可审计。
func canonKey(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ".")
	if i := strings.IndexByte(s, '-'); i > 0 {
		s = s[:i]
	}
	for {
		hit := false
		for _, suf := range canonSuffixes {
			if len(s) > len(suf) && strings.HasSuffix(s, suf) {
				s = s[:len(s)-len(suf)]
				hit = true
				break
			}
		}
		if !hit {
			return s
		}
	}
}

// dashless 全 dashless 判（族宇宙 CFD 形判据——全 inst_id 无 '-'=经纪商
// CFD 符名域可与 venue 符 canon 对拍；含 '-'=交易所合约型 venue 无对应
// 合约类=结构性不能）。
func dashless(insts []string) bool {
	for _, s := range insts {
		if strings.Contains(s, "-") {
			return false
		}
	}
	return len(insts) > 0
}

// familyUniverse 族宇宙=fin_opp_eval 近 90d 实评 distinct inst_id（评估
// 族实际覆盖面——eval 空窗族=宇宙未实证如实记不虚构）。
func (r *Runner) familyUniverse(ctx context.Context, family string) ([]string, error) {
	rows, err := r.md.Query(ctx, `SELECT DISTINCT inst_id FROM fin_opp_eval
		WHERE template=$1 AND ts>now()-interval '90 days' ORDER BY inst_id`, family)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// verdicts 承接面判词全族遍（manifest 册=族全集真源；判词三档进升件行
// 决策素材）。返 verdicts+counts+canon→venueSym 映射（map_draft 原料）。
func (r *Runner) verdicts(ctx context.Context, venueSyms []string) (map[string]any, map[string]int, map[string]string, error) {
	venueCanon := map[string]string{} // canon→venue 符（首中者——碰撞计入 draft 注记）
	for _, s := range venueSyms {
		k := canonKey(s)
		if k == "" {
			continue
		}
		if _, ok := venueCanon[k]; !ok {
			venueCanon[k] = s
		}
	}
	out := map[string]any{}
	counts := map[string]int{}
	cfdCanon := map[string]bool{} // dashless 族宇宙 canon 并集（map_draft 对照域）
	for _, decl := range execfam.FamilyManifest() {
		insts, err := r.familyUniverse(ctx, decl.Family)
		if err != nil {
			out[decl.Family] = map[string]any{"verdict": "读败", "err": err.Error()}
			counts["读败"]++
			continue
		}
		if len(insts) == 0 {
			out[decl.Family] = map[string]any{"verdict": "无宇宙实证"}
			counts["无宇宙实证"]++
			continue
		}
		if !dashless(insts) {
			out[decl.Family] = map[string]any{"verdict": "不能",
				"reason": "宇宙=交易所合约型——venue CFD 面无对应合约类", "total": len(insts)}
			counts["不能"]++
			continue
		}
		uc := map[string]bool{}
		for _, s := range insts {
			uc[canonKey(s)] = true
			cfdCanon[canonKey(s)] = true
		}
		covered := 0
		for k := range uc {
			if _, ok := venueCanon[k]; ok {
				covered++
			}
		}
		verdict := "部分"
		switch {
		case covered == len(uc):
			verdict = "能"
		case covered == 0:
			verdict = "不能"
		}
		out[decl.Family] = map[string]any{"verdict": verdict,
			"covered": covered, "total": len(uc)}
		counts[verdict]++
	}
	return out, counts, canonMap(venueCanon, cfdCanon), nil
}

// canonMap map_draft.pairs 原料——venue 符 canon ∈ CFD 族宇宙并集→
// {canon: venueSym} 对（canon=策略侧名做键——venue 符做值可回查）。
func canonMap(venueCanon map[string]string, want map[string]bool) map[string]string {
	out := map[string]string{}
	for k, v := range venueCanon {
		if want[k] {
			out[k] = v
		}
	}
	return out
}

// mapDraft 方言对拍草案（venue 符↔策略宇宙 canon 对照——unmatched 按类
// 分组计数非千行平铺；pairs 只载 canon 命中对）。
func mapDraft(venue string, syms []string, pairs map[string]string) map[string]any {
	paired := map[string]bool{}
	for _, v := range pairs {
		paired[v] = true
	}
	byClass := map[string]int{}
	var unmatchedSample []string
	for _, s := range syms {
		if paired[s] {
			continue
		}
		byClass[symClass(s)]++
		if len(unmatchedSample) < 30 {
			unmatchedSample = append(unmatchedSample, s)
		}
	}
	return map[string]any{"venue": venue, "generated_by": "venue-survey-arm",
		"pairs": pairs, "pairs_n": len(pairs), "unmatched_n": len(syms) - len(paired),
		"unmatched_by_class": byClass, "unmatched_sample": unmatchedSample}
}

// symClass 未配符粗分类（判词原料——fx 对/金属/指数数字/其他）。
func symClass(s string) string {
	k := canonKey(s)
	switch {
	case strings.HasPrefix(k, "XAU"), strings.HasPrefix(k, "XAG"),
		strings.HasPrefix(k, "XPT"), strings.HasPrefix(k, "XPD"):
		return "metal"
	case strings.ContainsAny(k, "0123456789"):
		return "index_numeric"
	case len(k) == 6 && alphaOnly(k):
		return "fx_pair"
	default:
		return "other"
	}
}

func alphaOnly(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return s != ""
}
