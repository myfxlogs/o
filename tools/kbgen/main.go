// kbgen —— O 经验库索引生成器（nox tools/noxkb gen 面的迁移件；induct/LLM
// 归纳管线不搬——O 分期再议）。子命令：
//
//	kbgen gen            生成 索引.md+技能清单.md+检索物料.tsv（幂等：相同不写）
//	kbgen gen --check    只校验不落盘——索引/检索物料陈旧 exit 1（机检门禁用）
//	kbgen read <id>      读条目全文到 stdout
//
// 路径：自 cwd 上溯锚定 docs/<lib>/条目（repo root 标记——错 cwd 显式报错非静默
// 写丢）；O_KB_DIR env 覆写全部基准（工具引导参非业务配置）。
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	entriesDir      string
	indexFile       string
	skillsIndexFile string
	materialFile    string
)

const (
	maxLines    = 400    // 索引行数硬顶（~330 条容量，触顶再上浮——grep 检索面预算；T-0416 321→320→360/T-0456 361→400）
	maxBytes    = 196608 // 索引字节硬顶（<192KB grep 廉价面；T-0442归并日 131298 触顶上浮 128KB→192KB——T-0416 行顶同例）
	maxKeywords = 5
)

// Entry 对应条目 frontmatter（工程经验库 Schema v2——与 nox 同字段名直迁）。
type Entry struct {
	ID            string                   `yaml:"id"`
	Kind          string                   `yaml:"kind"`
	Risk          string                   `yaml:"risk"`
	Status        string                   `yaml:"status"`
	Keywords      []string                 `yaml:"keywords"`
	Scope         []string                 `yaml:"scope"`
	Summary       string                   `yaml:"summary"`
	SchemaVersion int                      `yaml:"schema_version"`
	FirstSeen     string                   `yaml:"first_seen"`
	LastVerified  string                   `yaml:"last_verified"`
	Skills        []string                 `yaml:"skills"`
	EvidenceRefs  []map[string]interface{} `yaml:"evidence_refs"`
	EALink        string                   `yaml:"ea_link"`
}

const header = `# O 工程经验索引

> 任务开始时读取一次；命中后读 docs/工程经验库/条目/<id>.md 全文（或 kbgen read <id>）。

| id | kind | keywords | summary | status | risk |
|----|------|----------|---------|--------|------|
`

// parse 解析单个条目文件的 frontmatter（--- YAML ---）。
func parse(data []byte) (*Entry, error) {
	content := string(data)
	fmStart := 0
	if strings.HasPrefix(content, "---\n") {
		fmStart = 4
	} else {
		idx := strings.Index(content, "---\n")
		if idx < 0 {
			return nil, fmt.Errorf("frontmatter opening not found")
		}
		fmStart = idx + 4
	}
	end := strings.Index(content[fmStart:], "\n---\n")
	if end < 0 {
		return nil, fmt.Errorf("frontmatter closing not found")
	}
	var e Entry
	if err := yaml.Unmarshal([]byte(content[fmStart:fmStart+end]), &e); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if e.ID == "" {
		return nil, fmt.Errorf("id is empty")
	}
	return &e, nil
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	for _, c := range []string{"\n", "\t", "\r"} {
		s = strings.ReplaceAll(s, c, " ")
	}
	return strings.TrimSpace(s)
}

func keywordsCell(kw []string) string {
	if len(kw) > maxKeywords {
		kw = kw[:maxKeywords]
	}
	return sanitize(strings.Join(kw, " "))
}

func renderRow(e *Entry) string {
	return fmt.Sprintf("| %s | %s | %s | %s | %s | %s |",
		sanitize(e.ID), sanitize(e.Kind), keywordsCell(e.Keywords),
		sanitize(e.Summary), sanitize(e.Status), sanitize(e.Risk))
}

