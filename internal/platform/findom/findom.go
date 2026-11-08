// Package findom —— 仓账写域钉（T-0489 径A：simvenue 仓写径分域）。
//
// 域闭集二值：Live=真户正本（fin_*），Sim=仿真仓域（sim_*）。仓生命周期
// 五表全簇随域同迁（execution/position/order_leg/cashflow/guard_close——
// 引用闭合同域内；卫星表分域则 execution_id/position_ref 撞号回染真户行）。
// 表名映射只出白名五键——禁自由拼接（写面收窄成显式域声明，漏接=编译错
// 非静默写错表）。
package findom

// Domain 仓账写域（live|sim——域闭集外值 fail-closed，见 Must）。
type Domain string

const (
	Live Domain = "live" // 真户正本——fin_* 表
	Sim  Domain = "sim"  // 仿真仓域——sim_* 表
)

// Must 域合法钉（构造面调用——闭集外值=编程错当场炸，禁静默归一真户域
// 复现「sim 写进 fin」病灶本型）。
func (d Domain) Must() Domain {
	if d != Live && d != Sim {
		panic("findom: 非法写域 " + string(d))
	}
	return d
}

func (d Domain) t(base string) string {
	if d == Sim {
		return "sim_" + base
	}
	return "fin_" + base
}

// 五表域名（白名键——本仓仓生命周期簇全件）：
func (d Domain) Execution() string  { return d.t("execution") }
func (d Domain) Position() string   { return d.t("position") }
func (d Domain) OrderLeg() string   { return d.t("order_leg") }
func (d Domain) Cashflow() string   { return d.t("cashflow") }
func (d Domain) GuardClose() string { return d.t("guard_close") }
