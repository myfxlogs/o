package main

import (
	"fmt"
	"regexp"
	"strings"
)

// feed_gloss.go —— [O·动] 白话词典（工单级翻译层）:
// 席/事前缀译成「谁·干什么」，note 内机产缩写过术语表。
// 未录者原样透出（字典非白名单——宁露黑话不漏事）。

var feedSeatCN = map[string]string{
	"唤知席": "值班", "唤知机": "值班机",
	"施工席": "施工", "施工机": "施工机",
	"自我": "CEO", "自我机": "CEO机",
	"探测席": "探针", "审计席": "审计", "审计机": "审计机",
	"处置席": "处置", "外席": "外包",
	"决策席-DM写径": "参数决策", "DEPLOY": "部署",
	"独白席": "独白", "跑席": "跑批", "接市官": "接市",
	"研究席": "研究", "研究员": "研究",
}

var feedEventCN = map[string]string{
	"会话收口": "交班", "处置": "处置", "工单": "新工单",
	"认领": "认领", "升件": "警报", "恢复": "自愈",
	"备忘": "备注", "判词": "判词", "审计": "审计",
	"审计疑点": "审计疑点", "部署": "部署", "决策": "决策",
	"在办": "在办", "独白": "独白", "独语": "独白",
	"自述": "独白", "自问": "自问", "条件请求": "求物",
	"求助值守": "求救", "量表": "报表",
}

// feedGloss note 内术语表——机产缩写→人话（长词先排防短词截断误伤）。
var feedGloss = strings.NewReplacer(
	"矿双", "僵尸矿",
	"升件", "警报",
	"面净", "干净",
	"收口", "交班",
	"唤知", "值班",
	"施工席", "施工",
	"自我席", "CEO",
	"探测席", "探针",
	"审计席", "审计",
	"量表", "报表",
	"恢复", "自愈",
	"唤起", "上岗",
)

// feedHead 席/事前缀白话——「值班/交班 #19212」。
func feedHead(seat, event string) string {
	s := feedSeatCN[seat]
	if s == "" {
		s = seat
	}
	e := feedEventCN[event]
	if e == "" {
		e = event
	}
	return s + "/" + e
}

// ── 人话翻译层（工单O-32265）────────────────────────────────────
// 业主痛点：「落成快径/谱系证空/同根簇折叠」不解释即天书——存储层
// （判据机语，append-only 正确不动）与呈现层（业主账页）未分离。
// 本层=呈现层纯翻译：不改 ops_event 写径/存径，模板表机械映射禁
// LLM（判据不外包）。未中类回退原文不丢信息。
// feedHumanRules 模板表——event/seat/word 三锚先中先得（""=不锚），
// render 返 "" 视为未中续试下行。加新类=加一行：词面锚宁专勿宽
// （宽锚误中=人话说错比黑话更糟）。

type feedHumanRule struct {
	event, seat, word string
	render            func(feedRow) string
}

var (
	feedHumanBranchRe  = regexp.MustCompile(`支=(\S+)`)
	feedHumanBranchRe2 = regexp.MustCompile(`\b(?:fix|feat|canon|audit|t|revfix|hotfix)/[\w.-]+`)
	feedHumanSHARe     = regexp.MustCompile(`sha=([0-9a-f]{9,40})`)
	feedHumanHexRe     = regexp.MustCompile(`[0-9a-f]{9,40}`)
	feedHumanEscRe     = regexp.MustCompile(`升件#([0-9]+)\s*销`)
	feedHumanTicketRe  = regexp.MustCompile(`工单#([0-9]+)\s*销`)
	feedHumanOTkRe     = regexp.MustCompile(`O-[0-9]+`)
	feedHumanEscNRe    = regexp.MustCompile(`升件未决=([0-9]+)`)
	feedHumanTickNRe   = regexp.MustCompile(`工单=([0-9]+)`)
)

// feedHumanBranch 支名提取——「支=<name>」优先，次取 fix|feat|canon|…/ token。
func feedHumanBranch(note string) string {
	if m := feedHumanBranchRe.FindStringSubmatch(note); m != nil {
		return strings.TrimRight(m[1], "，。；、)")
	}
	if m := feedHumanBranchRe2.FindStringSubmatch(note); m != nil {
		return m[0]
	}
	return ""
}

// feedHumanSHA 版本戳提取——sha= 键优先，次取首枚 ≥9 位 hex 串截 9 位。
func feedHumanSHA(note string) string {
	if m := feedHumanSHARe.FindStringSubmatch(note); m != nil {
		return m[1][:9]
	}
	if m := feedHumanHexRe.FindStringSubmatch(note); m != nil {
		return m[0][:9]
	}
	return ""
}

// feedHumanSum 摘要截 60 字（rune 安全）——报警/自愈类人话事由共用。
func feedHumanSum(note string) string {
	sum := feedTidy(note)
	if rs := []rune(sum); len(rs) > 60 {
		sum = string(rs[:60]) + "…"
	}
	return sum
}

