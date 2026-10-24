// kbgen —— PHASE 经验库索引生成器（nox tools/noxkb gen 面的迁移件；induct/LLM
// 归纳管线不搬——PHASE 分期再议）。子命令：
//
//	kbgen gen            生成 索引.md+技能清单.md（幂等：相同不写）
//	kbgen gen --check    只校验不落盘——索引陈旧 exit 1（机检门禁用）
//	kbgen read <id>      读条目全文到 stdout
//
// 路径：自 cwd 上溯锚定 docs/<lib>/条目（repo root 标记——错 cwd 显式报错非静默
// 写丢）；PHASE_KB_DIR env 覆写全部基准（工具引导参非业务配置）。
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
)

const (
	maxLines    = 320    // 索引行数硬顶（~300 条容量，触顶再上浮——grep 检索面预算）
	maxBytes    = 131072 // 索引字节硬顶（<128KB grep 廉价面）
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

const header = `# PHASE 工程经验索引

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

// renderSkillsIndex 按条目 skills: 标签生成关联清单（指针式——条目唯一真相源）。
func renderSkillsIndex(entries []*Entry) []byte {
	groups := map[string][]*Entry{}
	var tags []string
	for _, e := range entries {
		for _, s := range e.Skills {
			if s = strings.TrimSpace(s); s == "" {
				continue
			}
			if _, ok := groups[s]; !ok {
				tags = append(tags, s)
			}
			groups[s] = append(groups[s], e)
		}
	}
	sort.Strings(tags)
	var buf bytes.Buffer
	buf.WriteString("# 技能关联条目清单（kbgen gen 生成，勿手编）\n\n")
	buf.WriteString("> 按条目 frontmatter `skills:` 标签归组——清单是指针，条目全文见 条目/<id>.md。\n\n")
	for _, tag := range tags {
		buf.WriteString("## " + sanitize(tag) + "\n\n| id | summary |\n|----|---------|\n")
		es := groups[tag]
		sort.Slice(es, func(i, j int) bool { return es[i].ID < es[j].ID })
		for _, e := range es {
			buf.WriteString(fmt.Sprintf("| %s | %s |\n", sanitize(e.ID), sanitize(e.Summary)))
		}
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

func writeIfChanged(path string, out []byte) error {
	if ex, err := os.ReadFile(path); err == nil && bytes.Equal(ex, out) {
		return nil
	}
	return os.WriteFile(path, out, 0644)
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
	if check {
		if ex, err := os.ReadFile(indexFile); err != nil || !bytes.Equal(ex, out) {
			return fmt.Errorf("index out of date")
		}
		if ex, err := os.ReadFile(skillsIndexFile); err != nil ||
			!bytes.Equal(ex, renderSkillsIndex(entries)) {
			return fmt.Errorf("skills index out of date")
		}
		return nil
	}
	if err := writeIfChanged(indexFile, out); err != nil {
		return err
	}
	return writeIfChanged(skillsIndexFile, renderSkillsIndex(entries))
}

// resolvePaths repo root 锚定：上溯找 docs/<lib>/条目 目录；PHASE_KB_DIR 优先。
func resolvePaths(lib string) error {
	if v := os.Getenv("PHASE_KB_DIR"); v != "" {
		entriesDir = filepath.Join(v, "条目")
		indexFile = filepath.Join(v, "索引.md")
		skillsIndexFile = filepath.Join(v, "技能清单.md")
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
