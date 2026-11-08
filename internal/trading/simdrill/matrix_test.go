// matrix_test.go —— T-0461 矩阵批跑桩测：腿形注册覆盖/钦定格枚举/xvenue
// 腿序+ticket 全周期/双腿残留=0/凭据 cell_key 落账/armed→open 闸已在
// lifecycle 面锚（opener_test.TestExecCellHoldGate）。
package simdrill

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/excell"
	"o/internal/platform/execfam"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
)

// 锚①：腿形注册覆盖——execfam 词表全模板有腿形映射（新增族未登记=
// 矩阵面缺口 fail-closed 不静默漏）。
func TestLegShapeCoverage(t *testing.T) {
	for _, tmpl := range execfam.All() {
		if _, ok := legShapeOf(tmpl); !ok {
			t.Fatalf("execfam 模板 %s 腿形未登记", tmpl)
		}
	}
	// 钦定映射抽查：funding/basis/xvenue/dcd 四族归位正确。
	cases := map[string]legShape{
		"funding-hedge": shapeOKXTwoLeg, "quarter_basis_long": shapeOKXTwoLeg,
		"perp_quarter_spread": shapeOKXTwoLeg, "dcd_put": shapeDCD,
		"dcd_call": shapeDCD, "xvenue-carry": shapeXvenue,
	}
	for tmpl, want := range cases {
		if got, _ := legShapeOf(tmpl); got != want {
			t.Fatalf("%s 腿形=%s want %s", tmpl, got, want)
		}
	}
}

// 锚②：xvenue 腿序+ticket 全周期——BuildXvenueLegs 产 mt5 腿先+okx 腿后；
// mt5sim OrderSend→ticket→OpenedOrders→OrderClose→票消残史全周期；
// mtexec.Executor 经 RouterExecutor 分派产线同码。
func TestXvenueLegOrderAndTicketLifecycle(t *testing.T) {
	mt := newMT5Venue("BTC-USDT-SWAP")
	cell := excell.Cell{Template: "xvenue-carry", VenuePair: excell.VenuePairXvenue,
		Direction: excell.DirLongPerpShortCFD}
	legs, err := xvenueLegs(cell, excell.Tier{Notional: 50}, "BTC-USDT-SWAP", mt)
	if err != nil {
		t.Fatal(err)
	}
	if legs[0].Venue != "mt5" || legs[1].Venue != "okx" {
		t.Fatalf("腿序应 mt5 先+okx 后（DM 裁定钉死）: %s→%s", legs[0].Venue, legs[1].Venue)
	}
	if legs[0].Side != "sell" || legs[1].Side != "buy" {
		t.Fatalf("long_perp_short_cfd 向位=mt5卖+okx买: %s/%s", legs[0].Side, legs[1].Side)
	}
	// 反向格：mt5买+okx卖。
	cell.Direction = excell.DirShortPerpLongCFD
	legs, err = xvenueLegs(cell, excell.Tier{Notional: 50}, "BTC-USDT-SWAP", mt)
	if err != nil {
		t.Fatal(err)
	}
	if legs[0].Side != "buy" || legs[1].Side != "sell" {
		t.Fatalf("short_perp_long_cfd 向位=mt5买+okx卖: %s/%s", legs[0].Side, legs[1].Side)
	}
}