var feedHumanRules = []feedHumanRule{
	// 落地：支合主干（落成/落成快径两形同译——快径=免复审径非另一类）
	{"落地", "", "", func(r feedRow) string {
		if b := feedHumanBranch(r.note); b != "" {
			return "「" + b + "」已合并进主干"
		}
		return "一支已合并进主干"
	}},
	// 部署：谱系证空=diff 全在依赖闭包外 docs-only，无需重启
	{"部署", "", "谱系证空", func(feedRow) string {
		return "只有文档改动，无需重启，水位已记录"
	}},
	{"部署", "", "", func(r feedRow) string {
		if sha := feedHumanSHA(r.note); sha != "" {
			return "产线已换上 " + sha
		}
		return "产线部署一轮（明细见机器行）"
	}},
	// 心跳：证空跳部署变体先中（「心跳 证空跳部署」形）
	{"心跳", "", "证空", func(feedRow) string {
		return "无代码改动，部署跳过，水位照记"
	}},
	{"心跳", "", "", func(r feedRow) string {
		if sha := feedHumanSHA(r.note); sha != "" {
			return "部署水位正常（" + sha + "）"
		}
		return "部署水位正常"
	}},
	// 升件：器官=esc_scope（空退 esc_class 再退「系统」），事由=人话摘要
	{"升件", "", "", func(r feedRow) string {
		organ := r.escScope
		if organ == "" {
			organ = r.escClass
		}
		if organ == "" {
			organ = "系统"
		}
		return fmt.Sprintf("「%s」报警：%s", organ, feedHumanSum(r.note))
	}},
	// 处置销账：升件#N 销=警报处理；工单#N 销=工单结——「#N 销」紧邻形
	// 双锚取先者（主语在前）；销账词面外处置行（无#N销形）回退原文。
	{"处置", "", "销", func(r feedRow) string {
		ei := feedHumanEscRe.FindStringSubmatchIndex(r.note)
		ti := feedHumanTicketRe.FindStringSubmatchIndex(r.note)
		switch {
		case ei != nil && (ti == nil || ei[0] < ti[0]):
			return "警报 #" + r.note[ei[2]:ei[3]] + " 已处理"
		case ti != nil:
			return "工单 #" + r.note[ti[2]:ti[3]] + " 已结"
		}
		return ""
	}},
	// 量表·自态账：升件未决=/工单= 计数解析（feedExclude 剔集内——
	// 本规则服务将来若放行/他径复用，模板不空挂）
	{"量表", "", "升件未决=", func(r feedRow) string {
		n := feedHumanEscNRe.FindStringSubmatch(r.note)
		if n == nil {
			return "机器自检快报"
		}
		if t := feedHumanTickNRe.FindStringSubmatch(r.note); t != nil {
			return "机器自检：" + n[1] + " 条报警待处理，" + t[1] + " 张工单开着"
		}
		return "机器自检：" + n[1] + " 条报警待处理"
	}},
	// 完工/复审/新票
	{"完工回执", "", "", func(r feedRow) string {
		if b := feedHumanBranch(r.note); b != "" {
			return "「" + b + "」完工，等验收"
		}
		return "一件完工，等验收"
	}},
	{"工单完工", "", "", func(r feedRow) string {
		if b := feedHumanBranch(r.note); b != "" {
			return "「" + b + "」完工"
		}
		return "工单完工"
	}},
	{"复审过", "", "", func(r feedRow) string {
		if b := feedHumanBranch(r.note); b != "" {
			return "「" + b + "」复审通过"
		}
		return "一支复审通过"
	}},
	{"记事", "", "完工回执", func(r feedRow) string {
		if m := feedHumanOTkRe.FindStringSubmatch(r.note); m != nil {
			return "工单 " + m[0] + " 完工回执"
		}
		return "完工回执落账"
	}},
	{"工单", "", "", func(r feedRow) string {
		return "新工单：" + feedHumanSum(r.note)
	}},
	// 自愈
	{"恢复", "", "", func(r feedRow) string {
		return "自愈：" + feedHumanSum(r.note)
	}},
}

// feedHuman 人话句——规则表先中先得；全表不中=""（调用侧回退机器行原文）。
func feedHuman(r feedRow) string {
	for _, rule := range feedHumanRules {
		if rule.event != "" && rule.event != r.event {
			continue
		}
		if rule.seat != "" && rule.seat != r.seat {
			continue
		}
		if rule.word != "" && !strings.Contains(r.note, rule.word) {
			continue
		}
		if s := rule.render(r); s != "" {
			return s
		}
	}
	return ""
}

// feedRender 单事件行渲染——人话命中=人话行+缩进机器行双行；未中=机器行
// 原文单行。行形取舍：人话行在前=业主主读面一眼懂（「业主看账页」律落地），
// 机器行缩进在下=判据/id 对账不丢；未覆盖类单行原文零变化。
func feedRender(r feedRow) string {
	head := fmt.Sprintf("%s #%d %s", feedHead(r.seat, r.event), r.id, feedTidy(r.note))
	if h := feedHuman(r); h != "" {
		return h + "\n  " + head
	}
	return head
}
