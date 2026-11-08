// excell_test.go —— T-0461 矩阵格集枚举器桩测：cell_key 唯一编码/
// Normalize 方言折格/枚举机生（占用+钦定+免演三源）/GreenKeys 判读。
package excell

import (
	"context"
	"testing"

	"o/internal/platform/execfam"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"

	"github.com/jackc/pgx/v5/pgxpool"
)

// setupExcell 测试基建（schema 隔离池+fin 迁移——testutil.Pool 不自动迁）。
func setupExcell(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	return ctx, pool
}

// 锚①：cell_key 唯一编码格式钉死（凭据/闸判共用——禁各处自拼）。
func TestCellKeyFormat(t *testing.T) {
	k := CellKey("funding-hedge", "okx", "short_perp_long_spot", 100)
	want := "funding-hedge@okx/short_perp_long_spot#usd=100"
	if k != want {
		t.Fatalf("cell_key=%q want %q", k, want)
	}
	// 面值去尾零（50.0→50——凭据键稳定不随浮点格式漂移）。
	if got := CellKey("xvenue-carry", VenuePairXvenue, DirLongPerpShortCFD, XvenueNotional); got !=
		"xvenue-carry@mt5xokx/long_perp_short_cfd#usd=50" {
		t.Fatalf("xvenue cell_key=%q", got)
	}
}

// 锚②：Normalize 方言折格——xvenue 副腿行 venue='mt5' 向位 short_cfd/
// long_cfd 折回 pair 向（与主行/proposal 同格——双腿不重格不漏格）。
func TestNormalizeXvenueFold(t *testing.T) {
	vp, dir := Normalize("xvenue-carry", "okx", "long_perp_short_cfd")
	if vp != VenuePairXvenue || dir != DirLongPerpShortCFD {
		t.Fatalf("主行归一=%s/%s", vp, dir)
	}
	vp, dir = Normalize("xvenue-carry", "mt5", "short_cfd")
	if vp != VenuePairXvenue || dir != DirLongPerpShortCFD {
		t.Fatalf("副腿行归一=%s/%s（应折回 pair 向）", vp, dir)
	}
	_, dir = Normalize("xvenue-carry", "mt5", "long_cfd")
	if dir != DirShortPerpLongCFD {
		t.Fatalf("副腿行归一向=%s", dir)
	}
	// 非 xvenue：venue 空=okx 兜底。
	vp, _ = Normalize("funding-hedge", "", "short_perp_long_spot")
	if vp != "okx" {
		t.Fatalf("单所归一 venue=%s", vp)
	}
}

