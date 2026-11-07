package simdrill

import (
	"context"
	"math"
	"strings"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/risk"
	"o/internal/store/pgstore"
	"o/internal/trading/simvenue"
)

// fixture CORE 案（T-0419 仓#33 形——数值=fixture 非产线常量）：
// quote 买 25 USDT@0.0125 费 0.001→实得 1998 CORE；perp 20 张×ctVal 100×
// px 0.0125=$25 均衡；策略名义 50。
func coreVenue() *simvenue.SimVenue {
	return simvenue.New().
		WithInstrument(simvenue.Instrument{InstID: "CORE-USDT", InstType: "SPOT",
			LotSz: 0.01, MinSz: 0.01, TickSz: 0.0001, TakerFee: 0.001, Px: 0.0125}).
		WithInstrument(simvenue.Instrument{InstID: "CORE-USDT-SWAP", InstType: "SWAP",
			LotSz: 0.01, MinSz: 0.01, TickSz: 0.0001, CtVal: 100, CtValCcy: "CORE",
			TakerFee: 0.0005, Px: 0.0125}).
		WithBalance("USDT", 1000)
}

func setup(t *testing.T) (context.Context, *Drill) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 测试面隔离：只清本包自有面（template_ref 打标级联——同 guard-test 律）。
	for _, stmt := range []string{
		`DELETE FROM fin_guard_close WHERE position_id IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate')))`,
		`DELETE FROM fin_cashflow WHERE position_ref IN
		   (SELECT id FROM fin_position WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate')))`,
		`DELETE FROM fin_position WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate'))`,
		`DELETE FROM fin_order_leg WHERE execution_id IN
		   (SELECT id FROM fin_execution WHERE strategy_ref IN
		     (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate')))`,
		`DELETE FROM fin_execution WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate'))`,
		`DELETE FROM fin_exec_drill WHERE strategy_ref IN
		   (SELECT id FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate'))`,
		"DELETE FROM fin_strategy WHERE template_ref IN ('simdrill','simdrill-gate')",
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
		"risk.kill_switch":       "false", "risk.max_notional_usd": "500",
		"risk.max_venue_pct": "0.35", "risk.max_exposure_usd": "1000",
		"risk.token_ttl_seconds": "30",
		"risk.max_dd_30d":        "0.03", "risk.dd_window_days": "30"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 行情真相锚（venue 成交价与 md_ticks 同价——防时差型假 drift；产线读面）。
	for _, inst := range []string{"CORE-USDT", "CORE-USDT-SWAP"} {
		if _, err := pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'sim',$1,0.0125)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`, inst); err != nil {
			t.Fatal(err)
		}
	}
	// 被闸策略 fixture（probe 态——毕业闸域；同型同名义演习对型跑）。
	var stratID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('CORE-USDT-SWAP','simdrill-gate','probe',50,'okx') RETURNING id`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	gate := risk.NewGate(pool, cfg).
		WithEquity(func(context.Context, string) (float64, error) { return 1000, nil })
	return ctx, &Drill{Pool: pool, Cfg: cfg, Gate: gate, Venue: coreVenue()}
}

