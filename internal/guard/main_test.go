package guard

import (
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestMain 包级共享 schema（工单#26108 per-test setup 税根治）——建面
// 一次+每测 Pool 定向清表代替逐测随机 schema+fin 全链迁移重施。
// SeedRestore 全量列（子→父序，清表两相位内部处理）：凡迁移有种子
// 或疑有种子/被写面/被 ops_event 触发器波及之表走快照还原=恒复位
// 迁移后基线（旧径每测新 schema 同义）；app_config 键册由共享径
// 自动快照复位，不入列。NoTrigger=ops_event——sem_gate 拒写/act
// pkey/chain 改写等触发器群，种子还原 INSERT 须旁路（测试期写径
// 触发器照常全活）。
// 序依据=fin_* 表间实测 FK 边（order_leg/position→execution、
// strategy_transition/paper_fill→strategy、risk_token→risk_decision、
// precedent→ops_event）。
func TestMain(m *testing.M) {
	testutil.ShareMain(m, testutil.ShareSpec{
		Domains: []string{"fin"},
		Migrate: pgstore.Migrate,
		SeedRestore: []string{
			"fin_guard_close",
			"fin_order_leg",
			"fin_position",
			"fin_execution",
			"fin_risk_token",
			"fin_risk_decision",
			"fin_strategy_transition",
			"fin_paper_fill",
			"fin_cashflow",
			"fin_guard_suspend",
			"fin_holding",
			"fin_opp_template",
			"ops_decision",
			"ops_headless",
			"ops_event_act",
			"ops_precedent",
			"ops_event",
			"md_ticks",
			"md_equity_curve",
			"md_funding",
			"fin_strategy",
		},
		NoTrigger: []string{"ops_event"},
	})
}
