package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ── 人话翻译层（工单O-32265）────────────────────────────────────
// 业主痛点：「落成快径/谱系证空/同根簇折叠」不解释即天书——存储层
// （判据机语，append-only 正确不动）与呈现层（业主账页）未分离。
// 本层=呈现层纯翻译：不改 ops_event 写径/存径，模板表机械映射禁
// LLM（判据不外包）。未中类回退原文不丢信息。
// feedHumanRules 模板表——event/seat/word 三锚先中先得（""=不锚），
// render 返 "" 视为未中续试下行。加新类=加一行：词面锚宁专勿宽
// （宽锚误中=人话说错比黑话更糟）。

var (
	feedHumanBranchRe  = regexp.MustCompile(`支=([A-Za-z0-9_./-]+)`)
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

// feedSeatCNOr 席名白话——未录原样透出（字典非白名单律同 feedHead）。
func feedSeatCNOr(seat string) string {
	if s := feedSeatCN[seat]; s != "" {
		return s
	}
	return seat
}

// feedTicketRefRe 票号引用两形：工单#N（ops_event id 域）与 O-NNNN
// （同域别名——frameSeats/票桌同约定）。升件#N=esc id 异域不入本表。
var feedTicketRefRe = regexp.MustCompile(`工单\s*#\s*([0-9]+)|工单\s+O-([0-9]+)|\bO-([0-9]+)\b`)

// feedTicketTag 首枚票号引用人话形（无票号=""）——「工单#123」/
// 「工单 O-27488」，供句式嵌入；票题注入在 feedRender 层（需池）。
func feedTicketTag(note string) string {
	m := feedTicketRefRe.FindStringSubmatch(note)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return "工单#" + m[1]
	}
	if m[2] != "" {
		return "工单 O-" + m[2]
	}
	return "O-" + m[3]
}

// feedTicketEnrichRe 人话行内票号引用（与 feedTicketTag 产出同形+处置/
// 记事规则直写形「工单 #N」「工单 O-N」全覆盖）。
var feedTicketEnrichRe = regexp.MustCompile(`工单\s*#\s*([0-9]+)|工单\s+O-([0-9]+)|\bO-([0-9]+)\b`)

// feedTicketEnrich 人话行票号注题——subj(id)=票题白话（空=不注，
// 原样透出不硬编）。subj=nil=纯渲染无注（桩测/他径）。
func feedTicketEnrich(h string, subj func(int64) string) string {
	if subj == nil {
		return h
	}
	return feedTicketEnrichRe.ReplaceAllStringFunc(h, func(m string) string {
		i := strings.IndexAny(m, "0123456789")
		if i < 0 {
			return m
		}
		id, err := strconv.ParseInt(m[i:], 10, 64)
		if err != nil {
			return m
		}
		s := subj(id)
		if s == "" {
			return m
		}
		return m + "「" + s + "」"
	})
}

// feedHuman 人话句——规则表先中先得；全表不中=""（调用侧走通用兜底）。
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

// feedRender 单事件行渲染——O-35002 全谱人话化：人话命中=人话行
// （票号注题）+缩进机器行双行；未中类=显式通用兜底人话行+机器行——
// 原生 note 永不作业主面首行（黑话不静默漏），机读审计行照旧缩进在下
// （id/原文不丢）。subj=票题解析器（nil=不注题）。
func feedRender(r feedRow, subj ...func(int64) string) string {
	head := fmt.Sprintf("%s #%d %s", feedHead(r.seat, r.event), r.id, feedTidy(r.note))
	var sf func(int64) string
	if len(subj) > 0 {
		sf = subj[0]
	}
	h := feedHuman(r)
	if h == "" {
		e := feedEventCN[r.event]
		if e == "" {
			e = "动态"
		}
		h = feedSeatCNOr(r.seat) + "记了一笔「" + e + "」"
	}
	return feedTicketEnrich(h, sf) + "\n  " + head
}
