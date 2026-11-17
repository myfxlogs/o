package simdrill

import (
	"testing"
)

// 工单#22576 测锚（验收③）：演习开平一轮后 venue<>'sim' 现金流零新增——
// 腿级现金行/滑点采样 venue 列归 sim 域（WithBookVenue 覆写径），
// venue<>'sim' 谓词机器口径不再吞演习流水。
func TestDrillCashflowSimVenueDomain(t *testing.T) {
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
		t.Fatalf("CORE 案全径应绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	// 本钻腿级现金行（source_ref=腿行 id——验收①同式归属键）：
	// venue<>'sim' 必须零行；venue='sim' 必须有行（写径活证——spot 买+平两腿）。
	var realN, simN int
	if err := d.Pool.QueryRow(ctx, `SELECT
		count(*) FILTER (WHERE c.venue<>'sim'),
		count(*) FILTER (WHERE c.venue='sim')
		FROM fin_cashflow c JOIN fin_order_leg l
		  ON c.source_table='fin_order_leg' AND c.source_ref=l.id
		WHERE l.execution_id IN ($1,$2)`, res.OpenExecID, res.CloseExecID).
		Scan(&realN, &simN); err != nil {
		t.Fatal(err)
	}
	if realN != 0 {
		t.Fatalf("演习腿现金行 venue<>'sim' 残留 %d 行（验收③破——sim 域覆写失效）", realN)
	}
	if simN < 2 {
		t.Fatalf("演习腿现金行 venue='sim' 应≥2（spot_buy+spot_sell 写径活证）: %d", simN)
	}
	// 仓级归属现金行（position_ref——费行/补链行同域断言）。
	var posID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_position WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`,
		res.OpenExecID).Scan(&posID); err != nil {
		t.Fatal(err)
	}
	var posRealN int
	if err := d.Pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_cashflow WHERE position_ref=$1 AND venue<>'sim'`, posID).
		Scan(&posRealN); err != nil {
		t.Fatal(err)
	}
	if posRealN != 0 {
		t.Fatalf("演习仓归属现金行 venue<>'sim' 残留 %d 行", posRealN)
	}
	// 滑点采样同域（fin_slip_sample.venue——演习 fill 不簿记真 venue）。
	var slipRealN int
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM fin_slip_sample s
		JOIN fin_order_leg l ON s.leg_id=l.id
		WHERE l.execution_id IN ($1,$2) AND s.venue<>'sim'`,
		res.OpenExecID, res.CloseExecID).Scan(&slipRealN); err != nil {
		t.Fatal(err)
	}
	if slipRealN != 0 {
		t.Fatalf("演习腿滑点采样 venue<>'sim' 残留 %d 行", slipRealN)
	}
	// 路由真相不覆写：fin_order_leg.venue 保 'okx'（guard 平仓腿回读须路由得回）。
	var legVenue string
	if err := d.Pool.QueryRow(ctx,
		`SELECT venue FROM fin_order_leg WHERE execution_id=$1 LIMIT 1`, res.OpenExecID).
		Scan(&legVenue); err != nil {
		t.Fatal(err)
	}
	if legVenue != "okx" {
		t.Fatalf("fin_order_leg.venue 应保路由真相 okx: %q", legVenue)
	}
}