// setupMatrix 矩阵测试基建（drill_test 同配置域+隔离清理——本包 schema
// 专属整面重置：占用格夹具跨测残留会污染后续 Enumerate 全谱断言）。
func setupMatrix(t *testing.T) (context.Context, *Matrix) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DELETE FROM fin_strategy_transition`,
		`DELETE FROM fin_paper_fill`,
		`DELETE FROM fin_guard_close WHERE position_id IN (SELECT id FROM fin_position)`,
		`DELETE FROM fin_cashflow WHERE position_ref IN (SELECT id FROM fin_position)`,
		`DELETE FROM fin_position`,
		`DELETE FROM fin_order_leg WHERE execution_id IN (SELECT id FROM fin_execution)`,
		`DELETE FROM fin_execution`,
		`DELETE FROM fin_exec_drill`,
		`DELETE FROM fin_strategy`,
		`DELETE FROM md_dcd_product WHERE venue='sim'`,
		`DELETE FROM ops_event WHERE esc_class='执行径演习' AND ticket='T-0426'`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{
		"invest.lifecycle.live_switch": "true",
		"exec.hedge_band_bps":          "20",
		"guard.flip_bps":               "5", "guard.max_hold_hours": "72",
		"guard.retry_max_attempts": "3", "guard.retry_backoff_seconds": "0",
		"guard.bare_bag_min_usd": "1",
		"risk.kill_switch":       "false", "risk.max_notional_usd": "5000",
		"risk.max_venue_pct": "0.9", "risk.max_exposure_usd": "100000",
		"risk.token_ttl_seconds": "30",
		"risk.max_dd_30d":        "0.5", "risk.dd_window_days": "30"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1e9, nil })
	return ctx, &Matrix{Pool: pool, Cfg: cfg, Gate: gate}
}

// mkOcc 占用格夹具（fin_strategy paper 行——格枚举占用源；两档名义可编程）。
func mkOcc(t *testing.T, ctx context.Context, m *Matrix, inst, tmpl, dir, venue string, notionals ...float64) {
	t.Helper()
	for _, n := range notionals {
		if _, err := m.Pool.Exec(ctx, `INSERT INTO fin_strategy
			(inst_id,template_ref,state,notional_usd,direction,venue)
			VALUES($1,$2,'paper',$3,$4,$5)`, inst, tmpl, n, dir, venue); err != nil {
			t.Fatal(err)
		}
	}
}

// 锚③：矩阵全周期冒烟——钦定 xvenue 双向格空占用跑（$50 单档）：
// 双腿原子开→持→哨戒强平→对账→凭据 cell_key 落账；venue 真相
// （mt5 ticket 消+okx 净持归零）残留=0。
func TestMatrixXvenueCells(t *testing.T) {
	ctx, m := setupMatrix(t)
	// 只跑 xvenue 钦定格——其余占用格不造（词表零占=免演跳过）。
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	var xv []excell.Cell
	for _, c := range cells {
		if c.VenuePair == excell.VenuePairXvenue && !c.Exempt {
			xv = append(xv, c)
		}
	}
	if len(xv) != 2 {
		t.Fatalf("钦定 xvenue 格应 2 向: %d", len(xv))
	}
	for _, c := range xv {
		for _, tr := range c.Tiers {
			res := m.runCell(ctx, c, tr, shapeXvenue)
			if res.Verdict != "pass" {
				t.Fatalf("xvenue 格 %s 档 %.0f 败: %s", res.CellKey, tr.Notional, res.Detail)
			}
			// 凭据落账 cell_key（部署闸消费面）。
			var verdict, ck string
			if err := m.Pool.QueryRow(ctx, `SELECT verdict,cell_key FROM fin_exec_drill
				WHERE cell_key=$1 ORDER BY id DESC LIMIT 1`, res.CellKey).
				Scan(&verdict, &ck); err != nil || verdict != "pass" || ck != res.CellKey {
				t.Fatalf("凭据缺/不符: %q %q err=%v", verdict, ck, err)
			}
		}
	}
}

// 锚④：funding-hedge 占用格两档全周期——min+中位档各跑一遍；凭据
// cell_key=模板@okx/方向#usd=<档>；venue 残留归零（flatFn 内断）。
func TestMatrixFundingCellTwoTiers(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "FOO-USDT-SWAP", "funding-hedge", "short_perp_long_spot", "okx", 40)
	mkOcc(t, ctx, m, "BAR-USDT-SWAP", "funding-hedge", "short_perp_long_spot", "okx", 80)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	idx := excell.IndexByCoord(cells)
	c := idx[excell.CoordKey("funding-hedge", "okx", "short_perp_long_spot")]
	if c == nil {
		t.Fatal("funding-hedge 格未枚举")
	}
	if len(c.Tiers) != 2 {
		t.Fatalf("两档名义应两档: %+v", c.Tiers)
	}
	for _, tr := range c.Tiers {
		res := m.runCell(ctx, *c, tr, shapeOKXTwoLeg)
		if res.Verdict != "pass" {
			t.Fatalf("funding 格档 %.0f 败: %s", tr.Notional, res.Detail)
		}
		if !strings.Contains(res.CellKey, "#usd=") {
			t.Fatalf("cell_key 缺面值档: %s", res.CellKey)
		}
	}
}

// 锚⑤：DCD 全周期——申购（quote→trade→status LIVE）→hold→到期结算
// （SETTLED→exec/position 态迁）→残留=0；凭据 cell_key 落账。
func TestMatrixDcdLifecycle(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "DCDPUT-TEST-001", "dcd_put", "dcd_put", "okx", 60)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	idx := excell.IndexByCoord(cells)
	c := idx[excell.CoordKey("dcd_put", "okx", "dcd_put")]
	if c == nil {
		t.Fatal("dcd_put 格未枚举")
	}
	for _, tr := range c.Tiers {
		res := m.runCell(ctx, *c, tr, shapeDCD)
		if res.Verdict != "pass" {
			t.Fatalf("dcd 格档 %.0f 败: %s", tr.Notional, res.Detail)
		}
		// exec/position 态迁断言——SETTLED→closed 产线同律。
		var estate string
		var closed bool
		if err := m.Pool.QueryRow(ctx, `SELECT e.state::text,p.closed_at IS NOT NULL
			FROM fin_execution e JOIN fin_position p ON p.execution_id=e.id
			WHERE e.id=$1`, res.OpenExecID).Scan(&estate, &closed); err != nil {
			t.Fatal(err)
		}
		if estate != "closed" || !closed {
			t.Fatalf("dcd 结算态迁异常: exec=%s closed=%v", estate, closed)
		}
	}
}

// 锚⑥：basis 占用格——-YYMMDD 期货键腿构+全周期（quarter_basis_long
// 现货腿/perp_quarter_spread perp 腿两族登记面）。
func TestMatrixBasisCells(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "BTC-USDT-260328", "quarter_basis_long", "basis_quarter", "okx", 60)
	mkOcc(t, ctx, m, "ETH-USDT-260328", "perp_quarter_spread", "perp_quarter_spread", "okx", 60)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	idx := excell.IndexByCoord(cells)
	for _, coord := range [][2]string{
		{"quarter_basis_long", "basis_quarter"}, {"perp_quarter_spread", "perp_quarter_spread"}} {
		c := idx[excell.CoordKey(coord[0], "okx", coord[1])]
		if c == nil {
			t.Fatalf("basis 格未枚举 %s", coord[0])
		}
		for _, tr := range c.Tiers {
			res := m.runCell(ctx, *c, tr, shapeOKXTwoLeg)
			if res.Verdict != "pass" {
				t.Fatalf("basis 格 %s 档 %.0f 败: %s", res.CellKey, tr.Notional, res.Detail)
			}
		}
	}
}

// 锚⑦：Run 批跑入口全谱——免演格跳过不产凭据+占用格全档跑+汇总
// Failed=0+凭据行数=跑过格档数（格病=verdict fail 不炸批——批跑机
// 语义锚；单格径细节锚②~⑥）。
func TestMatrixRunBatch(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "BATCH-USDT-SWAP", "funding-hedge", "short_perp_long_spot", "okx", 55)
	res, err := m.Run(ctx)
	if err != nil {
		t.Fatalf("批跑机械败: %v", err)
	}
	if res.Failed != 0 {
		t.Fatalf("批跑应全绿: %+v", res.Results)
	}
	// 占用格 1 档+钦定 xvenue 2 格×1 档=3 凭据（免演词表格零产出）。
	if len(res.Results) != 3 {
		t.Fatalf("批跑格档数应=3（1 占用+2 钦定）: %+v", res.Results)
	}
	// 免演登记格在 Enumerate 可见但 Run 跳过——词表零占不产凭据。
	cells, _ := excell.Enumerate(ctx, m.Pool)
	exempt := 0
	for _, c := range cells {
		if c.Exempt {
			exempt++
		}
	}
	if exempt == 0 {
		t.Fatal("词表零占免演登记缺席")
	}
	var n int
	if err := m.Pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_exec_drill WHERE cell_key IS NOT NULL`).Scan(&n); err != nil || n != 3 {
		t.Fatalf("cell_key 凭据行数应=3: n=%d err=%v", n, err)
	}
}
