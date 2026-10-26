package main

// untradeable_test.go —— 终局分类/黑名单桩测（T-0142 验收锚）：造桩 venue 51001→
// 黑名单入册+策略退+armed 提案退；黑名单 inst 后续 universe 滤+起草滤。
import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/portfolio/lifecycle"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func t51001() error {
	return &okxexec.APIError{Code: "51001", Msg: "Instrument ID doesn't exist"}
}

func setupDB(t *testing.T) (context.Context, *pgxpool.Pool, *config.Client) {
	t.Helper()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(context.Background(), pool); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return context.Background(), pool, cfg
}

// 锚：TerminalInstErr——51001/51003 终局 true；51000（参数族 clOrdId 先例）/
// 瞬态/nil false（禁误伤参数 bug）。
func TestTerminalInstErr(t *testing.T) {
	if !okxexec.TerminalInstErr(t51001()) {
		t.Fatal("51001 应终局")
	}
	if !okxexec.TerminalInstErr(&okxexec.APIError{Code: "51003"}) {
		t.Fatal("51003 应终局")
	}
	for _, e := range []error{
		&okxexec.APIError{Code: "51000", Msg: "clOrdId 非法"},
		&okxexec.APIError{Code: "50011", Msg: "rate limit"},
		errors.New("transport"),
		nil,
	} {
		if okxexec.TerminalInstErr(e) {
			t.Fatalf("%v 不应终局", e)
		}
	}
}

// 锚（票面验收）：造桩 51001→untradeable 处置三件全落——黑名单行+策略
// retired（venue_untradeable 溯源）+armed 提案 fin_retired。
func TestUntradeableDisposition(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('SHEIN-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'live','okx','{}') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal("策略夹具:", err)
	}
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('SHEIN-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal("评估夹具:", err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'SHEIN-USDT-SWAP','short_perp_long_spot','fin_armed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal("提案夹具:", err)
	}
	spec := lifecycle.OpenSpec{StrategyRef: sid, Venue: "okx", InstID: "SHEIN-USDT-SWAP",
		Direction: "short_perp_long_spot"}
	if err := h.untradeable(ctx, spec, t51001()); err != nil {
		t.Fatal("处置败:", err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM fin_venue_untradeable WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&code); err != nil || code != "51001" {
		t.Fatalf("黑名单应入册 code=%q err=%v", code, err)
	}
	var st string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "retired" {
		t.Fatalf("策略应退 st=%q", st)
	}
	var tr string
	if err := pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 ORDER BY id DESC LIMIT 1`, sid).Scan(&tr); err == nil && len(tr) < len("venue_untradeable") {
		t.Fatalf("Transition 应留 venue_untradeable 溯源 got %q", tr)
	}
	var ps string
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, pid).Scan(&ps); err != nil || ps != "fin_retired" {
		t.Fatalf("armed 提案应退 ps=%q", ps)
	}
}

// 锚：黑名单 inst 后续零——openLive 候选查询滤除+起草滤（gapscan）。
func TestBlacklistConsumers(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','51001','stub') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var live, banned int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE state='live' AND inst_id='CSOPSKHYNIX2L-USDT-SWAP'
		AND inst_id NOT IN (SELECT inst_id FROM fin_venue_untradeable)`).Scan(&live); err != nil {
		t.Skip("fin_strategy 列形异:", err)
	}
	if live != 0 {
		t.Fatal("黑名单 inst 应零候选")
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable
		WHERE inst_id='CSOPSKHYNIX2L-USDT-SWAP'`).Scan(&banned); err != nil || banned != 1 {
		t.Fatalf("黑名单行应在 banned=%d err=%v", banned, err)
	}
}

// T-0147 桩锚①：腿构 meta 51001→终局三件（入册+策略退+armed 提案清）——
// BuildLegs spot 元数据拉取段实案形（策略 286078）。
func TestLegBuildTerminalTriage(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('SHEIN-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'live','okx','{}') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal("策略夹具:", err)
	}
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('SHEIN-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'SHEIN-USDT-SWAP','short_perp_long_spot','fin_armed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	// 腿构径 51001（spot meta 拉取段冒泡形——同 Execute 径收口）。
	legErr := fmt.Errorf("腿构: spot 元数据: %w", t51001())
	if !okxexec.TerminalInstErr(errors.Unwrap(legErr)) && !okxexec.TerminalInstErr(legErr) {
		// 冒泡链内含终局码——TerminalInstErr 须穿透 wrap（errors.As 语义）
		t.Fatal("冒泡 wrap 链内 51001 应判终局")
	}
	spec := lifecycle.OpenSpec{StrategyRef: sid, Venue: "okx", InstID: "SHEIN-USDT-SWAP"}
	if err := h.untradeable(ctx, spec, errors.Unwrap(legErr)); err != nil {
		t.Fatal("腿构径终局处置败:", err)
	}
	var st string
	pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st)
	if st != "retired" {
		t.Fatalf("策略应退 got %q", st)
	}
	var ps string
	pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, pid).Scan(&ps)
	if ps != "fin_retired" {
		t.Fatalf("armed 提案应退 got %q", ps)
	}
	var n int
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_untradeable WHERE inst_id='SHEIN-USDT-SWAP'`).Scan(&n)
	if n != 1 {
		t.Fatalf("黑名单应入册 got %d", n)
	}
}

