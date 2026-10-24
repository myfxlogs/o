// Humanize 告警载荷人话化（T-0081 钦定规格：[PHASE告警] 白话标题+发生/定性/
// 要你做 三段式；黑话 kind 名挪括注尾——业主只应条件请求，三域外降级 [经营事件]）。
package notify

import "fmt"

// kindSpec 处置性分档表（业主可处置域=出资/凭据/权限三件——域外降级事件级）。
var kindSpec = map[string]struct {
	title    string // 白话标题
	actable  bool   // 业主可处置（true=告警级，false=降级事件级）
	relation string // 定性注脚（已定性票号在案者注）
}{
	"ledger_diff_alert":   {"账实差超限", true, "未定性——需排查"},
	"rollback_failed":     {"平仓回滚败", false, "残留敞口人工升级"},
	"close_gave_up":       {"平仓放弃", false, "仓残人工处置"},
	"dd_hardline_tripped": {"净值硬线触停", true, "风险停手已自动执行"},
	"dd_close_exhausted":  {"触停平仓耗尽", false, "仓残人工处置"},
	"kill_switch":         {"急停开关", true, "系统停手状态变化"},
	"transfer_ambiguous":  {"划转歧态势", true, "钱动未确认——人工核实"},
	"net_budget_warn":     {"带宽预算预警", false, "观测面超阈"},
	"net_budget_crit":     {"带宽预算临界", false, "观测面超阈"},
	"bal_missing":         {"对账缺岗", false, "venue 读件缺席"},
}

// HumanText 人话载荷装配（三段式——业主零黑话）。
func HumanText(kind, msg string) (text string, actable bool) {
	spec, ok := kindSpec[kind]
	if !ok {
		spec = struct {
			title    string
			actable  bool
			relation string
		}{kind + " 事件", false, "未定性——需排查"}
	}
	act := "无需操作"
	if spec.actable {
		act = "请关注——可能需要条件配合（出资/凭据/权限）"
	}
	return fmt.Sprintf("%s\n发生：%s\n定性：%s\n要你做：%s（%s）",
		spec.title, msg, spec.relation, act, kind), spec.actable
}