// 锚③：枚举机生——execfam 词表零占模板=免演登记不产凭据；钦定
// xvenue 双向格无条件在册（$50 单档）；占用格两档=min+中位名义。
func TestEnumerateMandatedAndExempt(t *testing.T) {
	ctx, pool := setupExcell(t)
	cells, err := Enumerate(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	// 钦定 xvenue 双向格在册（无占用也在——部署前置凭据必跑）。
	idx := IndexByCoord(cells)
	for _, dir := range []string{DirLongPerpShortCFD, DirShortPerpLongCFD} {
		c := idx[CoordKey("xvenue-carry", VenuePairXvenue, dir)]
		if c == nil {
			t.Fatalf("钦定格缺 %s", dir)
		}
		if c.Exempt || len(c.Tiers) != 1 || c.Tiers[0].Notional != XvenueNotional {
			t.Fatalf("钦定格档位异常: %+v", c.Tiers)
		}
	}
	// execfam 词表全覆盖：册内每模板要么有占用格要么有免演登记——
	// 零占用不静默漏（枚举式落账断言）。
	seen := map[string]bool{}
	for _, c := range cells {
		seen[c.Template] = true
	}
	for _, tmpl := range execfam.All() {
		if !seen[tmpl] {
			t.Fatalf("execfam 模板 %s 矩阵面缺席（零占应免演登记）", tmpl)
		}
	}
	// 序定排——cell_key 序稳定（审计/断言面防漂移）。
	for i := 1; i < len(cells); i++ {
		a, b := cells[i-1], cells[i]
		if a.Template > b.Template ||
			(a.Template == b.Template && a.VenuePair > b.VenuePair) ||
			(a.Template == b.Template && a.VenuePair == b.VenuePair && a.Direction > b.Direction) {
			t.Fatalf("格序非定排: %d↔%d", i-1, i)
		}
	}
}

// 锚④：占用格两档=min 在产+中位名义（夹具策略行占位——档位键=
// 名义值去重；提案-only 格回退纸面名义档）。
func TestEnumerateTiers(t *testing.T) {
	ctx, pool := setupExcell(t)
	// 夹具：funding-hedge 三行不同名义占位（50/100/200——min=50 med=100
	// 上中位档；同 venue+inst+dir+template 唯一约束下异 inst_id 绕开）。
	for i, n := range []float64{50, 100, 200} {
		inst := "TIER" + string(rune('A'+i)) + "-USDT-SWAP"
		if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy
			(inst_id,template_ref,state,notional_usd,direction,venue)
			VALUES($1,'funding-hedge','paper',$2,'short_perp_long_spot','okx')`, inst, n); err != nil {
			t.Fatal(err)
		}
		defer func() {
			_, _ = pool.Exec(ctx, `DELETE FROM fin_strategy WHERE inst_id=$1`, inst)
		}()
	}
	cells, err := Enumerate(ctx, pool)
	if err != nil {
		t.Fatal(err)
	}
	idx := IndexByCoord(cells)
	c := idx[CoordKey("funding-hedge", "okx", "short_perp_long_spot")]
	if c == nil || !c.Occupied {
		t.Fatal("占用格未枚举")
	}
	// 档数=去重后 min+med——共享测试库他包残留行可能并入 notionals
	// （隔离 schema 内仅本包——但跨测试残留策略行同池可见），断言锚
	// =档集必含 50（min）且档数≤2 且全档 cell_key 可编码。
	var has50 bool
	for _, tr := range c.Tiers {
		if tr.Notional == 50 {
			has50 = true
		}
	}
	if !has50 {
		t.Fatalf("min 档 50 缺席: %+v", c.Tiers)
	}
	if len(c.Tiers) > 2 {
		t.Fatalf("档数超两档钉死: %+v", c.Tiers)
	}
	for _, k := range c.Keys() {
		if k == "" {
			t.Fatal("cell_key 空串")
		}
	}
}

// 锚⑤：GreenKeys/CellGreen 判读——缺行/非 pass=false fail-closed；
// 全档 latest=pass 才绿（任一档败=格不绿）。
func TestGreenKeysFailClosed(t *testing.T) {
	ctx, pool := setupExcell(t)
	cells := []Cell{{
		Template: "funding-hedge", VenuePair: "okx", Direction: "short_perp_long_spot",
		Tiers: []Tier{{Notional: 100}, {Notional: 200}},
	}}
	green, err := GreenKeys(ctx, pool, cells)
	if err != nil {
		t.Fatal(err)
	}
	if CellGreen(cells[0], green) {
		t.Fatal("无凭据格应不绿（fail-closed）")
	}
	// 单档 pass+单档缺=仍不绿（全档齐绿律）。
	k0 := cells[0].Keys()[0]
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
		VALUES(0,$1,'pass','test','', $1)`, k0); err != nil {
		t.Fatal(err)
	}
	green, _ = GreenKeys(ctx, pool, cells)
	if CellGreen(cells[0], green) {
		t.Fatal("半档绿应不绿（全档齐绿律）")
	}
	// 补第二档 pass→绿。
	k1 := cells[0].Keys()[1]
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
		VALUES(0,$1,'pass','test','', $1)`, k1); err != nil {
		t.Fatal(err)
	}
	green, _ = GreenKeys(ctx, pool, cells)
	if !CellGreen(cells[0], green) {
		t.Fatal("全档 pass 应绿")
	}
	// latest=fail 翻案——新 fail 行压过 pass=不绿（latest 判词律）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
		VALUES(0,$1,'fail','test','', $1)`, k0); err != nil {
		t.Fatal(err)
	}
	green, _ = GreenKeys(ctx, pool, cells)
	if CellGreen(cells[0], green) {
		t.Fatal("latest=fail 应不绿")
	}
	// 免演登记格=免凭据直绿（零占用无提案可放）。
	ex := Cell{Template: "vol_sell_strangle", Exempt: true}
	if !CellGreen(ex, green) {
		t.Fatal("免演格应直绿")
	}
}
