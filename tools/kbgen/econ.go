// econ.go —— 经济经验库 schema 门禁（D-587 自审三缺口①）：模板校验+索引生成。
// 库面=docs/经济经验库/ 散文式条目（区别于工程库 frontmatter Schema v2）——
// 必填字段：标题（# 行）+「**判据挂接**」EA-xx 指针。索引=条目首行清单自动
// 生成（禁手编——同工程库 索引.md 先例）。
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// econMat 经济库条目机读面（检索物料原料）——frontmatter（schema v3）与
// 散文体双形态归一：fm 在取 fm，散文件取标题+match 段+正文。
type econMat struct {
	file     string // 文件名（含 .md——索引行原样沿用）
	slug     string // 文件名去 .md
	id       string // frontmatter id，缺省=slug
	kind     string
	status   string
	risk     string
	title    string   // 索引行口径=首行 TrimPrefix "# "（与旧 econScan 逐字一致）
	display  string   // 召回展示=fm summary → 首个 `# ` 行 → title 兜底
	keywords []string // fm keywords
	scope    []string // match: 段值展平（payoff_class/canonicals/venues 值）
	text     string   // 正文净文本（去标题行/match 段/空行）
}

const (
	econIndexName   = "索引.md"
	econEntriesName = "条目"
	econSkipNames   = "README.md|模板.md|待归纳.md|索引.md|裁定账.md"
	econRequiredFm  = "**判据挂接**"
	// econRequiredFixpoint 回改点强制字段（消化链建制票③——无回改点不入库）：
	// eval 词/kill trigger/配对校验闸/仓位面枚举值/semantic:<目标> 五域前缀枚举。
	econRequiredFixpoint = "**回改点**"
)

// econDirConst 可变（resolvePaths repo-root 锚定重写——同三基准先例防 cwd 绑死）。
var econDirConst = "../../docs/经济经验库"

// econFixpointKinds 回改点合法 kind 前缀域（机械类直译成闸/语义类→自动立修订票）。
var econFixpointKinds = []string{"eval:", "kill_trigger:", "combo_gate:", "sizing:", "semantic:"}

// econMatchKeys match: 段合法键域（坑签名，D-587④ 续段③）——与
// econkb.matchEntry 消费键一一对应；段内未知键=签名键名漂移（匹配端读不到
// 即静默失效）→ gen-econ 门禁报错禁入索引。
var econMatchKeys = map[string]bool{
	"payoff_class": true,
	"canonicals":   true,
	"venues":       true,
}

func econDirPath() string {
	if d := os.Getenv("O_ECONKB_DIR"); d != "" {
		return d
	}
	return econDirConst
}

// econEntriesDir 条目目（消化链建制票：根目条目迁 条目/ 子目；无子目=根目回落——
// 迁移期双形态兼容）。
func econEntriesDir(dir string) string {
	if st, err := os.Stat(filepath.Join(dir, econEntriesName)); err == nil && st.IsDir() {
		return filepath.Join(dir, econEntriesName)
	}
	return dir
}

// econTitleOf 首个 `# ` 标题行文本（索引行与检索物料 display 共用口径）。
func econTitleOf(content string) string {
	for _, ln := range strings.Split(content, "\n") {
		if strings.HasPrefix(ln, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(ln, "# "))
		}
	}
	return ""
}

// econMatchSegValues match: 段值展平——段界判定与 econValidateMatchSeg 同式
// （match: 顶格行起、至首个非缩进/空行止）；段内 `key: v1, v2` 值项剥壳取词。
func econMatchSegValues(content string) []string {
	var vals []string
	inSeg := false
	for _, ln := range strings.Split(content, "\n") {
		t := strings.TrimRight(ln, " \t")
		if !inSeg {
			if strings.HasPrefix(t, "match:") {
				inSeg = true
			}
			continue
		}
		if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			break
		}
		tt := strings.TrimSpace(t)
		if tt == "" {
			break
		}
		i := strings.Index(tt, ":")
		if i <= 0 {
			continue
		}
		v := tt[i+1:]
		v = strings.NewReplacer("[", " ", "]", " ", "'", " ", "\"", " ", ",", " ").Replace(v)
		for _, tok := range strings.Fields(v) {
			vals = append(vals, tok)
		}
	}
	return vals
}

// econBodyText 正文净文本：去 `#` 标题行、match: 段（含起行）、空行——
// **field** 行与散文俱留（召回低权匹配面）。
func econBodyText(content string) string {
	var b strings.Builder
	inSeg := false
	for _, ln := range strings.Split(content, "\n") {
		t := strings.TrimRight(ln, " \t")
		if strings.HasPrefix(t, "match:") && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			inSeg = true
			continue
		}
		if inSeg {
			if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
				inSeg = false
			} else {
				continue
			}
		}
		if strings.HasPrefix(strings.TrimSpace(ln), "#") {
			continue
		}
		if s := strings.TrimSpace(ln); s != "" {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(s)
		}
	}
	return b.String()
}

