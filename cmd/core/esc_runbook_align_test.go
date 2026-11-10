// esc_runbook_align_test.go —— 探针 scope 面 vs 唤知 runbook 判词档对齐
// 机检：escEdge/escEdgeU 升件的每个 scope 字面量（或前缀拼接的钦定前缀）
// 必须在 runbook 文本中有档——探针升件而 runbook 无判词=升件进认知环
// 无人会裁（「系统看得见但脑不会判」对齐断档）。
package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// escScopeRe 抓 escEdge/escEdgeU 第四参 scope 字面量；
// escPrefixRe 抓 "xxx_"+ 前缀拼接形（动态尾 scope 的钦定前缀）。
var (
	escScopeRe  = regexp.MustCompile(`escEdgeU?\(ctx,\s*deadSince,\s*"[^"]+",\s*"([^"]+)"`)
	escPrefixRe = regexp.MustCompile(`"([a-z_]+_)"\s*\+`)
)

func TestEscScopesCoveredByRunbook(t *testing.T) {
	srcs, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	scopes := map[string]bool{}
	for _, f := range srcs {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		for _, m := range escScopeRe.FindAllSubmatch(b, -1) {
			scopes[string(m[1])] = true
		}
		for _, m := range escPrefixRe.FindAllSubmatch(b, -1) {
			scopes[strings.TrimSuffix(string(m[1]), "_")] = true
		}
	}
	rb, err := os.ReadFile(filepath.Join("..", "..", "scripts", "cognition", "runbook.md"))
	if err != nil {
		t.Fatalf("runbook: %v", err)
	}
	text := string(rb)
	var missing []string
	for s := range scopes {
		if !strings.Contains(text, s) {
			missing = append(missing, s)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Fatalf("探针 scope 缺 runbook 判词档: %v", missing)
	}
}
