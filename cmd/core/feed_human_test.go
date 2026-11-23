package main

import (
	"strings"
	"testing"
)

// feed_human_test.go —— 人话翻译层桩测（工单O-32265）。
// 案件：spec 主类各一条产预期人话/未覆盖类回退原文/机器行仍在双行形。

func TestFeedHumanClasses(t *testing.T) {
	for _, c := range []struct {
		name string
		row  feedRow
		want string // 人话句必含子串
	}{
		{"落地-支名", feedRow{event: "落地", note: "落成 支=canon/psql-cvar-prod-evidence 尖=4c6f4a 尖 merge=05dbcf6"}, "「canon/psql-cvar-prod-evidence」已合并进主干"},
		{"落地-快径", feedRow{event: "落地", note: "落成快径 支=fix/esc-psql-cvar 尖=3bd527a merge=c29de91"}, "「fix/esc-psql-cvar」已合并进主干"},
		{"部署-谱系证空", feedRow{event: "部署", note: "o-core-clean:05dbcf6edac3 rc=0 base=09496c981cb1 谱系证空——diff全件在cmd/core依赖闭包外"}, "只有文档改动，无需重启，水位已记录"},
		{"部署-换版", feedRow{event: "部署", note: "deploy-oci 09496c981cb1 rc=0 arms=research"}, "产线已换上 09496c981"},
		{"心跳-证空跳", feedRow{event: "心跳", note: "心跳 证空跳部署 sha=05dbcf6edac3 base=09496c981cb1"}, "无代码改动，部署跳过，水位照记"},
		{"心跳-水位", feedRow{event: "心跳", note: "心跳 clean水位 sha=c29de91b4309 base=09496c981cb1"}, "部署水位正常（c29de91b4"},
		{"升件-器官报警", feedRow{event: "升件", escClass: "断流", escScope: "MT5 桥行情", note: "v=2611220613 面断流 MT5 桥行情：末鲜 11-20 05:35Z（陈 49h）——超 dead 阈"}, "「MT5 桥行情」报警："},
		{"升件-无scope退class", feedRow{event: "升件", escClass: "finding", note: "digest 发现 funding-carry 够线"}, "「finding」报警："},
		{"处置-警报销", feedRow{event: "处置", note: "升件#32247 销——canon支滞留因复审行词面闸不识"}, "警报 #32247 已处理"},
		{"处置-工单销", feedRow{event: "处置", note: "工单#31997 销——merged(37e698e1)外审#32055 值守执业主令"}, "工单 #31997 已结"},
		{"量表-自态账", feedRow{event: "量表", note: "v=09496c981cb1|0|11|live0/probe0/仓1 自态账: 部署=09496c981cb1(44m0s) 升件未决=0 工单=11(近限) 在场认领=0 NAV=10800"}, "机器自检：0 条报警待处理，11 张工单开着"},
		{"完工回执-支名", feedRow{event: "完工回执", note: "完工推支候审 fix/nav-residual-lens sha=1594ad6d643907089a1786e8fe29cc607cd4bf61——books_beat残差透镜"}, "「fix/nav-residual-lens」完工，等验收"},
		{"记事-完工回执", feedRow{event: "记事", seat: "施工席", note: "完工回执 O-27488 tradingBot普查研究件 digest 落账"}, "工单 O-27488 完工回执"},
		{"复审过-支名", feedRow{event: "复审过", seat: "唤知席", note: "复审过 sha=fd9406b38c5b2a4 canon/psql-cvar-prod-evidence——docs-only戳绿"}, "「canon/psql-cvar-prod-evidence」复审通过"},
		{"工单-新票", feedRow{event: "工单", seat: "外席", note: "工单/digest-finding-spawn: 研究digest发现词面→求知票自产闸"}, "新工单："},
		{"恢复-自愈", feedRow{event: "恢复", note: "升件#32247 复绿——支面三探撤编归日计数"}, "自愈："},
	} {
		got := feedHuman(c.row)
		if !strings.Contains(got, c.want) {
			t.Errorf("%s: feedHuman=%q 要含 %q", c.name, got, c.want)
		}
	}
}

func TestFeedHumanFallback(t *testing.T) {
	// 规则表全不中=feedHuman 空串（feedRender 层走通用兜底人话——O-35002）
	for _, ev := range []string{"等锁", "面清单新面", "参数调整", "监账", "叙事", "全新事件类xyz"} {
		r := feedRow{id: 1, seat: "跑席", event: ev, note: "k=v 机产行原文 venue读=1.0"}
		if got := feedHuman(r); got != "" {
			t.Errorf("未覆盖类 %s 应回退空串，得 %q", ev, got)
		}
	}
	// 处置非销账行（无 #N 销 形）也回退
	if got := feedHuman(feedRow{event: "处置", note: "升件#32247 关联工单#32055 在办"}); got != "" {
		t.Errorf("处置非销账应回退，得 %q", got)
	}
}

