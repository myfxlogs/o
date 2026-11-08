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
	"guard_close_loop":       {"平仓循环自残", false, "守卫环熔断——同策略开仓已挂起（T-0423），解冻归裁定席判词/业主令"},
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
	// T-0441 补录：全谱告警 kind 归一（前漏 11 件——裸 kind 名曾作标题出组）。
	"bare_bag":              {"平仓残袋挂账", false, "平仓未回资产已挂账待处置"},
	"band_audit_drift":      {"收益档阈漂移", false, "族级阈两级档漂移——观测面待查"},
	"close_residual_perp":   {"合约残仓未平", false, "平仓未净敞口人工介入"},
	"dcd_auto_retire":       {"策略连败自动退役", false, "策略连败退役留痕——复盘归裁定席"},
	"notify_queue_usage":    {"通知队列水位告警", false, "PG 通知队列水位超阈——供方洪泛先兆"},
	"pool_tx_residue":       {"长事务钉住数据库", false, "会话事务超时钉死清理水位——先报不杀"},
	"projector_silent":      {"数据镜像断流", false, "SG 镜像写入停滞——投影臂待查"},
	"residual_clear_failed": {"杂币清仓失败", false, "白名单外持仓清仓败——升件不补单"},
	"residual_exposure":     {"杂币未授权持仓", false, "白名单外持仓清仓径处置中"},
	"scout_recommend":       {"新交易所注册建议", true, "注册属重大事项——业主裁决"},
	"volsell_auto_retire":   {"波动卖出策略退役", false, "策略连败退役留痕——复盘归裁定席"},
}

// HumanText 人话载荷装配（三段式——业主零黑话）。
func HumanText(kind, msg string) (text string, actable bool) {
	spec, ok := kindSpec[kind]
	if !ok {
		spec = struct {
			title    string
			actable  bool
			relation string
		}{"未定性系统事件", false, "未定性——需排查"}
	}
	act := "无需操作"
	if spec.actable {
		act = "请关注——可能需要条件配合（出资/凭据/权限）"
	}
	return fmt.Sprintf("%s\n发生：%s\n定性：%s\n要你做：%s（%s）",
		spec.title, msg, spec.relation, act, kind), spec.actable
}
