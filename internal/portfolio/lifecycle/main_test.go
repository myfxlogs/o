package lifecycle

import (
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// TestMain 包级共享 schema（工单#26108 per-test setup 税根治）——建面
// 一次+每测 Pool 定向清表代替逐测随机 schema+fin 全链迁移重施。
// SeedRestore 全量列（子→父序，清表两相位内部处理）：写面并集=
// 包直写表+fin 族 FK 邻表+ops_event 触发器波及表+econ_lesson→
// ops_decision 镜像触发器源表。app_config 键册由共享径自动快照
// 复位，不入列。NoTrigger=ops_event——sem_gate 拒写/act pkey/
// chain 改写等触发器群，种子还原 INSERT 须旁路。
// 序依据=fin_* 表间实测 FK 边（order_leg/position→execution、
// proposal→opp_eval、risk_token→risk_decision、transition/paper_fill
// →strategy、precedent→ops_event）。
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
			"fin_proposal",
			"fin_opp_eval",
			"fin_sub_request",
			"fin_eng_lesson",
			"fin_econ_lesson",
			"fin_econ_rule",
			"fin_eval_reject",
			"fin_exec_drill",
			"fin_slip_sample",
			"fin_trx_account",
			"fin_trx_txs",
			"fin_trx_vote_apr",
			"fin_cashflow",
			"fin_guard_suspend",
			"fin_holding",
			"fin_opp_template",
			"fin_opp_hypothesis",
			"fin_venue_candidate",
			"ops_decision",
			"ops_headless",
			"ops_event_act",
			"ops_precedent",
			"ops_event",
			"ops_state_row",
			"ops_face_snapshot",
			"md_calendar_pull",
			"md_eval_pulse",
			"md_dcd_product",
			"md_macro_event",
			"md_mark_index",
			"md_klines",
			"md_signal",
			"md_ticks",
			"md_tick_scan",
			"md_equity_curve",
			"md_funding",
			"fin_strategy",
		},
		NoTrigger: []string{"ops_event"},
	})
}