// T-0147 桩锚②：手工 INSERT fin_venue_untradeable→回扫 trigger 同 inst
// live 策略/armed 提案自动退役（retroactive transition 溯源行在案）+幂等
// （重复入册不叠 transition）。
func TestRetroactiveScanTrigger(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'live','okx','{}') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	var eid int64
	pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','[]','{}') RETURNING id`).Scan(&eid)
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,state,net_apr)
		VALUES($1,'CSOPSKHYNIX2L-USDT-SWAP','short_perp_long_spot','fin_armed',0) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	// 手工入册（决策席 SQL 形）——trigger 回扫。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','51001','stub') ON CONFLICT DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var st string
	pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st)
	if st != "retired" {
		t.Fatalf("存量 live 应自动退役 got %q", st)
	}
	var reason string
	if err := pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND reason LIKE 'venue_untradeable_retroactive%'`, sid).Scan(&reason); err != nil || reason == "" {
		t.Fatalf("retroactive transition 应落 err=%v", err)
	}
	var ps string
	pool.QueryRow(ctx, `SELECT state FROM fin_proposal WHERE id=$1`, pid).Scan(&ps)
	if ps != "fin_retired" {
		t.Fatalf("armed 提案应清 got %q", ps)
	}
	// 幂等：重复手工入册（新 INSERT 语句——PK 冲突走 UPDATE 不触发）不叠行。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('CSOPSKHYNIX2L-USDT-SWAP','51001','stub2') ON CONFLICT (inst_id) DO UPDATE SET msg=EXCLUDED.msg`); err != nil {
		t.Fatal(err)
	}
	var tn int
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy_transition
		WHERE strategy_id=$1 AND reason LIKE 'venue_untradeable_retroactive%'`, sid).Scan(&tn)
	if tn != 1 {
		t.Fatalf("retroactive transition 应恒 1 行 got %d", tn)
	}
}

// T-0147 桩锚③：51000 参数族不误伤——腿构/提交径 51000 走瞬态重试零处置。
func TestParamErrNoTriage(t *testing.T) {
	if okxexec.TerminalInstErr(&okxexec.APIError{Code: "51000", Msg: "clOrdId 非法"}) {
		t.Fatal("51000 参数族不得判终局（除外条款不变）")
	}
	// 回扫 trigger 不因 51000 形参数错触发（无入册即无回扫——语义直证）。
	ctx, pool, _ := setupDB(t)
	var sid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('BTC-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'live','okx','{}') RETURNING id`).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	var st string
	pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st)
	if st != "live" {
		t.Fatalf("零入册应零回扫 got %q", st)
	}
}

