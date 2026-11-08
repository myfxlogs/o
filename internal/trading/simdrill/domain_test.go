package simdrill

import (
	"testing"
)

// T-0489 径A 红桩：simvenue 仓写径分域——sim 产跑仓行落 sim_* 命名域表，
// fin_* 正本真户面零新增；postmortem 队列对 sim 仓零立案。
// 修前红：现码直写 fin_* 正本（病灶实证 fin_position #35-39——业主对账扑空/
// 诊断空烧/T-0145 判据腐蚀）；修后同断言绿。
func TestSimDomainWriteIsolation(t *testing.T) {
	ctx, d := setup(t)
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='simdrill-gate' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "pass" {
		t.Fatalf("全径应绿（分域不改演习语义）: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	// 锚①：真户五面零新增——fin_* 正本只纳真户（判据：sim 产跑后 fin_position
	// 真户面零新增；腿/现金/守卫账同域伴迁防 id 撞号回染）。
	for _, tbl := range []string{
		"fin_execution", "fin_position", "fin_order_leg", "fin_cashflow", "fin_guard_close",
	} {
		var n int
		if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM `+tbl).Scan(&n); err != nil {
			t.Fatalf("%s 读败: %v", tbl, err)
		}
		if n != 0 {
			t.Fatalf("sim 仓污染正本：%s 有 %d 行（真户面应零新增）", tbl, n)
		}
	}
	// 锚②：sim 域五表行齐——仓/执行/腿/现金/守卫账全簇闭合同域。
	var simPos int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM sim_position WHERE execution_id=$1`, res.OpenExecID).Scan(&simPos); err != nil {
		t.Fatalf("sim 仓行缺（open exec=#%d）: %v", res.OpenExecID, err)
	}
	var simExecState string
	if err := d.Pool.QueryRow(ctx,
		`SELECT state FROM sim_execution WHERE id=$1`, res.OpenExecID).Scan(&simExecState); err != nil {
		t.Fatalf("sim 执行行缺: %v", err)
	}
	var legs, flows int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM sim_order_leg WHERE execution_id IN ($1,$2)`,
		res.OpenExecID, res.CloseExecID).Scan(&legs); err != nil || legs < 4 {
		t.Fatalf("sim 腿行缺（开平双腿应≥4）: legs=%d err=%v", legs, err)
	}
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM sim_cashflow WHERE position_ref=$1`, simPos).Scan(&flows); err != nil || flows == 0 {
		t.Fatalf("sim 现金行缺（费/腿级对称行应落 sim 域）: flows=%d err=%v", flows, err)
	}
	var closed bool
	if err := d.Pool.QueryRow(ctx,
		`SELECT closed_at IS NOT NULL FROM sim_position WHERE id=$1`, simPos).Scan(&closed); err != nil || !closed {
		t.Fatalf("sim 仓应闭环: closed=%v err=%v", closed, err)
	}
	// 锚③：postmortem/D1 对 sim 仓零立案——诊断队列不见仓案（诊断额度空烧
	// 病灶根修；判据钦定「postmortem 队列对 sim 仓零立案」）。
	var cases int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_econ_lesson WHERE kind='trade_postmortem'`).Scan(&cases); err != nil {
		t.Fatal(err)
	}
	if cases != 0 {
		t.Fatalf("sim 仓进 postmortem 立案队列: %d 案（sim 域应天然不入）", cases)
	}
}
