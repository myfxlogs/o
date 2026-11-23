// Humanize 告警载荷人话化（T-0081 钦定规格：[O告警] 白话标题+发生/定性/
// 要你做 三段式；黑话 kind 名挪括注尾——业主只应条件请求，三域外降级 [经营事件]）。
package notify

import (
	"fmt"
	"strings"
)

// kindSpec 处置性分档表（业主可处置域=出资/凭据/权限三件——域外降级事件级）。
// plain=「发生」段人话摘要（工单O-34997：payload=机读诊断原文只喂唤知
// 分诊径，业主面禁黑话——plain 产白话三件事：怎么了/要紧程度/票在管否；
// nil=走 genericPlain 通用白话兜底，actable=true 者禁 nil=桩测红）。
var kindSpec = map[string]struct {
	title    string // 白话标题
	actable  bool   // 业主可处置（true=告警级，false=降级事件级）
	relation string // 定性注脚（已定性票号在案者注）
	plain    func(payload string) string
}{
	"ledger_diff_alert": {"账实差超限", true, "未定性——需排查", plainLedgerDiff},
	"rollback_failed":   {"平仓回滚败", false, "残留敞口人工升级", nil},
	"close_gave_up":     {"平仓放弃", false, "仓残人工处置", nil},
	"guard_close_loop":  {"平仓循环自残", false, "守卫环熔断——同策略开仓已挂起（T-0423），解冻归裁定席判词/业主令", nil},
	"dd_hardline_tripped": {"净值硬线触停", true, "风险停手已自动执行", func(string) string {
		return "账户净值跌破安全硬线，系统已自动停手减仓——风控闸在正常工作，钱在保，不用动。"
	}},
	"dd_close_exhausted": {"触停平仓耗尽", false, "仓残人工处置", nil},
	"dd_line_blind": {"净值硬线失明", true, "硬线未在判撤——判据/急停开关/权益读不出，需人工排查", func(string) string {
		return "净值安全线暂时判断不了（数据读不出来）——硬线保护可能失效，机器在排查，可能需要你看一眼。"
	}},
	"kill_switch": {"急停开关", true, "系统停手状态变化", plainKillSwitch},
	"transfer_ambiguous": {"划转歧态势", true, "钱动未确认——人工核实", func(string) string {
		return "一笔资金划转结果不确定——钱可能已经动了也可能没动，需要人工核实，先别重复操作。"
	}},
	"net_budget_warn": {"带宽预算预警", false, "观测面超阈", nil},
	"net_budget_crit": {"带宽预算临界", false, "观测面超阈", nil},
	"bal_missing":     {"对账缺岗", false, "venue 读件缺席", nil},
	"research_arm_silent": {"研究臂静默", true, "在编臂零产出——机会面停摆（T-0117）", func(string) string {
		return "负责找机会的研究模块一段时间没有产出——赚钱机会面可能停摆，机器在查。"
	}},
	"retention_volume_alert": {"体积哨兵告警", false, "库/单表体积超阈——新无底洞查证（T-0153）", nil},
	"okxws_face_dead":        {"行情面断流", false, "订阅标的行情全冻——系统面故障（T-0129）", nil},
	"bridge_stale":           {"采集桥断链", false, "dev 蒸馏写径超时——SG REST 兜底自动接管（T-0137）", nil},
	// T-0441 补录：全谱告警 kind 归一（前漏 11 件——裸 kind 名曾作标题出组）。
	"bare_bag":              {"平仓残袋挂账", false, "平仓未回资产已挂账待处置", nil},
	"band_audit_drift":      {"收益档阈漂移", false, "族级阈两级档漂移——观测面待查", nil},
	"close_residual_perp":   {"合约残仓未平", false, "平仓未净敞口人工介入", nil},
	"dcd_auto_retire":       {"策略连败自动退役", false, "策略连败退役留痕——复盘归裁定席", nil},
	"notify_queue_usage":    {"通知队列水位告警", false, "PG 通知队列水位超阈——供方洪泛先兆", nil},
	"pool_tx_residue":       {"长事务钉住数据库", false, "会话事务超时钉死清理水位——先报不杀", nil},
	"projector_silent":      {"数据镜像断流", false, "SG 镜像写入停滞——投影臂待查", nil},
	"residual_clear_failed": {"杂币清仓失败", false, "白名单外持仓清仓败——升件不补单", nil},
	"residual_exposure":     {"杂币未授权持仓", false, "白名单外持仓清仓径处置中", nil},
	"scout_recommend": {"新交易所注册建议", true, "注册属重大事项——业主裁决", func(string) string {
		return "发现一个新的交易所值得注册——注册是大事（要钱要凭据），需要你拍板。"
	}},
	"volsell_auto_retire": {"波动卖出策略退役", false, "策略连败退役留痕——复盘归裁定席", nil},
}

