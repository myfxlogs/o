// residual_test.go —— 残币处置面桩测（T-0439 验收锚）：
//   - 白名单币停泊旧径零回归（摆渡+申购；拒=回摆+托管标）；
//   - 白名单外阈上=执行径市价清 USDT+首检升件行+spot_sell 现金流行；
//   - 粉尘阈下=托管如实标；豁免册在册=双径跳过；
//   - 清仓败/歧态=升件不补单同键节流；终拒不可卖=豁免册落行；
//   - 在持腿币不误动；白名单外零 Earn 钱动（扫袋面闸死）。
package earn

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// residualKeys 处置面判据键播种集（粉尘阈必播——缺键 fail-closed 处置让位）。
var residualKeys = map[string]string{"earn.residual_clear_min_usd": "1"}

// fakeClearer 可控清仓执行缝（调用记录+定真态注入——钱动锚全走此件）。
type fakeClearer struct {
	res        ClearResult
	err        error // Place 传输/装配败注入
	resolveRes ClearResult
	resolveErr error
	places     []ClearLeg
	resolves   int
}

func (f *fakeClearer) Place(_ context.Context, leg ClearLeg, clOrdID string) (ClearResult, error) {
	f.places = append(f.places, leg)
	if f.err != nil {
		return ClearResult{ClOrdID: clOrdID, State: "unknown"}, f.err
	}
	r := f.res
	r.ClOrdID = clOrdID
	return r, nil
}

func (f *fakeClearer) Resolve(_ context.Context, leg ClearLeg, clOrdID string) (ClearResult, error) {
	f.resolves++
	if f.resolveErr != nil {
		return ClearResult{ClOrdID: clOrdID, State: "unknown"}, f.resolveErr
	}
	r := f.resolveRes
	r.ClOrdID = clOrdID
	return r, nil
}

// seedWhitelist 播扫袋白名单 jsonb 数组键（setupW 标量播种只出 JSON 串——
// 数组键须直插原生 jsonb）+热载。
func seedWhitelist(t *testing.T, ctx context.Context, w *Worker, list string) {
	t.Helper()
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO app_config(key,value) VALUES('earn.sweep_whitelist',$1::jsonb)
		ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, list); err != nil {
		t.Fatal(err)
	}
	if err := w.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// residualSetup 开仓腿 fixture（venue 域=worker 同域——FundCcy venue 过滤对齐）。
