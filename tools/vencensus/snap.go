package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ---- 快照 ----

func writeSnapshot(path string, srcNote string, gaps map[string]map[string][]string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "<!-- vencensus 差集快照（工单#25843 venue 能力面普查基线——工具产，禁手编）\n     生成: %s  源: %s\n     对拍: skills/* canon + internal/cmd/tools 非 gen 码面调用点\n     重跑幂等——同文件 diff 出新增/消失项 -->\n\n", time.Now().UTC().Format("2006-01-02 15:04Z"), srcNote)
	venues := []string{"okx", "mt4", "mt5", "binance"}
	for _, v := range venues {
		g, ok := gaps[v]
		if !ok {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", v)
		fams := make([]string, 0, len(g))
		for f := range g {
			fams = append(fams, f)
		}
		sort.Strings(fams)
		for _, f := range fams {
			eps := g[f]
			sort.Strings(eps)
			fmt.Fprintf(&b, "### %s/%s (%d) —— %s\n\n", v, f, len(eps), use(v, f))
			for _, e := range eps {
				fmt.Fprintf(&b, "- %s\n", e)
			}
			b.WriteString("\n")
		}
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// pullOKXEndpoints 重拉官方页抽 /api/v5/* 字面量→tsv（源面可复拉钦定径）。
func pullOKXEndpoints(url, out string) (int, error) {
	resp, err := http.Get(url) //nolint:gosec // 官方只读页
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("docs 页 %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	re := regexp.MustCompile(`/api/v5/[a-zA-Z0-9/_-]*[a-zA-Z0-9]`)
	seen := map[string]bool{}
	var lines []string
	for _, m := range re.FindAll(b, -1) {
		p := strings.TrimSuffix(string(m), "/")
		if seen[p] {
			continue
		}
		seen[p] = true
		fam := strings.Split(strings.TrimPrefix(p, "/"), "/")[2]
		lines = append(lines, p+"\t"+fam)
	}
	sort.Strings(lines)
	if len(lines) < 100 {
		return 0, fmt.Errorf("抽得 %d 端点（<100 疑页体变——拒覆册）", len(lines))
	}
	return len(lines), os.WriteFile(out, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