// 红桩主锚（T-0426×T-0419 首案重放）：CORE quote-unit 腿全径演习——开仓
// qty_base=venue 实得（1998 非美元额 25）、平仓腿量=腿实得币量（修前裸拷
// 美元额当 base 发=错平病灶——修后码上绿）、venue 残余清零、凭据落账。
func TestDrillQuoteUnitFullPath(t *testing.T) {
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
	if res.Trigger != "strategy_degraded" {
		t.Fatalf("触发应降级联动（drift 假报禁线）: %q", res.Trigger)
	}
	// 开腿账=venue 实得（flow.legPx×sim fill 对拍——qty_base 唯一真源）。
	var szUnit string
	var qtyBase float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT sz_unit,qty_base FROM fin_order_leg WHERE execution_id=$1 AND inst_id='CORE-USDT'`,
		res.OpenExecID).Scan(&szUnit, &qtyBase); err != nil {
		t.Fatal(err)
	}
	if szUnit != "quote" || math.Abs(qtyBase-1998) > 1e-6 {
		t.Fatalf("开腿量纲: sz_unit=%s qty_base=%.6f（want quote/1998）", szUnit, qtyBase)
	}
	// 红桩主断言：平仓腿量=腿实得币量 1998（修前=美元额 25——错平病灶）。
	var closeSz float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT sz FROM fin_order_leg WHERE execution_id=$1 AND inst_id='CORE-USDT'`,
		res.CloseExecID).Scan(&closeSz); err != nil {
		t.Fatal(err)
	}
	if math.Abs(closeSz-1998) > 1e-6 {
		t.Fatalf("平仓腿应=腿实得 1998 非美元额 25（T-0419 病灶重放）: %.6f", closeSz)
	}
	// venue 真相：现货残余清零+合约净持归零（账实同步）。
	if r := d.Venue.Balance("CORE"); r > 0.01 {
		t.Fatalf("venue CORE 残余应清零: %.8f", r)
	}
	if p := d.Venue.Position("CORE-USDT-SWAP"); math.Abs(p) > 1e-9 {
		t.Fatalf("venue 净持应归零: %v", p)
	}
	// 仓行闭环+净袋（干净平仓无挂账）。
	var closed bool
	var bag *float64
	var settled *float64
	if err := d.Pool.QueryRow(ctx,
		`SELECT closed_at IS NOT NULL,bare_bag_qty,settled_amount FROM fin_position
		WHERE execution_id=$1 ORDER BY id DESC LIMIT 1`, res.OpenExecID).
		Scan(&closed, &bag, &settled); err != nil {
		t.Fatal(err)
	}
	if !closed || bag != nil {
		t.Fatalf("仓应 closed 且无挂账: closed=%v bag=%v", closed, bag)
	}
	if settled == nil || math.Abs(*settled-(-0.049975)) > 0.001 {
		t.Fatalf("结算落位（现金净+无挂账）: %v", settled)
	}
	// 凭据落账（毕业闸消费面——verdict+指纹）。
	var verdict, rv string
	if err := d.Pool.QueryRow(ctx,
		`SELECT verdict,rules_version FROM fin_exec_drill WHERE strategy_ref=$1 ORDER BY id DESC LIMIT 1`,
		stratID).Scan(&verdict, &rv); err != nil {
		t.Fatal(err)
	}
	if verdict != "pass" || rv != simvenue.RulesVersion {
		t.Fatalf("凭据: verdict=%s rules=%s", verdict, rv)
	}
}