func residualSetup(t *testing.T, ctx context.Context, pool *pgxpool.Pool, venue string) {
	t.Helper()
	var xid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT',100,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT',100) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,state)
		VALUES($1,1,'cl-res',$2,'buy','ETH-USDT',1,'filled')`, xid, venue); err != nil {
		t.Fatal(err)
	}
}

func custodyCount(t *testing.T, ctx context.Context, w *Worker, ccy string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_residual_custody WHERE venue=$1 AND ccy=$2`, w.venue, ccy).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func opsCount(t *testing.T, ctx context.Context, w *Worker, event, ccy string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM ops_event WHERE event=$1 AND note LIKE $2`,
		event, w.venue+" "+ccy+" %").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func exemptCount(t *testing.T, ctx context.Context, w *Worker, ccy string) int {
	t.Helper()
	var n int
	if err := w.pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_known_residual WHERE venue=$1 AND ccy=$2`, w.venue, ccy).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func seedTick(t *testing.T, ctx context.Context, w *Worker, inst string, px float64) {
	t.Helper()
	if _, err := w.pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,inst_id,last) VALUES(now(),$1,$2)`, inst, px); err != nil {
		t.Fatal(err)
	}
}

// 锚①：白名单币停泊旧径零回归——FIL 入白名单后够起投→摆渡+申购+凭据行
// （decision intent ccy=FIL；kind 空位零现金流）。
func TestResidualWhitelistPark(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	seedWhitelist(t, ctx, w, `["USDT","FIL"]`)
	seedTick(t, ctx, w, "FIL-USDT", 5)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"FIL": 200, "ETH": 0.1}, nil // ETH=在持腿币候选外
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 1 || fake.transfers[0] != "18->6:200.0000" {
		t.Fatalf("白名单币摆渡径: %v", fake.transfers)
	}
	if len(fake.purchases) != 1 || fake.purchases[0] != 200 {
		t.Fatalf("白名单币申购径: %v", fake.purchases)
	}
	var ccy string
	if err := w.pool.QueryRow(ctx, `SELECT intent->>'ccy' FROM fin_risk_decision
		WHERE reason='transfer' ORDER BY id DESC LIMIT 1`).Scan(&ccy); err != nil || ccy != "FIL" {
		t.Fatalf("凭据行 ccy=%q err=%v（应 FIL——停泊留真）", ccy, err)
	}
	if n := len(cashflowOf(t, ctx, w, "earn_in")) + len(cashflowOf(t, ctx, w, "earn_out")); n != 0 {
		t.Fatalf("零换手不得落现金流行: %d", n)
	}
	if custodyCount(t, ctx, w, "FIL") != 0 {
		t.Fatal("申购成不落托管标")
	}
}

// 锚②：白名单币申购业务拒=确认未动回摆补偿+托管标（停泊径语义守恒）。
func TestResidualWhitelistParkRejectCompensates(t *testing.T) {
	ctx, w, fake := setupW(t, &fakeAPI{purchaseErr: errors.New("58110 risk locked")}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	seedWhitelist(t, ctx, w, `["USDT","FIL"]`)
	seedTick(t, ctx, w, "FIL-USDT", 5)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"FIL": 200}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(fake.transfers) != 2 || fake.transfers[1] != "6->18:200.0000" {
		t.Fatalf("申购拒须回摆补偿: %v", fake.transfers)
	}
	if custodyCount(t, ctx, w, "FIL") != 1 {
		t.Fatal("申购拒须落托管标")
	}
}

// 锚③：白名单外阈上杂币=执行径市价清 USDT——Place 卖腿（CORE-USDT sell
// Close=true）+首检升件行+decision/spot_sell 双留痕；Earn 扫袋零钱动。
func TestResidualClearPath(t *testing.T) {
	clr := &fakeClearer{res: ClearResult{State: "filled", FillPx: 0.0198, FillSz: 2000}}
	ctx, w, fake := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "CORE-USDT", 0.02)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"CORE": 2000}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 1 {
		t.Fatalf("清仓应 Place 一次: %v", clr.places)
	}
	leg := clr.places[0]
	if leg.InstID != "CORE-USDT" || leg.Side != "sell" || !leg.Close || leg.Sz != "2000" {
		t.Fatalf("清仓腿契约破: %+v", leg)
	}
	if opsCount(t, ctx, w, "杂币敞口", "CORE") != 1 {
		t.Fatal("首检升件行未落")
	}
	var op string
	if err := w.pool.QueryRow(ctx, `SELECT intent->>'op' FROM fin_risk_decision
		WHERE reason='transfer' ORDER BY id DESC LIMIT 1`).Scan(&op); err != nil || op != "residual_clear" {
		t.Fatalf("清仓凭据 op=%s err=%v", op, err)
	}
	cf := cashflowOf(t, ctx, w, "spot_sell")
	if len(cf) != 1 || fmt.Sprintf("%.2f", cf[0]) != "39.60" {
		t.Fatalf("spot_sell 现金流行=%v（应 2000×0.0198=39.60）", cf)
	}
	if len(fake.transfers) != 0 || len(fake.purchases) != 0 {
		t.Fatalf("白名单外 Earn 扫袋须零钱动: %v %v", fake.transfers, fake.purchases)
	}
}

// 锚④：粉尘/无价两岔钉死——有价阈下=托管如实标非清仓非升件（挂账待归并）；
// 无价（usd=0）=敞口不可证小→升件+清仓径（保守侧——不可拿估值缺席当粉尘藏）。
func TestResidualDustCustody(t *testing.T) {
	clr := &fakeClearer{res: ClearResult{State: "absent", VenueCode: "51120", VenueMsg: "too small"}}
	ctx, w, fake := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "DUSTY-USDT", 0.0001)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"DUSTY": 500, "DOGE": 5}, nil
		// DUSTY=$0.05 有价阈下→托管标；DOGE 无 tick 无价→升件+清仓径
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if custodyCount(t, ctx, w, "DUSTY") != 1 {
		t.Fatal("有价粉尘阈下须托管标")
	}
	if len(fake.transfers) != 0 || len(fake.purchases) != 0 {
		t.Fatalf("杂币 Earn 扫袋须零钱动: %v %v", fake.transfers, fake.purchases)
	}
	if opsCount(t, ctx, w, "杂币敞口", "DUSTY") != 0 || len(clr.places) > 1 {
		t.Fatal("有价粉尘不应升件不应清仓")
	}
	if opsCount(t, ctx, w, "杂币敞口", "DOGE") != 1 || len(clr.places) != 1 ||
		clr.places[0].InstID != "DOGE-USDT" {
		t.Fatal("无价敞口应升件+清仓径处置")
	}
	if opsCount(t, ctx, w, "杂币清仓", "DOGE") != 1 {
		t.Fatal("无价清仓拒=升件不补单")
	}
}

// 锚⑤：豁免册在册=双径跳过（清仓/停泊/升件全静默）。
func TestResidualExemptSkips(t *testing.T) {
	clr := &fakeClearer{}
	ctx, w, fake := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "XEMX-USDT", 0.5)
	if _, err := w.pool.Exec(ctx, `INSERT INTO fin_known_residual(venue,ccy,amount_usd,reason,ticket_ref,expires_at)
		VALUES($1,'XEMX',100,'venue 不可卖残存','T-0439',now()+interval '30 days')`, w.venue); err != nil {
		t.Fatal(err)
	}
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"XEMX": 500}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 0 || len(fake.transfers) != 0 || len(fake.purchases) != 0 {
		t.Fatal("豁免册在册被误动")
	}
	if opsCount(t, ctx, w, "杂币敞口", "XEMX") != 0 || opsCount(t, ctx, w, "杂币清仓", "XEMX") != 0 {
		t.Fatal("豁免册在册不应升件")
	}
}

// 锚⑥：清仓败（传输败→定真 absent）=「杂币清仓」升件行+不补单——复扫同键
// 节流零重发零连鸣。
func TestResidualClearFailEscalates(t *testing.T) {
	clr := &fakeClearer{err: errors.New("conn reset"),
		resolveRes: ClearResult{State: "absent", VenueCode: "51008", VenueMsg: "Insufficient balance"}}
	ctx, w, _ := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "CORE-USDT", 0.02)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"CORE": 2000}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 1 || clr.resolves != 1 {
		t.Fatalf("清仓应 Place+Resolve 各一次: places=%d resolves=%d", len(clr.places), clr.resolves)
	}
	if opsCount(t, ctx, w, "杂币清仓", "CORE") != 1 {
		t.Fatal("清仓败升件行未落")
	}
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 1 {
		t.Fatalf("已升件应同键节流不补单: places=%d", len(clr.places))
	}
	if opsCount(t, ctx, w, "杂币清仓", "CORE") != 1 || opsCount(t, ctx, w, "杂币敞口", "CORE") != 1 {
		t.Fatal("复扫连鸣——升件行应幂等")
	}
}

// 锚⑦：歧态势（传输败+查态亦败）=升件禁补单同键节流（双卖险）。
func TestResidualClearAmbiguousEscalates(t *testing.T) {
	clr := &fakeClearer{err: errors.New("timeout"), resolveErr: errors.New("state down")}
	ctx, w, _ := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "CORE-USDT", 0.02)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"CORE": 2000}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if opsCount(t, ctx, w, "杂币清仓", "CORE") != 1 {
		t.Fatal("歧态势升件行未落")
	}
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 1 {
		t.Fatalf("歧态势禁补单——复扫应零重发: places=%d", len(clr.places))
	}
}

// 锚⑧：venue 终拒（51001 标的不可交易）=不可卖残存入豁免册（30 天窗）——
// 复扫双径跳过零重发；册行携拒码审计锚。
func TestResidualTerminalExempts(t *testing.T) {
	clr := &fakeClearer{res: ClearResult{State: "absent", VenueCode: "51001", VenueMsg: "Instrument ID doesn't exist"}}
	ctx, w, _ := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "SHIBX-USDT", 0.001)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"SHIBX": 5000}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if exemptCount(t, ctx, w, "SHIBX") != 1 {
		t.Fatal("终拒不可卖须入豁免册")
	}
	var reason string
	if err := w.pool.QueryRow(ctx, `SELECT reason FROM fin_known_residual
		WHERE venue=$1 AND ccy='SHIBX'`, w.venue).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 1 {
		t.Fatalf("豁免册在册复扫应零清仓: places=%d", len(clr.places))
	}
	if opsCount(t, ctx, w, "杂币清仓", "SHIBX") != 0 {
		t.Fatal("终拒走豁免册非清仓升件")
	}
}

// 锚⑨：在持腿基币不误动（fund 集内——清仓/停泊/升件全静默）。
func TestResidualOpenLegNotSwept(t *testing.T) {
	clr := &fakeClearer{}
	ctx, w, fake := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "ETH-USDT", 3000)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"ETH": 0.032}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if len(clr.places) != 0 || len(fake.transfers) != 0 || len(fake.purchases) != 0 ||
		custodyCount(t, ctx, w, "ETH") != 0 || opsCount(t, ctx, w, "杂币敞口", "ETH") != 0 {
		t.Fatal("在持腿基币被误动（fund 集守恒破）")
	}
}

// 锚⑩：首检升件幂等——同币两拍敞口行恰 1 行（周期复扫不连鸣；清仓成交留
// 痕两拍各一——敞口再现=新处置非连鸣）。
func TestResidualExposureEscalateOnce(t *testing.T) {
	clr := &fakeClearer{res: ClearResult{State: "filled", FillPx: 0.02, FillSz: 2000}}
	ctx, w, _ := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	w.AttachClearer(clr)
	seedTick(t, ctx, w, "CORE-USDT", 0.02)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"CORE": 2000}, nil
	})
	for i := 0; i < 2; i++ {
		if err := w.sweepResidual(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if opsCount(t, ctx, w, "杂币敞口", "CORE") != 1 {
		t.Fatal("敞口升件行应幂等 1 行")
	}
}

// 锚⑪：清仓缝未挂=检出升件照发+清仓径让位（clearer_absent 计数门）——
// 敞口不因执行缝缺席而失声。
func TestResidualNoClearer(t *testing.T) {
	ctx, w, _ := setupW(t, &fakeAPI{}, residualKeys)
	residualSetup(t, ctx, w.pool, w.venue)
	seedTick(t, ctx, w, "CORE-USDT", 0.02)
	w.AttachCoins(func(context.Context) (map[string]float64, error) {
		return map[string]float64{"CORE": 2000}, nil
	})
	if err := w.sweepResidual(ctx); err != nil {
		t.Fatal(err)
	}
	if opsCount(t, ctx, w, "杂币敞口", "CORE") != 1 {
		t.Fatal("执行缝缺席检出升件仍应落")
	}
	if w.rej.Snapshot()["earn/clearer_absent"] == 0 {
		t.Fatal("clearer_absent 计数门未计")
	}
}
