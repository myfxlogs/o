package main

// skills 子令（T-0052）：扫 skills/*/SKILL.md frontmatter 产 skills/索引.md——
// 「坑关键词→归口 skill」反向路由速查（harness 注入清单只答有无，歧义路由靠此表）。
// --check 漂移档：生成输出与库内 diff 不同=exit 非零（pre-commit 挂点消费）。
// 行数硬顶 60——索引自身不能成 token 负担。

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const skillsIndexMaxLines = 60

var latinToken = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_-]{2,}`)

type skillFM struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// parseSkillFM frontmatter（--- YAML ---）+首表头行（关键词补充源——venue/端点/坑键）。
func parseSkillFM(data []byte) (fm skillFM, firstHeader string, err error) {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return fm, "", fmt.Errorf("frontmatter opening not found")
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return fm, "", fmt.Errorf("frontmatter closing not found")
	}
	if err := yaml.Unmarshal([]byte(s[4:4+end]), &fm); err != nil {
		return fm, "", err
	}
	for _, line := range strings.Split(s[4+end:], "\n") {
		if strings.HasPrefix(line, "|") {
			firstHeader = strings.TrimSpace(line)
			break
		}
	}
	return fm, firstHeader, nil
}

// extractKeywords description+首表头抽拉丁 token（venue 名/端点名/坑键），去重封顶 5。
func extractKeywords(desc, header string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		t = strings.Trim(t, "-_")
		if len(t) < 3 || seen[t] {
			return
		}
		seen[t] = true
		out = append(out, t)
	}
	for _, t := range latinToken.FindAllString(desc, -1) {
		add(t)
	}
	for _, cell := range strings.Split(header, "|") {
		for _, t := range latinToken.FindAllString(cell, -1) {
			add(t)
		}
	}
	if len(out) > maxKeywords {
		out = out[:maxKeywords]
	}
	return out
}

// pitRefs 技能清单.md（kbgen gen 产出）反向解析：skill→条目 id 集（坑引源）。
func pitRefs() map[string][]string {
	refs := map[string][]string{}
	b, err := os.ReadFile(skillsIndexFile)
	if err != nil {
		return refs
	}
	cur := ""
	for _, line := range strings.Split(string(b), "\n") {
		if h, ok := strings.CutPrefix(line, "## "); ok {
			cur = strings.TrimSpace(h)
			continue
		}
		if cur == "" {
			continue
		}
		if id, ok := tableIDCell(line); ok {
			refs[cur] = append(refs[cur], id)
		}
	}
	return refs
}

func tableIDCell(line string) (string, bool) {
	if !strings.HasPrefix(line, "|") {
		return "", false
	}
	c := strings.Split(strings.Trim(line, "|"), "|")
	if len(c) == 0 {
		return "", false
	}
	id := strings.TrimSpace(c[0])
	if id == "id" || id == "" || strings.HasPrefix(id, "-") {
		return "", false
	}
	return id, true
}

const skillsHeader = `# skills 索引（kbgen skills 生成，勿手编）

> 反向路由：坑关键词→归口 skill；全文 skills/<域>/SKILL.md；坑引=工程经验库条目 id（kbgen read <id>）。缺描述=schema 哨兵。

| 域 | 覆盖 | 关键词 | 坑引 |
|----|------|--------|------|
`

// genSkills 扫描+渲染+落盘/--check。
func genSkills(skillsDir, outFile string, check bool) error {
	dirs, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}
	var names []string
	for _, d := range dirs {
		if d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
			names = append(names, d.Name())
		}
	}
	sort.Strings(names)
	refs := pitRefs()
	var buf bytes.Buffer
	buf.WriteString(skillsHeader)
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(skillsDir, name, "SKILL.md"))
		if err != nil {
			continue // 非 skill 目录（README 等）跳过
		}
		fm, header, perr := parseSkillFM(data)
		cover, kws := "缺描述", ""
		if perr == nil {
			cover = firstSentence(fm.Description)
			kws = strings.Join(extractKeywords(fm.Description, header), " ")
		}
		buf.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
			sanitize(name), sanitize(cover), sanitize(kws),
			sanitize(strings.Join(refs[name], " "))))
	}
	out := buf.Bytes()
	if n := bytes.Count(out, []byte("\n")); n > skillsIndexMaxLines {
		return fmt.Errorf("skills 索引超顶: %d>%d 行", n, skillsIndexMaxLines)
	}
	if check {
		ex, err := os.ReadFile(outFile)
		if err != nil || !bytes.Equal(ex, out) {
			return fmt.Errorf("skills index out of date（跑 kbgen skills 重建）")
		}
		return nil
	}
	return writeIfChanged(outFile, out)
}

func firstSentence(s string) string {
	if i := strings.IndexAny(s, "。；;"); i >= 0 {
		return s[:i]
	}
	if i := strings.Index(s, "——"); i >= 0 {
		return s[:i]
	}
	return s
}