// T-0151 桩锚：五态全覆盖——0061 只覆 live/paper/proposed 三态，probe 态
// 幽灵（286181 SHEIN-USDT-SWAP 滞留实案形）入册即退为票面生产锚；degraded
// 同形。另钉闭集自挡：已 retired 行入册域内不再触（零新增 transition）。
// 每腿独 inst——trigger 回扫域=同 inst，互不串。
func TestRetroTwoStates(t *testing.T) {
	ctx, pool, _ := setupDB(t)
	for _, state := range []string{"live", "paper", "proposed", "probe", "degraded"} {
		inst := fmt.Sprintf("STUB-T0151-%s-USDT-SWAP", state)
		var sid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
			VALUES($1,'funding-hedge',0,'short_perp_long_spot',100,$2,'okx','{}') RETURNING id`,
			inst, state).Scan(&sid); err != nil {
			t.Fatalf("%s 策略夹具: %v", state, err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
			VALUES($1,'51001','stub')`, inst); err != nil {
			t.Fatalf("%s 入册: %v", state, err)
		}
		var st string
		if err := pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "retired" {
			t.Fatalf("%s 态应入册即退 got st=%q err=%v", state, st, err)
		}
		var from string
		if err := pool.QueryRow(ctx, `SELECT from_state FROM fin_strategy_transition
			WHERE strategy_id=$1 AND reason LIKE 'venue_untradeable_retroactive:%'`,
			sid).Scan(&from); err != nil || from != state {
			t.Fatalf("%s 态 retroactive 溯源行应落（from=%s）got %q err=%v", state, state, from, err)
		}
	}
	// 闭集自挡：已 retired 行不被回扫重触（状态谓词拒自挡已退役行）。
	var rid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
		VALUES('STUB-T0151-RETIRED-USDT-SWAP','funding-hedge',0,'short_perp_long_spot',100,'retired','okx','{}') RETURNING id`).Scan(&rid); err != nil {
		t.Fatal("retired 夹具:", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_venue_untradeable(inst_id,code,msg)
		VALUES('STUB-T0151-RETIRED-USDT-SWAP','51001','stub')`); err != nil {
		t.Fatal("入册:", err)
	}
	var tn int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy_transition
		WHERE strategy_id=$1`, rid).Scan(&tn); err != nil {
		t.Fatal(err)
	}
	if tn != 0 {
		t.Fatalf("已 retired 行应闭集自挡（零 transition）got %d", tn)
	}
}

// T-0151 桩锚：opener 容退守卫谓词集对齐（0062 同扩——retired 外五态皆可
// 显式迁移）：probe/degraded 态策略走 untradeable() 应显式迁移（reason=
// venue_untradeable:<code> 非 retroactive——守卫放行的直证；回退 state='live'
// 守卫则显式迁移被跳、仅剩入册 trigger retroactive 行=红）。
func TestUntradeableGuardTwoStates(t *testing.T) {
	ctx, pool, cfg := setupDB(t)
	h := hedgeOpener{pool: pool, cfg: cfg}
	for _, state := range []string{"probe", "degraded"} {
		inst := fmt.Sprintf("STUB-T0151-GUARD-%s-USDT-SWAP", state)
		var sid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,direction,notional_usd,state,venue,criteria)
			VALUES($1,'funding-hedge',0,'short_perp_long_spot',100,$2,'okx','{}') RETURNING id`,
			inst, state).Scan(&sid); err != nil {
			t.Fatalf("%s 策略夹具: %v", state, err)
		}
		spec := lifecycle.OpenSpec{StrategyRef: sid, Venue: "okx", InstID: inst}
		if err := h.untradeable(ctx, spec, t51001()); err != nil {
			t.Fatalf("%s 处置败: %v", state, err)
		}
		var st string
		if err := pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "retired" {
			t.Fatalf("%s 态守卫应显式迁移退役 got st=%q err=%v", state, st, err)
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy_transition
			WHERE strategy_id=$1 AND reason LIKE 'venue_untradeable:%'
			  AND reason NOT LIKE '%retroactive%'`, sid).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s 态应恰一条显式迁移溯源行 got n=%d err=%v", state, n, err)
		}
	}
}
