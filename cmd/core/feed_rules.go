package main

import "fmt"

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
	// ── O-35002 常见事件类补全（spec 钦定全谱）─────────────────────
	// 唤起三族：果=上岗成功；败=上岗失败；裸=唤起动作（剔集内不泄——
	// 规则仍录：他径/放行变体不漏译）。
	{"唤起果", "", "", func(r feedRow) string {
		return "值班机上岗干活"
	}},
	{"唤起败", "", "", func(r feedRow) string {
		return "值班机上岗失败：" + feedHumanSum(r.note)
	}},
	{"唤起", "", "", func(r feedRow) string {
		return "值班机被叫醒"
	}},
	// 认领：工单#N 或 O-NNNN 票号→票题白话（subj 注入 feedRender 层）。
	{"认领", "", "", func(r feedRow) string {
		if t := feedTicketTag(r.note); t != "" {
			return "认领了" + t + "，开工"
		}
		return "认领了一件工单开工"
	}},
	// 在办：租约续期/状态行——票号可解则带题。
	{"在办", "", "", func(r feedRow) string {
		if t := feedTicketTag(r.note); t != "" {
			return "在办：" + t
		}
		return "在办推进：" + feedHumanSum(r.note)
	}},
	// 会话收口=交班：谁交的什么班。
	{"会话收口", "", "", func(r feedRow) string {
		return feedSeatCNOr(r.seat) + "交班：" + feedHumanSum(r.note)
	}},
	// 判词：裁定席裁决——内容即 note 摘要。
	{"判词", "", "", func(r feedRow) string {
		return "裁决：" + feedHumanSum(r.note)
	}},
	// 候审回执：完工挂起等复审。
	{"候审回执", "", "", func(r feedRow) string {
		if b := feedHumanBranch(r.note); b != "" {
			return "「" + b + "」完工，挂起等复审"
		}
		return "一件完工，挂起等复审"
	}},
	// 挂起/败/勘误：状态类通用句式。
	{"挂起", "", "", func(r feedRow) string {
		return "挂起：" + feedHumanSum(r.note)
	}},
	{"败", "", "", func(r feedRow) string {
		return "失败：" + feedHumanSum(r.note)
	}},
	{"勘误", "", "", func(r feedRow) string {
		return "勘误：" + feedHumanSum(r.note)
	}},
	// profit_daily：盈亏日报——金额摘要人话化。
	{"profit_daily", "", "", func(r feedRow) string {
		return "今日盈亏日报：" + feedHumanSum(r.note)
	}},
	// 审计/决策/备忘/独白族：通用「谁记了什么」句式（黑话摘要层已过
	// feedGloss 术语表——原文不直出）。
	{"审计", "", "", func(r feedRow) string {
		return "审计：" + feedHumanSum(r.note)
	}},
	{"审计疑点", "", "", func(r feedRow) string {
		return "审计疑点：" + feedHumanSum(r.note)
	}},
	{"决策", "", "", func(r feedRow) string {
		return "决策：" + feedHumanSum(r.note)
	}},
	{"备忘", "", "", func(r feedRow) string {
		return "备注：" + feedHumanSum(r.note)
	}},
	{"独白", "", "", func(r feedRow) string {
		return "独白：" + feedHumanSum(r.note)
	}},
	{"独语", "", "", func(r feedRow) string {
		return "独白：" + feedHumanSum(r.note)
	}},
	{"自述", "", "", func(r feedRow) string {
		return "独白：" + feedHumanSum(r.note)
	}},
	{"自问", "", "", func(r feedRow) string {
		return "自问：" + feedHumanSum(r.note)
	}},
}
