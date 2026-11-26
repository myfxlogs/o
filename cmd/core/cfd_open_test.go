// cfd_open_test.go —— O-41368 CFD 腿构纯件锚：cfdLots lot 网格换算+
// min_lot fail-closed+参数非法拒；LegShapeOf/Drillable 注册锚。
package main

import (
	"context"
	"math"
	"testing"

	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/excell"
	"o/internal/portfolio/lifecycle"
)

func TestCfdLotsGridAndMinLot(t *testing.T) {
	// $30 名义/px=2000/ctVal=0.001/lotStep=0.001 → raw=15 手整网格命中。
	sz, err := cfdLots(30, 2000, 0.001, 0.001, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sz-15) > 1e-9 {
		t.Fatalf("lots 应=15: %.6f", sz)
	}
	// 非整网格向下取整（raw=15.0075→15.007——lotStep=0.001 网格 floor）。
	sz, err = cfdLots(30, 1999, 0.001, 0.001, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sz-15.007) > 1e-9 {
		t.Fatalf("非整网格应 floor=15.007: %.6f", sz)
	}
	// min_lot 起不来=fail-closed 报实数（$0.001 名义→raw=0.0005 手<0.001 网格 floor=0）。
	if _, err := cfdLots(0.001, 2000, 0.001, 0.001, 0.001); err == nil {
		t.Fatal("min_lot 不足应拒")
	}
	// 参数非法=fail-closed（px=0/ctVal=0/lotStep=0 三态全拒）。
	for _, c := range [][4]float64{{30, 0, 0.001, 0.001}, {30, 2000, 0, 0.001}, {30, 2000, 0.001, 0}} {
		if _, err := cfdLots(c[0], c[1], c[2], c[3], 0.001); err == nil {
			t.Fatalf("参数非法应拒: %v", c)
		}
	}
}

func TestMt5DirMomLegShapeRegistered(t *testing.T) {
	shape, ok := excell.LegShapeOf("mt5_dir_mom")
	if !ok || shape != excell.ShapeCFDSingleLeg {
		t.Fatalf("mt5_dir_mom 腿形应=ShapeCFDSingleLeg: %v %v", shape, ok)
	}
	if !excell.Drillable("mt5_dir_mom") {
		t.Fatal("mt5_dir_mom 应转 Drillable（paper-only→probe 可产——O-41368）")
	}
}

// fakeCFDSess/fakeCFDDial cfdVenueFor 桩测缝（xvenueSess/mtexec.Dialect
// 窄面假件——会话/方言译名不触网，方言仓走真 DialectStore 落库行）。
type fakeCFDSess struct{ acct string }

func (f fakeCFDSess) account(context.Context) (string, error)    { return f.acct, nil }
func (f fakeCFDSess) Authed(ctx context.Context) context.Context { return ctx }
func (f fakeCFDSess) Conn() *grpc.ClientConn                     { return nil }
func (f fakeCFDSess) MT5() mt5pb.MT5Client                       { return nil }
func (f fakeCFDSess) Suspect()                                   {}

type fakeCFDDial struct{}

func (fakeCFDDial) VenueSymbol(context.Context, string) (string, error) { return "X", nil }
func (fakeCFDDial) Escalate(context.Context, string, string)            {}

// cfdVenueFor 方言覆盖解析锚：'mt5' 域标签→名册 cfd 户首中方言行户
// （exness 先序）；单户覆盖=xmglobal 落位；全灭=fail-closed 列候选。
func TestCfdVenueForDialectCoverage(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	const canon = "ZZCFDTEST"
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fin_symbol_dialect WHERE canonical=$1`, canon)
	}()
	seedDialect := func(venue, acct string) {
		t.Helper()
		if _, err := pool.Exec(ctx, `INSERT INTO fin_symbol_dialect
			(venue,account,canonical,venue_symbol,ct_val,tick_sz,min_sz,lot_unit,lot_step,digits,trade_mode,spec_hash)
			VALUES($1,$2,$3,$4||'m',100000,0.00001,0.01,'lot',0.01,5,4,'h1')
			ON CONFLICT (venue,account,canonical) DO NOTHING`,
			venue, acct, canon, canon); err != nil {
			t.Fatal(err)
		}
	}
	accts := map[string]string{"exness": "3001157", "xmglobal": "110660304"}
	newFleet := func() *xvenueFleet {
		return &xvenueFleet{pool: pool, ds: mt5.NewDialectStore(pool),
			deps: map[string]*venueExecDeps{},
			newDep: func(v string) *venueExecDeps {
				return &venueExecDeps{venue: v, cli: fakeCFDSess{acct: accts[v]},
					dial: fakeCFDDial{}, ds: mt5.NewDialectStore(pool)}
			}}
	}
	h := hedgeOpener{pool: pool}
	spec := lifecycle.OpenSpec{Venue: "mt5", InstID: canon, Direction: "long_perp"}
	// ① exness 覆盖→exness 落位（先序钉死）。
	seedDialect("exness", accts["exness"])
	seedDialect("xmglobal", accts["xmglobal"])
	h.xf = newFleet()
	v, d, err := h.cfdVenueFor(ctx, spec)
	if err != nil || v != "exness" || d == nil {
		t.Fatalf("双户覆盖应解 exness: v=%q err=%v", v, err)
	}
	// ② 仅 xmglobal 覆盖→xmglobal 落位（exness 查无跳过不误判）。
	if _, err := pool.Exec(ctx, `DELETE FROM fin_symbol_dialect
		WHERE venue='exness' AND canonical=$1`, canon); err != nil {
		t.Fatal(err)
	}
	h.xf = newFleet()
	v, _, err = h.cfdVenueFor(ctx, spec)
	if err != nil || v != "xmglobal" {
		t.Fatalf("单户覆盖应解 xmglobal: v=%q err=%v", v, err)
	}
	// ③ 全灭=fail-closed（禁猜户）。
	if _, err := pool.Exec(ctx, `DELETE FROM fin_symbol_dialect WHERE canonical=$1`, canon); err != nil {
		t.Fatal(err)
	}
	h.xf = newFleet()
	if _, _, err := h.cfdVenueFor(ctx, spec); err == nil {
		t.Fatal("全户无方言覆盖应 fail-closed")
	}
}
