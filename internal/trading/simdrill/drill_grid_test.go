// drill_grid_test.go —— ShapeGridBot bot 全径演习锚（O-41370——
// grid_spot Drillable 翻真+armed→drill 转段凭据产能）。
package simdrill

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/excell"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/okxgrid"
	"o/internal/trading/simvenue"
)

// 锚①：grid_spot LegShapeOf→ShapeGridBot+Drillable 翻真（probe→live
// 毕业链 Drillable 闸不再 exec_drill_unshaped 拦——armed→drill 转段
// 触达的第一块砖）。
func TestGridSpotDrillable(t *testing.T) {
	shape, ok := excell.LegShapeOf("grid_spot")
	if !ok || shape != excell.ShapeGridBot {
		t.Fatalf("grid_spot 腿形应=grid_bot: %v %v", shape, ok)
	}
	if !excell.Drillable("grid_spot") {
		t.Fatal("grid_spot Drillable 应翻真（bot 全径演习产能在册）")
	}
}

// 锚②：venue 全径绿——Launch→挂单在册→喂价成交→Stop→残留清零→
// 收割径通（simgrid 引擎+产线 Adapter 五端点同码）。
func TestGridVenueDrillPass(t *testing.T) {
	v, det, trg := gridVenueDrill(context.Background(), okxgrid.LaunchReq{
		InstID: "BTC-USDT", MinPx: 80000, MaxPx: 100000,
		GridNum: 10, QuoteSz: 100, AlgoClID: "ogtest", Tag: "okxgrid"})
	if v != "pass" {
		t.Fatalf("grid bot 全径应绿: %s %s", v, det)
	}
	if trg == "" {
		t.Fatal("trigger 应记喂价成交锚")
	}
}

// 锚②b：参数非法=venue 拒→fail 凭据语义（band 反/档数<2/投≤0——
// simgrid 校验面与产线 LaunchReq 契约同径）。
func TestGridVenueDrillBadParams(t *testing.T) {
	v, det, _ := gridVenueDrill(context.Background(), okxgrid.LaunchReq{
		InstID: "BTC-USDT", MinPx: 100000, MaxPx: 80000,
		GridNum: 10, QuoteSz: 100})
	if v != "fail" {
		t.Fatalf("band 反应判 fail: %s %s", v, det)
	}
}

// gridSetup 策略级演习夹具（fin_opp_eval 判词证据带 band/档数/capital
// ——runGridDrill 读链=strategy→proposal→eval 同 launchGate 形）。
func gridSetup(t *testing.T) (context.Context, *Drill, int64) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref='grid_spot')`,
		`DELETE FROM fin_proposal WHERE inst_id='BTC-USDT' AND direction='grid_spot'`,
		`DELETE FROM fin_opp_eval WHERE inst_id='BTC-USDT' AND template='grid_spot'`,
		`DELETE FROM fin_strategy WHERE template_ref='grid_spot'`,
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1e9, nil })
	var eid, pid, sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('BTC-USDT','grid_spot','grid_spot',0.08,'proposed','{}',
		'{"band_lo":80000,"band_hi":100000,"n_lines":10,"capital":100,"last_close":90000}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,'BTC-USDT','grid_spot',0.08) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,direction,venue)
		VALUES('BTC-USDT','grid_spot',$1,'probe',100,'grid_spot','okx') RETURNING id`, pid).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return ctx, &Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: simvenue.New()}, sid
}

// 锚③：策略级全径——Run→runGridDrill→bot 拉起/挂单/喂价/kill 绿→
// fin_exec_drill 凭据 pass 落账（毕业闸 execDrillPass 可读径同构）。
func TestDrillGridBotFullPath(t *testing.T) {
	ctx, d, sid := gridSetup(t)
	res, err := d.Run(ctx, sid)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "pass" {
		t.Fatalf("grid bot 全径应绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	var v string
	if err := d.Pool.QueryRow(ctx,
		`SELECT verdict FROM fin_exec_drill WHERE strategy_ref=$1 ORDER BY id DESC LIMIT 1`,
		sid).Scan(&v); err != nil || v != "pass" {
		t.Fatalf("凭据应落 pass: %v %q", err, v)
	}
}

// 锚③b：判词证据参数不合=fail 凭据落账（fail-closed——闸读败因不晋，
// 不静默漏；证据缺=演习无参可推→机械败 err 上抛同 drill 律）。
func TestDrillGridBadEvidence(t *testing.T) {
	ctx, d, sid := gridSetup(t)
	if _, err := d.Pool.Exec(ctx,
		`UPDATE fin_opp_eval SET evidence='{"band_lo":0}' WHERE inst_id='BTC-USDT' AND template='grid_spot'`); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, sid)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "fail" {
		t.Fatalf("证据参数不合应判 fail: %s", res.Verdict)
	}
	var v string
	if err := d.Pool.QueryRow(ctx,
		`SELECT verdict FROM fin_exec_drill WHERE strategy_ref=$1 ORDER BY id DESC LIMIT 1`,
		sid).Scan(&v); err != nil || v != "fail" {
		t.Fatalf("fail 凭据应落账: %v %q", err, v)
	}
}