func render(entries []*Entry) []byte {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	var buf bytes.Buffer
	buf.WriteString(header)
	for _, e := range entries {
		buf.WriteString(renderRow(e))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// loadEntries 扫描条目目录（顶层 .md——前身-nox/ 参考区不进索引）。
func loadEntries(dir string) ([]*Entry, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.md"))
	if err != nil {
		return nil, fmt.Errorf("glob: %w", err)
	}
	var entries []*Entry
	for _, m := range matches {
		data, err := os.ReadFile(m)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", m, err)
		}
		e, err := parse(data)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(m), err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func checkBudget(output []byte) error {
	if lines := bytes.Count(output, []byte{'\n'}); lines > maxLines {
		return fmt.Errorf("index %d lines exceeds max %d", lines, maxLines)
	}
	if len(output) > maxBytes {
		return fmt.Errorf("index %d bytes exceeds max %d", len(output), maxBytes)
	}
	return nil
}

func gen(check bool) error {
	entries, err := loadEntries(entriesDir)
	if err != nil {
		return err
	}
	out := render(entries)
	if err := checkBudget(out); err != nil {
		return err
	}
	material := renderMaterial(entries)
	if check {
		if ex, err := os.ReadFile(indexFile); err != nil || !bytes.Equal(ex, out) {
			return fmt.Errorf("index out of date")
		}
		if ex, err := os.ReadFile(skillsIndexFile); err != nil ||
			!bytes.Equal(ex, renderSkillsIndex(entries)) {
			return fmt.Errorf("skills index out of date")
		}
		// 检索物料=gitignored 再生物——在则须鲜（与索引同律），缺席合法
		// （kb_recall.sh 自愈径兜底重生，fail-open 召回不依赖它先在场）。
		if ex, err := os.ReadFile(materialFile); err == nil && !bytes.Equal(ex, material) {
			return fmt.Errorf("retrieval material out of date")
		}
		return nil
	}
	if err := writeIfChanged(indexFile, out); err != nil {
		return err
	}
	if err := writeIfChanged(skillsIndexFile, renderSkillsIndex(entries)); err != nil {
		return err
	}
	return writeIfChanged(materialFile, material)
}

// resolvePaths repo root 锚定：上溯找 docs/<lib>/条目 目录；O_KB_DIR 优先。
func resolvePaths(lib string) error {
	if v := os.Getenv("O_KB_DIR"); v != "" {
		entriesDir = filepath.Join(v, "条目")
		indexFile = filepath.Join(v, "索引.md")
		skillsIndexFile = filepath.Join(v, "技能清单.md")
		materialFile = filepath.Join(v, "检索物料.tsv")
		return nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getwd: %w", err)
	}
	for {
		base := filepath.Join(dir, "docs", lib)
		if st, err := os.Stat(filepath.Join(base, "条目")); err == nil && st.IsDir() {
			entriesDir = filepath.Join(base, "条目")
			indexFile = filepath.Join(base, "索引.md")
			skillsIndexFile = filepath.Join(base, "技能清单.md")
			materialFile = filepath.Join(base, "检索物料.tsv")
			return nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			return fmt.Errorf("kbgen: repo root not found (no docs/%s/条目 ancestor from cwd)", lib)
		} else {
			dir = parent
		}
	}
}

func main() {
	lib := flag.String("lib", "工程经验库", "库名（docs/<lib>/）——工程经验库|经济经验库")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kbgen [-lib 库名] gen [--check] | skills [--check] | read <id>")
		flag.PrintDefaults()
	}
	flag.Parse()
	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(2)
	}
	if err := resolvePaths(*lib); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var err error
	switch args[0] {
	case "gen":
		err = gen(len(args) > 1 && args[1] == "--check")
	case "gen-econ":
		err = econGen(len(args) > 1 && args[1] == "--check")
	case "econ-verdict":
		err = econVerdictCmd()
	case "skills":
		// skills 域锚定：repo root=entriesDir 上溯三级（docs/<lib>/条目）。
		root := filepath.Dir(filepath.Dir(filepath.Dir(entriesDir)))
		err = genSkills(filepath.Join(root, "skills"), filepath.Join(root, "skills", "索引.md"),
			len(args) > 1 && args[1] == "--check")
	case "read":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: kbgen read <id>")
			os.Exit(2)
		}
		var b []byte
		b, err = os.ReadFile(filepath.Join(entriesDir, args[1]+".md"))
		if err == nil {
			_, err = os.Stdout.Write(b)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
