// doc.go —— 普查判词文档机写面（O-41375——docs/能力账/<venue>-survey-
// <date>.md 自动生成，机写判词非人工文）。落点=O_VENUE_SURVEY_DIR env→
// 缺省 $HOME/o-surveys（OCI 无仓 checkout 实证——判词全文另存
// evidence.survey 可由 JSON 再生，doc_path 落证据行+升件行双引）。
package survey

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// docDir 判词文档落点（env 覆写优先——部署面可指仓内径）。
func docDir() string {
	if d := os.Getenv("O_VENUE_SURVEY_DIR"); d != "" {
		return d
	}
	if h, err := os.UserHomeDir(); err == nil {
		return filepath.Join(h, "o-surveys")
	}
	return "surveys"
}

// writeDoc 判词 md 落盘+返路径（写败=err 上浮由调用方记 evidence——
// 判词数据本体已在 evidence 不丢，仅文档镜像缺）。
func writeDoc(venue string, d *surveyData, now time.Time) (string, error) {
	dir := docDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("%s-survey-%s.md",
		venue, now.UTC().Format("2006-01-02")))
	return path, os.WriteFile(path, []byte(surveyMarkdown(venue, d, now)), 0o644)
}

// surveyMarkdown 判词文档本体（机写——T-0099 能力账文档同构：账户面+
// 符号谱+spread/swap/margin 三表+逐族判词+map_draft 摘要）。
func surveyMarkdown(venue string, d *surveyData, now time.Time) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s 上岗普查判词（机写——venue-survey 臂 %s）\n\n",
		venue, now.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- 符号总数: %d（partial=%v）\n", len(d.Symbols), d.Partial)
	acc := d.Account
	fmt.Fprintf(&b, "- 户头: login=%v lev=%v ccy=%v equity=%.2f free_margin=%.2v balance=%.2v\n\n",
		acc["login"], acc["leverage"], acc["ccy"], numF(acc["equity"]),
		numF(acc["free_margin"]), numF(acc["balance"]))
	b.WriteString("## 抽样报价活性\n\n")
	qh := d.QuoteHealth
	fmt.Fprintf(&b, "- sampled=%v alive=%v dead_pct=%.3f grade=%v\n\n",
		qh["sampled"], qh["alive"], numF(qh["dead_pct"]), qh["grade"])
	b.WriteString("## spread 谱（spread_pts）\n\n")
	sp := d.Spread
	fmt.Fprintf(&b, "- n=%v min=%v p50=%v p90=%v max=%v zero=%v\n\n",
		sp["n"], sp["min"], sp["p50"], sp["p90"], sp["max"], sp["zero"])
	b.WriteString("## swap 三值分布\n\n")
	sw := d.Swap
	fmt.Fprintf(&b, "- type=%v\n- triple_day=%v\n- long_nonzero=%v short_nonzero=%v\n\n",
		kvSorted(sw["type"]), kvSorted(sw["triple_day"]),
		sw["long_nonzero"], sw["short_nonzero"])
	b.WriteString("## margin_1lot 分层（对权益）\n\n")
	mt := d.MarginTiers
	fmt.Fprintf(&b, "- ≤$100: %v | ≤$500: %v | ≤权益50%%: %v | >: %v | 无数据: %v（equity=%.2f）\n\n",
		mt["le_100"], mt["le_500"], mt["le_eq50"], mt["gt"], mt["no_data"], numF(mt["equity"]))
	b.WriteString("## 承接面判词（族宇宙∩venue 符集——决策素材非决策）\n\n")
	b.WriteString("| 族 | 判词 | 覆盖/总数 |\n|---|---|---|\n")
	fams := make([]string, 0, len(d.FamilyVerdict))
	for f := range d.FamilyVerdict {
		fams = append(fams, f)
	}
	sort.Strings(fams)
	for _, f := range fams {
		v := d.FamilyVerdict[f].(map[string]any)
		cov := ""
		if v["covered"] != nil {
			cov = fmt.Sprintf("%v/%v", v["covered"], v["total"])
		}
		reason := ""
		if v["reason"] != nil {
			reason = "（" + fmt.Sprint(v["reason"]) + "）"
		}
		fmt.Fprintf(&b, "| %s | %v%s | %s |\n", f, v["verdict"], reason, cov)
	}
	b.WriteString("\n## map_draft 摘要\n\n")
	md := d.MapDraft
	fmt.Fprintf(&b, "- pairs=%v unmatched=%v by_class=%v\n",
		md["pairs_n"], md["unmatched_n"], kvSorted(md["unmatched_by_class"]))
	return b.String()
}

// numF any→float64 显示（判词文档用）。
func numF(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case int:
		return float64(x)
	}
	return 0
}

// kvSorted map[string]any/int 计数表排序渲染（k=v 逗串——判词行稳定序）。
func kvSorted(v any) string {
	m, ok := v.(map[string]int)
	if !ok {
		if mm, ok2 := v.(map[string]any); ok2 {
			out := make(map[string]int, len(mm))
			for k, x := range mm {
				out[k] = int(numF(x))
			}
			m = out
		} else {
			return fmt.Sprint(v)
		}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, m[k]))
	}
	return strings.Join(parts, ",")
}