// plainLedgerDiff 账实差「发生」段人话（工单O-34997 示范口径——payload
// 机读诊断不解析回读：方向/金额由生产侧 AlertPlain 显式载荷直给，本
// 函数只兜底未带显式载荷的调用形）。
func plainLedgerDiff(payload string) string {
	return "账上记的钱和交易所实际显示对不上——金额与初步定性机器在查，初步看不像钱丢了，不用动。"
}

// plainKillSwitch 急停开关方向敏感人话（→false=解除通报；→true=停手告警；
// 方向词面缺席=通用白话——payload 机读形不直出业主面）。
func plainKillSwitch(payload string) string {
	switch {
	case strings.Contains(payload, "→false"):
		return "系统刚从停手恢复放行——之前的停手已解除，不用动。"
	case strings.Contains(payload, "→true"):
		return "系统进入停手状态——新开仓已拦、平仓放行，钱在保；需要查明原因并复位。"
	default:
		return "系统停手开关状态有变化——机器在确认影响面。"
	}
}

// genericPlain 未知/未配 kind 的「发生」段兜底人话（工单O-34997——禁裸
// 机读文出业主面：payload 一律不进本函数返回值）。
func genericPlain(title string) string {
	return "系统发现一处需要关注的情况（" + title + "）——机器诊断中，明细见工单记录。"
}

// HumanText 人话载荷装配（三段式——业主零黑话）。显式 plain=生产侧
// 双载荷人话摘要（非空优先于 kindSpec.plain——机读 payload 只喂分诊径
// 不出业主面）。
func HumanTextPlain(kind, msg, plain string) (text string, actable bool) {
	spec, ok := kindSpec[kind]
	if !ok {
		spec = struct {
			title    string
			actable  bool
			relation string
			plain    func(payload string) string
		}{"未定性系统事件", false, "未定性——需排查", nil}
	}
	occurred := plain
	if occurred == "" {
		if spec.plain != nil {
			occurred = spec.plain(msg)
		}
	}
	if occurred == "" {
		occurred = genericPlain(spec.title)
	}
	// 方向敏感定性：kill_switch 同 kind 双向（cmd/core/notify.go 发射
	// msg=「kill_switch <prev>→<new>」）——解除=复位落点是通报非告警,
	// 勿用「请关注」报已解事件（2026-11-21 实证：复位通知被读成待办急件）。
	if kind == "kill_switch" {
		switch {
		case strings.Contains(msg, "→false"):
			spec.title, spec.actable = "急停解除", false
			spec.relation = "停手已解除——条件配合完成或人工翻闸复位"
		case strings.Contains(msg, "→true"):
			spec.title = "系统停手"
			spec.relation = "急停触发——新开仓已拦平仓放行；查明因+复位后翻闸"
		}
	}
	act := "无需操作"
	if spec.actable {
		act = "请关注——可能需要条件配合（出资/凭据/权限）"
	}
	return fmt.Sprintf("%s\n发生：%s\n定性：%s\n要你做：%s（%s）",
		spec.title, occurred, spec.relation, act, kind), spec.actable
}

// HumanText 人话载荷装配（三段式——业主零黑话）。
func HumanText(kind, msg string) (text string, actable bool) {
	return HumanTextPlain(kind, msg, "")
}