func TestFeedRenderDualLine(t *testing.T) {
	// 人话命中=双行：人话行在前+缩进机器行在下（id/原文不丢）
	r := feedRow{id: 32249, seat: "落地件", event: "落地",
		note: "落成 支=canon/psql-cvar-prod-evidence 尖=4c6f4a merge=05dbcf6"}
	out := feedRender(r)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("人话命中应双行，得 %d 行：%q", len(lines), out)
	}
	if !strings.Contains(lines[0], "已合并进主干") {
		t.Errorf("首行应人话句，得 %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "  ") || !strings.Contains(lines[1], "#32249") {
		t.Errorf("次行应缩进机器行含 id，得 %q", lines[1])
	}
}

// ── O-35002 业主面全谱人话化验收 ─────────────────────────────────

// 黑话词表——业主面首行（人话行）禁命中；机器审计行豁免（缩进行=机读层）。
var feedJargon = []string{
	"venue读", "账本净", "ops_event", "esc_scope", "k=v", "sha=",
	"merge=", "尖=", "rc=", "watermark", "fingerprint",
}

func feedHumanLine(t *testing.T, r feedRow) string {
	t.Helper()
	out := feedRender(r)
	return strings.SplitN(out, "\n", 2)[0]
}

func assertFeedNoJargon(t *testing.T, tag, line string) {
	t.Helper()
	for _, w := range feedJargon {
		if strings.Contains(line, w) {
			t.Fatalf("%s 黑话词 %q 命中人话行: %q", tag, w, line)
		}
	}
}

// spec 钦定常见事件类全覆盖——每类人话行非空且非原生 note 直出。
func TestFeedRenderCommonEvents(t *testing.T) {
	jargonNote := "venue读=9480.1008 账本净=9544.6921 k=v sha=abc123def merge=deadbeef"
	for _, ev := range []string{
		"唤起", "唤起果", "唤起败", "在办", "会话收口", "判词", "认领",
		"完工回执", "候审回执", "复审过", "升件", "处置", "工单", "部署",
		"落地", "记事", "量表", "profit_daily", "挂起", "败", "勘误",
		"恢复", "心跳", "审计", "决策", "备忘",
	} {
		r := feedRow{id: 99, seat: "唤知席", event: ev, note: jargonNote}
		line := feedHumanLine(t, r)
		if line == "" {
			t.Fatalf("%s 人话行为空", ev)
		}
		assertFeedNoJargon(t, ev, line)
		if strings.Contains(line, jargonNote) {
			t.Fatalf("%s 人话行漏原生 note: %q", ev, line)
		}
	}
}

// 未知事件类=显式兜底人话行+机器审计行双行——原生 note 不作首行。
func TestFeedRenderUnknownFallback(t *testing.T) {
	r := feedRow{id: 7, seat: "跑席", event: "全新事件类xyz",
		note: "venue读=1.0 账本净=2.0 k=v 原生机读行"}
	out := feedRender(r)
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("未知类应兜底人话+机器行双行，得 %d 行：%q", len(lines), out)
	}
	assertFeedNoJargon(t, "unknown", lines[0])
	if strings.Contains(lines[0], "原生机读行") {
		t.Fatalf("未知类人话行漏原生 note: %q", lines[0])
	}
	// 机器审计行=原文保留（id+note 不丢——append-only 对账面）
	if !strings.HasPrefix(lines[1], "  ") || !strings.Contains(lines[1], "#7") ||
		!strings.Contains(lines[1], "原生机读行") {
		t.Fatalf("机器审计行应保留原文，得 %q", lines[1])
	}
}

// 票题注入：工单#N/O-NNNN 引用→subj 解析非空则「题」随行；空=原样。
func TestFeedTicketSubjectEnrich(t *testing.T) {
	subj := func(id int64) string {
		if id == 35484 {
			return "出生闸三件套+credscan开集"
		}
		return ""
	}
	// 认领行：票号→票题
	out := feedRender(feedRow{id: 1, seat: "施工席", event: "认领",
		note: "认领 工单#35484 开工"}, subj)
	if !strings.Contains(out, "工单#35484「出生闸三件套+credscan开集」") {
		t.Fatalf("认领票题未注入: %q", out)
	}
	// O-NNNN 别名形（记事完工回执规则产「工单 O-N」）
	out2 := feedRender(feedRow{id: 2, seat: "施工席", event: "记事",
		note: "完工回执 O-35484 落成"}, subj)
	if !strings.Contains(out2, "O-35484「出生闸三件套+credscan开集」") {
		t.Fatalf("O- 票题未注入: %q", out2)
	}
	// 查无此票=原样不注（不硬编）
	out3 := feedRender(feedRow{id: 3, seat: "施工席", event: "认领",
		note: "认领 工单#99999 开工"}, subj)
	if strings.Contains(out3, "「") {
		t.Fatalf("无题票不应注入: %q", out3)
	}
	// subj=nil=纯渲染零注
	out4 := feedRender(feedRow{id: 4, seat: "施工席", event: "认领",
		note: "认领 工单#35484 开工"})
	if strings.Contains(out4, "「出生闸") {
		t.Fatalf("nil subj 不应注入: %q", out4)
	}
}