// econLoad 扫描经济库条目并抽取机读面：*.md 除 skip 名集；每文件校验标题+
// 判据挂接+回改点字段（无回改点不入库票面钉死）+match 段键域——与
// econScan 同闸同判，索引与检索物料同出一源不漂。
func econLoad(dir string) ([]econMat, error) {
	dir = econEntriesDir(dir)
	fis, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []econMat
	for _, fi := range fis {
		n := fi.Name()
		if fi.IsDir() || !strings.HasSuffix(n, ".md") || strings.Contains(econSkipNames, n) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			return nil, err
		}
		c := string(b)
		if !strings.Contains(c, econRequiredFm) {
			return nil, fmt.Errorf("econ schema: %s 缺 %s 字段", n, econRequiredFm)
		}
		if !econFixpointOK(c) {
			return nil, fmt.Errorf("econ schema: %s 缺 %s 或 kind 前缀非法（域=eval:|kill_trigger:|combo_gate:|sizing:|semantic:）", n, econRequiredFixpoint)
		}
		if err := econValidateMatchSeg(n, c); err != nil {
			return nil, err
		}
		// 索引行口径逐字保持旧 econScan：首行 TrimPrefix "# "（fm 条目首行=---
		// 时索引标题即 "---"——存量行为不动，检索物料 display 走更准的标题径）。
		firstLine, _, _ := strings.Cut(c, "\n")
		idxTitle := strings.TrimPrefix(strings.TrimSpace(firstLine), "# ")
		dispTitle := econTitleOf(c)
		if dispTitle == "" {
			dispTitle = idxTitle
		}
		if dispTitle == "" {
			dispTitle = strings.TrimSuffix(n, ".md")
		}
		m := econMat{
			file: n, slug: strings.TrimSuffix(n, ".md"), id: strings.TrimSuffix(n, ".md"),
			title: idxTitle, display: dispTitle,
			scope: econMatchSegValues(c), text: econBodyText(c),
		}
		// frontmatter（schema v3）在取 fm 字段——散文件 parse 败走缺省面。
		if e, perr := parse(b); perr == nil && e.ID != "" {
			m.id = e.ID
			m.kind, m.status, m.risk = e.Kind, e.Status, e.Risk
			m.keywords = e.Keywords
			if e.Summary != "" {
				m.display = e.Summary
			}
		}
		out = append(out, m)
	}
	return out, nil
}

// econFixpointOK 回改点值域校验：**回改点** 行存在且值以合法 kind 前缀起。
func econFixpointOK(content string) bool {
	for _, ln := range strings.Split(content, "\n") {
		if i := strings.Index(ln, econRequiredFixpoint); i >= 0 {
			v := strings.TrimSpace(strings.TrimLeft(ln[i+len(econRequiredFixpoint):], "：: "))
			for _, k := range econFixpointKinds {
				if strings.HasPrefix(v, k) {
					return true
				}
			}
		}
	}
	return false
}

// econValidateMatchSeg 正文 match: 段键域校验（段=顶格 `match:` 行至首个非缩进行）。
// 段内键 ∉ econMatchKeys → 报错（坑签名键名漂移防护——匹配端 econkb 只认三键，
// 写错键名=该签名永不命中且无告警，门禁单点拦）。
func econValidateMatchSeg(name, content string) error {
	inSeg := false
	for _, ln := range strings.Split(content, "\n") {
		t := strings.TrimRight(ln, " \t")
		if !inSeg {
			if strings.HasPrefix(t, "match:") {
				inSeg = true
			}
			continue
		}
		if !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			break // 段终
		}
		tt := strings.TrimSpace(t)
		if tt == "" {
			break
		}
		i := strings.Index(tt, ":")
		if i <= 0 {
			continue // 无键值形行（注释等）不属键域
		}
		key := strings.TrimSpace(tt[:i])
		if !econMatchKeys[key] {
			return fmt.Errorf("econ schema: %s match 段未知键 %q（合法域=payoff_class/canonicals/venues——与 econkb.matchEntry 消费键对齐）", name, key)
		}
	}
	return nil
}

// econGen 生成/校验 索引.md+检索物料.tsv（check 模式只比对不落盘）。
func econGen(check bool) error {
	dir := econDirPath()
	entries, err := econLoad(dir)
	if err != nil {
		return err
	}
	var titles []string
	for _, m := range entries {
		titles = append(titles, fmt.Sprintf("- %s —— `%s`", m.title, m.file))
	}
	sort.Strings(titles)
	var w strings.Builder
	w.WriteString("# 经济经验库·索引（noxkb gen-econ 自动生成禁手编）\n\n")
	for _, t := range titles {
		w.WriteString(t + "\n")
	}
	idx := filepath.Join(dir, econIndexName)
	material := renderEconMaterial(entries)
	if check {
		cur, err := os.ReadFile(idx)
		if err != nil || string(cur) != w.String() {
			return fmt.Errorf("经济经验库索引过期——跑 noxkb gen-econ 重生成")
		}
		// 检索物料=gitignored 再生物——在则须鲜，缺席合法（同 gen --check 律）。
		if ex, err := os.ReadFile(econMaterialFile(dir)); err == nil && string(ex) != string(material) {
			return fmt.Errorf("经济经验库检索物料过期——跑 noxkb gen-econ 重生成")
		}
		return nil
	}
	if err := writeIfChanged(idx, []byte(w.String())); err != nil {
		return err
	}
	return writeIfChanged(econMaterialFile(dir), material)
}
