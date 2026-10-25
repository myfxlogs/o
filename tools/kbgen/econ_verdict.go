// econ_verdict.go —— noxkb econ-verdict 分诊子命令（消化链建制票②）：
// 待归纳行裁定三分出路落库 + 行尾打 ✅ 消解标（机械与文件双轨同一裁定）。
//
//	noxkb econ-verdict -key <event_key> -verdict canon|dismiss|merge \
//	  -reason <理由必填> [-entry <条目id>] [-fixpoint <kind:target>] [-by <席位>]
//
// 语义：canon=转正条目（-entry 必填且条目文件须存在、含 **回改点**——无回改点
// 不入库）；merge=并入既有条目（-entry=目标 id）；dismiss=驳回（-reason 必填
// 防垃圾桶——同因重报由负结算接线转成 warn 病灶信号）。canon 且 fixpoint
// 前缀=semantic: → 自动立修订票（告警带 [角色:决策] 派发行——语义类教训
// 无机械闸可译，裁定落库同时把修订票推到决策席队列）。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func econVerdictCmd() error {
	fs := flag.NewFlagSet("econ-verdict", flag.ExitOnError)
	key := fs.String("key", "", "裁定键（机械行=settlement_event_key；[LLM草稿] 行=llm:<slug>）")
	verdict := fs.String("verdict", "", "canon|dismiss|merge")
	reason := fs.String("reason", "", "理由（必填——dismiss 垃圾桶防线）")
	entry := fs.String("entry", "", "canon=新条目 id / merge=并入目标 id")
	fixpoint := fs.String("fixpoint", "", "回改点 kind:target（canon 必填——eval|kill_trigger|combo_gate|sizing|semantic:<target>）")
	by := fs.String("by", "", "裁定席位署名")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}
	if *key == "" || *verdict == "" {
		return fmt.Errorf("usage: noxkb econ-verdict -key <k> -verdict canon|dismiss|merge -reason <r> [-entry <id>] [-fixpoint <k:t>]")
	}

	// T-0003 移植裁定径：nox 库表缺席——裁定账落 docs/经济经验库/裁定账.md
	// 行追加（台账件非库表；库表域另票——O 无 verdict 表）。
	dir := econDirPath()
	if *verdict == "canon" || *verdict == "merge" {
		if err := econEntryGate(dir, *entry); err != nil {
			return err
		}
	}
	ledger := filepath.Join(dir, "裁定账.md")
	lf, err := os.OpenFile(ledger, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("裁定账落位败: %w", err)
	}
	if _, err := fmt.Fprintf(lf, "%s|%s|%s|%s|%s|%s|%s\n",
		*key, *verdict,
		*reason, *entry, *fixpoint, *by, time.Now().UTC().Format("2006-01-02T15:04Z")); err != nil {
		lf.Close()
		return err
	}
	lf.Close()
	if err := econMarkPendingLine(filepath.Join(dir, "待归纳.md"), *key, *verdict, *entry); err != nil {
		return fmt.Errorf("裁定已落账但行标失败: %w", err)
	}
	fmt.Printf("econ-verdict: %s → %s (entry=%s fixpoint=%s)\n", *key, *verdict, *entry, *fixpoint)
	return nil
}

// econEntryGate canon/merge 条目闸：文件须存在（条目/ 优先、根目回落）且含
// **回改点** 字段（无回改点不入库——与 gen-econ 门禁同判据防绕）。
func econEntryGate(dir, id string) error {
	if id == "" {
		return fmt.Errorf("canon/merge 须 -entry <条目id>")
	}
	for _, p := range []string{
		filepath.Join(dir, "条目", id+".md"),
		filepath.Join(dir, id+".md"),
	} {
		if b, err := os.ReadFile(p); err == nil {
			if !strings.Contains(string(b), "**回改点**") {
				return fmt.Errorf("econ-verdict: %s 缺 **回改点** 字段——无回改点不入库", p)
			}
			return nil
		}
	}
	return fmt.Errorf("econ-verdict: 条目 %s 不存在（条目/ 与根目均无）", id)
}

// econMarkPendingLine 待归纳行尾打消解标：` ✅ <verdict>[:<entry>]`。
// 键未命中→错（裁定落库已发生，行标必须显式成败防半消）。
func econMarkPendingLine(path, key, verdict, entry string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	mark := " ✅ " + verdict
	if entry != "" {
		mark += ":" + entry
	}
	hit := false
	lines := strings.Split(string(b), "\n")
	for i, ln := range lines {
		if strings.Contains(ln, "key="+key) || strings.Contains(ln, "key="+key+" ") {
			if strings.Contains(ln, "✅") {
				return fmt.Errorf("行已消解: %s", strings.TrimSpace(ln))
			}
			lines[i] = strings.TrimRight(ln, " \t") + mark
			hit = true
		}
	}
	if !hit {
		return fmt.Errorf("待归纳行未命中 key=%s", key)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
