// Package migrations —— 迁移文件序进嵌入（schema_migrations 基线制，架构约束 §1）。
// domain.tsv 域册随包嵌入（T-0168：projector 桥表集对册 fail-closed——消费方
// 与册面单源，禁各自硬编码漂移）。
package migrations

import (
	"bufio"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.sql domain.tsv
var FS embed.FS

// DomainTSV 域册原文（表名→md|fin|bridge——check_migrations 断言⑦ 同源）。
func DomainTSV() string {
	b, err := FS.ReadFile("domain.tsv")
	if err != nil {
		return "" // 嵌入面缺册=编译期已炸（embed 指令在），防御形
	}
	return string(b)
}

// BridgeTables 域册 bridge 行（八桥表——投影臂消费面唯一合法清单源）。
func BridgeTables() ([]string, error) {
	var out []string
	sc := bufio.NewScanner(strings.NewReader(DomainTSV()))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			return nil, fmt.Errorf("domain.tsv 坏行 %q（应 表名\\t域）", line)
		}
		if parts[1] == "bridge" {
			out = append(out, parts[0])
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}
