// Humanize 告警载荷人话化（T-0081 钦定规格：[O告警] 白话标题+发生/定性/
// 要你做 三段式；黑话 kind 名挪括注尾——业主只应条件请求，三域外降级 [经营事件]）。
package notify

import "fmt"

// kindSpec 处置性分档表（业主可处置域=出资/凭据/权限三件——域外降级事件级）。
var kindSpec = map[string]struct {
	title    string // 白话标题
	actable  bool   // 业主可处置（true=告警级，false=降级事件级）
	relation string // 定性注脚（已定性票号在案者注）
}{
	"ledger_diff_alert":      {"账实差超限", true, "未定性——需排查"},
	"rollback_failed":        {"平仓回滚败", false, "残留敞口人工升级"},
	"close_gave_up":          {"平仓放弃", false, "仓残人工处置"},
	"dd_hardline_tripped":    {"净值硬线触停", true, "风险停手已自动执行"},
	"dd_close_exhausted":     {"触停平仓耗尽", false, "仓残人工处置"},
	"kill_switch":            {"急停开关", true, "系统停手状态变化"},
	"transfer_ambiguous":     {"划转歧态势", true, "钱动未确认——人工核实"},
	"net_budget_warn":        {"带宽预算预警", false, "观测面超阈"},
	"net_budget_crit":        {"带宽预算临界", false, "观测面超阈"},
	"bal_missing":            {"对账缺岗", false, "venue 读件缺席"},
	"research_arm_silent":    {"研究臂静默", true, "在编臂零产出——机会面停摆（T-0117）"},
	"retention_volume_alert": {"体积哨兵告警", false, "库/单表体积超阈——新无底洞查证（T-0153）"},
	"okxws_face_dead":        {"行情面断流", false, "订阅标的行情全冻——系统面故障（T-0129）"},
	"bridge_stale":           {"采集桥断链", false, "dev 蒸馏写径超时——SG REST 兜底自动接管（T-0137）"},
	"trxvote_claim":          {"TRX 投票领奖到账", false, "投票引擎自动执行——收入事件（T-0220）"},
	"trxvote_rebalance":      {"TRX 投票改投执行", false, "投票引擎自动执行——年化优化动作（T-0220）"},
	"trxvote_engine":         {"TRX 投票引擎异常", false, "自营盘自动面故障——需排查（T-0220）"},
	"trx_rent":               {"TRX 能量租金入账", false, "租金收款自动面——收入事件（T-0220）"},
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
