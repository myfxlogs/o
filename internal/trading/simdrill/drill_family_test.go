// drill_family_test.go —— T-0522 假想卡骨架批演习产能扩域桩测：
// basis 单钻（期货空+现货/perp 多——leggedDrill 共用体腿型分判锚）、
// vol_sell DCD 径（strangle 复合键双腿/single_call 单腿）、
// quarter_basis_short 显式拒（margin-borrow 面缺 fail-closed 不构错腿）。
// 同坑戒：KB simdrill-gate-domain-coverage-mismatch——毕业域新增族=
// 演习产能同 commit 补格，本件即补格锚。
package simdrill

import (
	"context"
	"strings"
	"testing"

	"o/internal/platform/excell"
	"o/internal/trading/simvenue"
)

// mkFamilyStrat 被闸族策略夹具（probe 态+手术级清场——drill_test 同律）。
func mkFamilyStrat(t *testing.T, ctx context.Context, d *Drill, inst, tmpl string, notional float64) int64 {
	t.Helper()
	var id int64
	if err := d.Pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES($1,$2,'probe',$3,'okx') RETURNING id`, inst, tmpl, notional).Scan(&id); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = d.Pool.Exec(context.Background(),
			`DELETE FROM fin_exec_drill WHERE strategy_ref=$1`, id)
		_, _ = d.Pool.Exec(context.Background(),
			`DELETE FROM fin_strategy WHERE id=$1`, id)
		for _, pid := range strings.Split(inst, "+") {
			_, _ = d.Pool.Exec(context.Background(),
				`DELETE FROM md_dcd_product WHERE venue='sim' AND product_id=$1`, pid)
		}
	})
	return id
}

// assertDrillPass 绿径公共断言（verdict=pass+凭据 verdict/rules 落账——
// 毕业闸消费面锚）。
func assertDrillPass(t *testing.T, ctx context.Context, d *Drill, stratID int64) Result {
	t.Helper()
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "pass" {
		t.Fatalf("全径应绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	var verdict, rv string
	if err := d.Pool.QueryRow(ctx,
		`SELECT verdict,rules_version FROM fin_exec_drill WHERE strategy_ref=$1 ORDER BY id DESC LIMIT 1`,
		stratID).Scan(&verdict, &rv); err != nil {
		t.Fatal(err)
	}
	if verdict != "pass" || rv != simvenue.RulesVersion {
		t.Fatalf("凭据: verdict=%s rules=%s", verdict, rv)
	}
	return res
}

// basis 单钻全径：期货空腿+现货买腿双腿原子→哨戒平→对拍四锚腿型分判
// （SPOT 多头腿=裸袋锚/FUTURES 衍生腿=净持锚）→凭据——T-0522 前 basis
// 单钻撞「演习面限」fail-forever 域。
func TestDrillBasisLongFullPath(t *testing.T) {
	ctx, d := setup(t)
	inst := "BTC-USDT-260328"
	d.Venue = VenueForInst(inst)
	stratID := mkFamilyStrat(t, ctx, d, inst, "quarter_basis_long", 50)
	assertDrillPass(t, ctx, d, stratID)
	// venue 真相：期货净持归零+现货裸袋清零（flat 双锚腿型分判件）。
	if p := d.Venue.Position(inst); p != 0 {
		t.Fatalf("期货净持应归零: %v", p)
	}
	if b := d.Venue.Balance("BTC"); b > 0.001 {
		t.Fatalf("现货裸袋应清零: %.8f", b)
	}
}

// perp_quarter_spread 单钻全径：期货空+perp 买双腿（多头腿合约域——
// recon 按 InstType 走净持锚非裸袋锚）。
func TestDrillPerpQuarterSpreadFullPath(t *testing.T) {
	ctx, d := setup(t)
	inst := "BTC-USDT-260328"
	d.Venue = VenueForInst(inst)
	stratID := mkFamilyStrat(t, ctx, d, inst, "perp_quarter_spread", 50)
	assertDrillPass(t, ctx, d, stratID)
	if p := d.Venue.Position(inst); p != 0 {
		t.Fatalf("期货净持应归零: %v", p)
	}
	if p := d.Venue.Position("BTC-USDT-SWAP"); p != 0 {
		t.Fatalf("perp 净持应归零: %v", p)
	}
}

// quarter_basis_short 显式拒（margin-borrow 面缺——不构错腿不产绿凭据；
// 机械败上抛禁假凭据——卡#5 park 裁词事实锚）。
func TestDrillQuarterBasisShortRejected(t *testing.T) {
	ctx, d := setup(t)
	inst := "BTC-USDT-260328"
	d.Venue = VenueForInst(inst)
	stratID := mkFamilyStrat(t, ctx, d, inst, "quarter_basis_short", 50)
	_, err := d.Run(ctx, stratID)
	if err == nil || !strings.Contains(err.Error(), "quarter_basis_short 演习不支持") {
		t.Fatalf("qbs 应显式拒（margin-borrow 缺）: %v", err)
	}
}

// vol_sell_strangle 单钻全径：复合键「Cprod+Pprod」拆双腿——各腿等权名义
// 逐腿开/平（volSellStrangle 执行器同构）；T-0522 前 vol_sell 误映
// okx_two_leg 拿产品键当现货键跑错腿。
func TestDrillVolSellStrangleFullPath(t *testing.T) {
	ctx, d := setup(t)
	inst := "BTC-C-150+BTC-P-100"
	stratID := mkFamilyStrat(t, ctx, d, inst, "vol_sell_strangle", 60)
	res := assertDrillPass(t, ctx, d, stratID)
	if !strings.Contains(res.Detail, "legs=2") {
		t.Fatalf("strangle 应双腿径: %s", res.Detail)
	}
	// pass 内含双腿全径闭环（每腿 LIVE 校验+态迁+残留锚——见 runDCDDrill）。
}

// vol_sell_single_call 单钻全径：单产品键单腿径（strangle 拆腿件复用——
// n=1 退化域）。
func TestDrillVolSellSingleCallFullPath(t *testing.T) {
	ctx, d := setup(t)
	stratID := mkFamilyStrat(t, ctx, d, "BTC-C-150", "vol_sell_single_call", 60)
	assertDrillPass(t, ctx, d, stratID)
}

// strangle 复合键形拒（非 C+P 形禁跑错腿——fail-closed 判词锚）。
func TestDrillVolSellStrangleBadKey(t *testing.T) {
	ctx, d := setup(t)
	stratID := mkFamilyStrat(t, ctx, d, "BTC-C-150", "vol_sell_strangle", 60)
	_, err := d.Run(ctx, stratID)
	if err == nil || !strings.Contains(err.Error(), "复合键形不符") {
		t.Fatalf("单键 strangle 应拒: %v", err)
	}
}

// 矩阵格：vol_sell_strangle 占用格走 shapeDCD 复合键径（T-0522 前误映
// okx_two_leg——产品键当现货键跑错腿的回归锚）。
func TestMatrixVolSellStrangleCell(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "BTC-C-150+BTC-P-100", "vol_sell_strangle", "vol_sell", "okx", 60)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	c := excell.IndexByCoord(cells)[excell.CoordKey("vol_sell_strangle", "okx", "vol_sell")]
	if c == nil {
		t.Fatal("strangle 格未枚举")
	}
	for _, tr := range c.Tiers {
		res := m.runCell(ctx, *c, tr, shapeDCD)
		if res.Verdict != "pass" {
			t.Fatalf("strangle 格档 %.0f 败: %s", tr.Notional, res.Detail)
		}
	}
}

// 矩阵格：quarter_basis_short 占用格=显式拒凭据（fail 落账非静默——
// margin-borrow 面缺诚实判词，卡#5 park 同锚）。
func TestMatrixQuarterBasisShortCell(t *testing.T) {
	ctx, m := setupMatrix(t)
	mkOcc(t, ctx, m, "BTC-USDT-260328", "quarter_basis_short", "basis_quarter", "okx", 60)
	cells, err := excell.Enumerate(ctx, m.Pool)
	if err != nil {
		t.Fatal(err)
	}
	c := excell.IndexByCoord(cells)[excell.CoordKey("quarter_basis_short", "okx", "basis_quarter")]
	if c == nil {
		t.Fatal("qbs 格未枚举")
	}
	res := m.runCell(ctx, *c, c.Tiers[0], shapeOKXTwoLeg)
	if res.Verdict != "fail" || !strings.Contains(res.Detail, "quarter_basis_short 演习不支持") {
		t.Fatalf("qbs 格应 fail+显式拒词: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
}