// 异常升件径（禁静默红）：venue 残余>挂账=「closed 了但货还在」盲区重放
// ——锚②破→ops_event 升件行+verdict fail（毕业闸 fail-closed 消费）。
func TestDrillResidueEscalation(t *testing.T) {
	ctx, d := setup(t)
	d.Venue.WithBalance("CORE", 500) // 账外预置残袋（台账不知情——Earn 收息形）
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='simdrill-gate' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "fail" || !strings.Contains(res.Detail, "残袋非零") {
		t.Fatalf("残袋盲应 fail+升件: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	var n int
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class='执行径演习' AND ticket='T-0426'
		AND note LIKE '演习 %残袋非零%'`).Scan(&n); err != nil || n == 0 {
		t.Fatalf("升件行缺（禁静默红）: n=%d err=%v", n, err)
	}
	if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM fin_exec_drill
		WHERE strategy_ref=$1 AND verdict='fail'`, stratID).Scan(&n); err != nil || n == 0 {
		t.Fatalf("fail 凭据缺: n=%d err=%v", n, err)
	}
}

// 平仓径拒单=演习未平（fail 禁假绿）+拒单一等事件入 leg 行（venue_err_code
// 禁吞——T-0040 径）+partials 态机经产线缝消费（ack 行）。
func TestDrillCloseRejectedEscalation(t *testing.T) {
	ctx, d := setup(t)
	// SWAP 平仓腿注入部分成交计划（无补成拍=冻结部分态——placeAndResolve
	// 消费 partially_filled→ack+未即时成交错→guard close_failed）。
	d.Venue.WithPlan("CORE-USDT-SWAP", simvenue.Plan{FillRatio: 0.5, OnlyClose: true})
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='simdrill-gate' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	res, err := d.Run(ctx, stratID)
	if err != nil {
		t.Fatalf("演习机械败: %v", err)
	}
	if res.Verdict != "fail" || !strings.Contains(res.Detail, "演习未平") {
		t.Fatalf("未平仓应 fail 禁假绿: verdict=%s detail=%s", res.Verdict, res.Detail)
	}
	// 拒单/部分态一等事件：close 腿行 ack 态留痕（venue 态机×产线消费缝）。
	var legState string
	if err := d.Pool.QueryRow(ctx, `SELECT state FROM fin_order_leg
		WHERE execution_id=$1 AND inst_id='CORE-USDT-SWAP'`, res.CloseExecID).
		Scan(&legState); err != nil || legState != "ack" {
		t.Fatalf("close 腿应 ack 态留痕: %q err=%v", legState, err)
	}
	if res.Trigger != "strategy_degraded" {
		t.Fatalf("触发应降级联动: %q", res.Trigger)
	}
}

// 开仓部分成交（venue 态机 partial→production 消费）：flow.go 消费
// partially_filled=未即时成交不造仓——leg ack 行+exec failed+演习机械败上抛。
func TestDrillPartialFillOpenAck(t *testing.T) {
	ctx, d := setup(t)
	d.Venue.WithPlan("CORE-USDT", simvenue.Plan{FillRatio: 0.5})
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='simdrill-gate' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	_, err := d.Run(ctx, stratID)
	if err == nil || !strings.Contains(err.Error(), "开仓执行败") {
		t.Fatalf("部分成交开仓应机械败上抛: %v", err)
	}
	// 腿 ack 行留痕（T-0107 live 确认轨语义——venue 态机经产线缝可重放）。
	var legState, execState string
	if err := d.Pool.QueryRow(ctx, `SELECT l.state,e.state::text FROM fin_order_leg l
		JOIN fin_execution e ON e.id=l.execution_id
		WHERE e.strategy_ref=(SELECT id FROM fin_strategy WHERE template_ref='simdrill' LIMIT 1)
		AND l.inst_id='CORE-USDT' ORDER BY l.id DESC LIMIT 1`).
		Scan(&legState, &execState); err != nil || legState != "ack" || execState != "failed" {
		t.Fatalf("partial 开仓应 leg=ack/exec=failed: %s/%s err=%v", legState, execState, err)
	}
}

// 拒单槽序×演习径：开仓腿违网格=venue 拒（51005）→leg failed 行携拒码
// ——「订单会不会出错」在纸面就能找到（业主令）。
func TestDrillOpenRejectedLegReceipt(t *testing.T) {
	ctx, d := setup(t)
	// 破坏网格：现货腿 lotSz 改 3——quote 买 25 额≈2000 base 等当量/3=不落网格
	//（合约量自 lotSz 派生恒在网格——破网格只能破现货腿）。
	inst, _ := d.Venue.Instrument("CORE-USDT")
	inst.LotSz = 3
	d.Venue.WithInstrument(inst)
	var stratID int64
	if err := d.Pool.QueryRow(ctx,
		`SELECT id FROM fin_strategy WHERE template_ref='simdrill-gate' LIMIT 1`).Scan(&stratID); err != nil {
		t.Fatal(err)
	}
	_, err := d.Run(ctx, stratID)
	if err == nil || !strings.Contains(err.Error(), "开仓执行败") {
		t.Fatalf("违网格开腿应拒: %v", err)
	}
	// 拒码入 leg 行（venue_err_code 禁吞——回执一等事件）。
	var code string
	if err := d.Pool.QueryRow(ctx, `SELECT l.venue_err_code FROM fin_order_leg l
		JOIN fin_execution e ON e.id=l.execution_id
		WHERE e.strategy_ref=(SELECT id FROM fin_strategy WHERE template_ref='simdrill' LIMIT 1)
		AND l.inst_id='CORE-USDT' ORDER BY l.id DESC LIMIT 1`).
		Scan(&code); err != nil || code != simvenue.RejOffGrid {
		t.Fatalf("拒码应入 leg 行 51005: %q err=%v", code, err)
	}
}
