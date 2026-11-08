// material.go —— 检索物料件生成（提案-KB向量检索-经验库召回面 首阶①）：
// 条目 frontmatter/正文字段抽取成机读 TSV——召回件 scripts/meta/kb_recall.sh
// 的唯一输入面，与索引同律入库再生物（gen 产、--check 在则验鲜、缺席合法
// ——gitignored，召回自愈径兜底重生，禁手编）。列：
//
//	lib id kind status risk display text keywords scope skills
//
// 多值字段（keywords/scope/skills）以 \x1f 连接；display=展示+低权匹配面，
// text=补充可匹配文本（工程库空、经济库散文正文）。
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const materialSep = "\x1f"

const materialHeader = `# 检索物料（kbgen gen 自动生成禁手编）——TSV 列：lib id kind status risk display text keywords scope skills；keywords/scope/skills 为 ` + "\\x1f" + ` 连接多值
`

// sanitizeTSV 机读面净化：剥离列分隔符/换行/多值分隔符（防字段错位）。
func sanitizeTSV(s string) string {
	for _, c := range []string{"\t", "\n", "\r", materialSep} {
		s = strings.ReplaceAll(s, c, " ")
	}
	return strings.TrimSpace(s)
}

func joinItems(items []string) string {
	var out []string
	for _, s := range items {
		if s = sanitizeTSV(s); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, materialSep)
}

// renderMaterial 工程库检索物料：display=summary，text 留空（summary 即低权
// 匹配面——提案钦定「summary+keywords 字段即可」不取正文）。
func renderMaterial(entries []*Entry) []byte {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	var buf bytes.Buffer
	buf.WriteString(materialHeader)
	for _, e := range entries {
		buf.WriteString(strings.Join([]string{
			"eng", sanitizeTSV(e.ID), sanitizeTSV(e.Kind), sanitizeTSV(e.Status),
			sanitizeTSV(e.Risk), sanitizeTSV(e.Summary), "",
			joinItems(e.Keywords), joinItems(e.Scope), joinItems(e.Skills),
		}, "\t"))
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// econMaterialFile 经济库检索物料落点（lib 根目——与 索引.md 同位）。
func econMaterialFile(dir string) string {
	return filepath.Join(dir, "检索物料.tsv")
}

// renderEconMaterial 经济库检索物料 TSV（列序同工程库 renderMaterial）。
func renderEconMaterial(entries []econMat) []byte {
	sort.Slice(entries, func(i, j int) bool { return entries[i].slug < entries[j].slug })
	var w strings.Builder
	w.WriteString(materialHeader)
	for _, m := range entries {
		w.WriteString(strings.Join([]string{
			"econ", sanitizeTSV(m.id), sanitizeTSV(m.kind), sanitizeTSV(m.status),
			sanitizeTSV(m.risk), sanitizeTSV(m.display), sanitizeTSV(m.text),
			joinItems(m.keywords), joinItems(m.scope), "",
		}, "\t"))
		w.WriteByte('\n')
	}
	return []byte(w.String())
}

// writeIfChanged 幂等写：内容同不动；写径=同目 temp+rename 原子换——
// 召回件自愈径（kb_recall.sh 缺料时 go run gen）与人工 gen 可并发，
// 裸 WriteFile 截断窗会让并发读者读到半截 TSV。
func writeIfChanged(path string, out []byte) error {
	if ex, err := os.ReadFile(path); err == nil && bytes.Equal(ex, out) {
		return nil
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kbgen-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}
