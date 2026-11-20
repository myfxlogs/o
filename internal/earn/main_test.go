package earn

import (
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestMain 包级共享 schema（工单#26108 per-test setup 税根治）——建面
// 一次+每测 Pool 定向清表代替逐测随机 schema+fin/md 全链迁移重施
// （序同 setupW：md 先 fin 后）。
// SeedRestore 全量列（子→父序，清表两相位内部处理）：写面并集=
// 包直写表+fin 族 FK 邻表+ops_event 触发器波及表。app_config 键册
// 由共享径自动快照复位，不入列。NoTrigger=ops_event——sem_gate
// 拒写/act pkey/chain 改写等触发器群，种子还原 INSERT 须旁路。
// 序依据=fin_* 表间实测 FK 边（order_leg/position→execution、
// risk_token→risk_decision、transition/paper_fill→strategy、
// precedent→ops_event）。
func TestMain(m *testing.M) {
	testutil.ShareMain(m, testutil.ShareSpec{
		Domains: []string{"md", "fin"},
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
			"fin_known_residual",
			"fin_residual_custody",
			"fin_cashflow",
			"fin_guard_suspend",
			"fin_holding",
			"fin_strategy",
			"ops_decision",
			"ops_headless",
			"ops_event_act",
			"ops_precedent",
			"ops_event",
			"ops_state_row",
			"md_account_events",
			"md_earn_rate",
			"md_ticks",
			"md_equity_curve",
			"md_funding",
		},
		NoTrigger: []string{"ops_event"},
	})
}
