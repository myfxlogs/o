package lifecycle

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"o/internal/platform/excell"
	"o/internal/research/funding"
)

type fakeOpener struct {
	calls []OpenSpec
	execN int64
	err   error
}

func (f *fakeOpener) Open(ctx context.Context, spec OpenSpec) (int64, error) {
	f.calls = append(f.calls, spec)
	if f.err != nil {
		return 0, f.err
	}
	f.execN++
	return f.execN, nil
}

func openerSetup(t *testing.T) (context.Context, *Runner, *fakeOpener) {
	t.Helper()
	ctx, pool, cfg := setup(t)
	for _, q := range []string{"DELETE FROM fin_position", "DELETE FROM fin_execution",
		"DELETE FROM fin_paper_fill", "DELETE FROM fin_strategy_transition",
		"DELETE FROM fin_strategy"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	fo := &fakeOpener{}
	return ctx, NewRunner(pool, cfg).AttachOpener(fo), fo
}

// seedAllCellsGreen T-0461 armed→open 格凭据闸测试凭据——excell 现查格集
// 全档 latest=pass 补种（strategy_ref=0 非策略行——凭据归属锚测试域）。
func seedAllCellsGreen(t *testing.T, ctx context.Context, r *Runner) {
	t.Helper()
	cells, err := excell.Enumerate(ctx, r.pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cells {
		if c.Exempt {
			continue
		}
		for _, k := range c.Keys() {
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill
				(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
				VALUES(0,$1,'pass','test','T-0461 测试格凭据',$1)`, k); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// seedCellKey 单格凭据补种（cell_key 显式——闸 fail-closed 专项锚用）。
func seedCellKey(t *testing.T, ctx context.Context, r *Runner, key, verdict string) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill
		(strategy_ref,inst_id,verdict,rules_version,detail,cell_key)
		VALUES(0,$1,$2,'test','T-0461 闸锚',$1)`, key, verdict); err != nil {
		t.Fatal(err)
	}
}

func mkStrategy(t *testing.T, ctx context.Context, r *Runner, state string, openedAgo time.Duration) int64 {
	t.Helper()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,venue,entered_at)
		VALUES('OP-USDT-SWAP','funding-hedge',$1,100,'short_perp_long_spot','okx',now()-$2::interval)
		RETURNING id`, state, openedAgo.String()).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// T-0050 锚①：live 无未平仓行→Opener 开仓；有行=幂等不再开（fin_position=天然幂等键）。
func TestOpenLiveFiresOnce(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	seedFunding(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "live", time.Hour)
	seedAllCellsGreen(t, ctx, r) // T-0461 格凭据闸——绿凭据补种后放行
	r.openLive(ctx)
	if len(fo.calls) != 1 || fo.calls[0].StrategyRef != sid || fo.calls[0].NotionalUSD != 100 ||
		fo.calls[0].InstID != "OP-USDT-SWAP" || fo.calls[0].Direction != "short_perp_long_spot" {
		t.Fatalf("live 开仓未过缝: %+v", fo.calls)
	}
	// 未平仓行落位（真实径=Execute 写 fin_position）→幂等不再开。
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES($1,'OP-USDT-SWAP',100,'open') RETURNING id`, sid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx,
		"INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd) VALUES($1,$2,'OP-USDT-SWAP',100)", eid, sid); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 1 {
		t.Fatalf("有未平仓行应幂等不再开: %d 次", len(fo.calls))
	}
	// 开仓败=计数不炸（事件续试语义）。
	if _, err := r.pool.Exec(ctx, "DELETE FROM fin_position"); err != nil {
		t.Fatal(err)
	}
	fo.err = errors.New("闸拒")
	r.openLive(ctx)
	if fo.calls[0].StrategyRef != sid && len(fo.calls) != 2 {
		t.Fatal("败径应续试记录")
	}
}

// T-0050 锚②：Opener 未接线=fail-closed 不开（零动作锚）。
func TestOpenLiveNilOpenerNoop(t *testing.T) {
	ctx, r, _ := openerSetup(t)
	r.op = nil
	mkStrategy(t, ctx, r, "live", time.Hour)
	r.openLive(ctx) // 无缝=零动作不炸
}

// T-0050 锚③：probe→live 晋升=观察窗+净回报续优+委员会开关（false=不晋）。
func TestProbeToLivePromotion(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	seedFunding(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "probe", 2*time.Hour)
	// T-0521：置信闸须≥3 收入笔（同额 sd=0→LB>0）。
	for i := 0; i < 3; i++ {
		if _, err := r.pool.Exec(ctx,
			"INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,ts) VALUES($1,'funding','OP-USDT-SWAP',0.5,now())", sid); err != nil {
			t.Fatal(err)
		}
	}
	// 开关 false（setup 缺省）→probe 不晋。
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&st); err != nil || st != "probe" {
		t.Fatalf("开关 false 应留 probe: %s err=%v", st, err)
	}
	// 开关 true（委员会翻位）→live 晋+过缝开仓。
	if _, err := r.pool.Exec(ctx,
		"UPDATE app_config SET value='\"true\"' WHERE key='invest.lifecycle.live_switch'"); err != nil {
		t.Fatal(err)
	}
	// T-0426 毕业闸：判据链新增「执行径演习通过」——绿凭据补种（缺/败
	// fail-closed，专项锚=execdrill_test）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,'OP-USDT-SWAP','pass','simvenue/1','T-0050 锚③测试凭据')`, sid); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	seedAllCellsGreen(t, ctx, r) // T-0461 格凭据闸——晋 live 后开仓须格绿
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&st); err != nil || st != "live" {
		t.Fatalf("开关 true+判据满应晋 live: %s err=%v", st, err)
	}
	if len(fo.calls) != 1 {
		t.Fatalf("晋 live 后应过缝开仓: %d", len(fo.calls))
	}
}

// seedFlip 判据键补种（T-0103：flip_bps=平闸同口径——setup 基表未含此键）。
func seedFlip(t *testing.T, ctx context.Context, r *Runner) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value)
		VALUES('guard.flip_bps',to_jsonb('5'::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
}

// seedFunding 健康 funding 行补种（判据过径——正费率+正预测不翻面；0.0006
// →carry=5.4×9=48.6bps>4×(5+3)×1.5=48 回本窗闸过——T-0111 尾注夹具提量）。
func seedFunding(t *testing.T, ctx context.Context, r *Runner) {
	t.Helper()
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.0006,0.0004)`); err != nil {
		t.Fatal(err)
	}
}

// T-0103 ①锚：预测翻负即开闸拒——next_funding 与开向异号=寿命低于结算窗，
// 不开+fin_opp_eval rejected/lifetime_below_min 落卡（开平判据对偶化）。
func TestOpenLifetimeRejectsFlip(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "live", time.Hour)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.0001,-0.0006)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("预测翻负仍开——判据分裂未修: %+v", fo.calls)
	}
	// T-0206①：拆表后拒因行在 fin_eval_reject（非 fin_opp_eval——
	// 夹具迁移债根治；verdict 显式列无，reason=显式列直查）。
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_eval_reject
		WHERE inst_id='OP-USDT-SWAP' ORDER BY id DESC LIMIT 1`).Scan(&reason); err != nil {
		t.Fatalf("rejected 评估卡未落: %v", err)
	}
	if reason != "lifetime_below_min" {
		t.Fatalf("拒因卡面: %s", reason)
	}
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "live" {
		t.Fatalf("寿命拒=不开非降级: %s", st)
	}
}

// T-0103 ③锚：同键 1h ≥3 轮开平=振荡保险丝——不开+降级（兜底判据再分裂）。
func TestOpenOscillationFuse(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "live", time.Hour)
	var eid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,inst_id,notional_usd,state)
		VALUES($1,'OP-USDT-SWAP',100,'closed') RETURNING id`, sid).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at,closed_at)
			VALUES($2,$1,'OP-USDT-SWAP',100,now()-interval '30 min',now()-interval '20 min')`, sid, eid); err != nil {
			t.Fatal(err)
		}
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("振荡窗仍喂开: %+v", fo.calls)
	}
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE id=$1`, sid).Scan(&st); err != nil || st != "degraded" {
		t.Fatalf("振荡保险丝未降级: %s", st)
	}
}

// 正例锚：funding 正+预测正=正常开（判据对偶不误杀好仓）。
func TestOpenLifetimePassesHealthy(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	mkStrategy(t, ctx, r, "live", time.Hour)
	// 0.0006→carry 48.6bps>48 回本窗闸过（T-0111 尾注——0.0005 夹具落闸下）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.0006,0.0004)`); err != nil {
		t.Fatal(err)
	}
	seedAllCellsGreen(t, ctx, r) // T-0461 格凭据闸
	r.openLive(ctx)
	if len(fo.calls) != 1 {
		t.Fatalf("健康仓被误拒: %+v", fo.calls)
	}
}

// T-0461 锚：armed→open 格凭据闸 fail-closed——无 cell_key 凭据=hold
// 不开仓+exec_cell_hold 计数；fail 凭据=hold；pass 凭据=放行落缝。
func TestExecCellHoldGate(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	seedFunding(t, ctx, r)
	mkStrategy(t, ctx, r, "live", time.Hour)
	// ① 无凭据=hold（fail-closed——矩阵未跑不放行）。
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("无格凭据仍开仓: %+v", fo.calls)
	}
	if n := int(r.rej.Snapshot()["lifecycle/exec_cell_hold"]); n != 1 {
		t.Fatalf("exec_cell_hold 计数=%d want 1", n)
	}
	var st string
	if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_strategy WHERE inst_id='OP-USDT-SWAP'`).Scan(&st); err != nil || st != "live" {
		t.Fatalf("hold 应留 live 不开仓非降级: %s", st)
	}
	// ② fail 凭据=hold 续（latest=fail 不放行）。
	cells, err := excell.Enumerate(ctx, r.pool)
	if err != nil {
		t.Fatal(err)
	}
	var cellKey string
	for _, c := range cells {
		if excell.CoordKey(c.Template, c.VenuePair, c.Direction) ==
			excell.CoordKey("funding-hedge", "okx", "short_perp_long_spot") && !c.Exempt {
			cellKey = c.Keys()[0]
		}
	}
	if cellKey == "" {
		t.Fatal("funding-hedge/short_perp_long_spot 格未枚举到")
	}
	seedCellKey(t, ctx, r, cellKey, "fail")
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("fail 凭据仍开仓: %+v", fo.calls)
	}
	// ③ pass 凭据=放行（latest=pass 翻绿自解）。
	seedCellKey(t, ctx, r, cellKey, "pass")
	r.openLive(ctx)
	if len(fo.calls) != 1 {
		t.Fatalf("pass 凭据未放行: %+v", fo.calls)
	}
}

// T-0111 ①锚（T-0297 同源化）：回本窗判据——毛收年化 0.05475 <
// 摩擦年化 cost_apr(32bps×365/3=0.3893)×1.5 → breakeven_beyond_hold 拒开；
// 拒卡 evidence.cost_apr 与 funding.CostAPR 同值（展示/判据同源对拍）。
func TestOpenBreakevenRejectsThin(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	mkStrategy(t, ctx, r, "live", time.Hour)
	// ETH 级薄 funding（0.5bps=0.00005——回本窗远超 72h 持仓上限）。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.00005,0.00005)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatal("回本窗不满仓被开出")
	}
	var costAPR float64
	if err := r.pool.QueryRow(ctx, `SELECT (evidence->>'cost_apr')::float8 FROM fin_eval_reject
		WHERE reason='breakeven_beyond_hold'`).Scan(&costAPR); err != nil {
		t.Fatalf("breakeven_beyond_hold 拒卡未落: %v", err)
	}
	want := funding.CostAPR(funding.Params{TakerFeeBps: 5, SlippageBps: 3, FailCostBps: 2, WindowLossBps: 4, HoldDays: 3})
	if costAPR != want {
		t.Fatalf("evidence cost_apr=%v 与 funding.CostAPR=%v 异源", costAPR, want)
	}
}

// T-0521 读侧合流锚：实测分量（fee=4/slip=10 往返 bps）新鲜→fee/slip 项
// 替声明参数（fail/window 续守）→cost=4+10+2+4=20bps→cost_apr=0.2433；
// friction_at 陈>30d→声明参数续守（32bps→0.3893）。
func TestBreakevenMeasuredOverride(t *testing.T) {
	ctx, r, _ := openerSetup(t)
	seedFlip(t, ctx, r)
	ins := func(k, v string) {
		if _, err := r.pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	ins("lifecycle.friction_fee_bps.okx", "4")
	ins("lifecycle.friction_slip_bps.okx", "10")
	// 薄 funding→仍拒,但拒卡 cost_apr 应取实测口径非声明口径。
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.00005,0.00005)`); err != nil {
		t.Fatal(err)
	}
	readCost := func() float64 {
		var v float64
		if err := r.pool.QueryRow(ctx, `SELECT (evidence->>'cost_apr')::float8 FROM fin_eval_reject
			WHERE reason='breakeven_beyond_hold' ORDER BY id DESC LIMIT 1`).Scan(&v); err != nil {
			t.Fatalf("breakeven 拒卡未落: %v", err)
		}
		return v
	}
	// ①无 at 戳（分量不齐）→声明口径 32bps。每轮退役旧行+清拒卡+
	// 建新行——denyAt 冷窗(300s)+active 唯一索引+reject 1h 幂等窗
	// 三件套同窗内同 inst 不再出新拒卡（先行退役+清卡再建）。
	mkStrategy(t, ctx, r, "live", time.Hour)
	r.openLive(ctx)
	if got, want := readCost(), funding.CostAPR(funding.Params{TakerFeeBps: 5, SlippageBps: 3,
		FailCostBps: 2, WindowLossBps: 4, HoldDays: 3}); got != want {
		t.Fatalf("分量不齐应续守声明口径: got=%v want=%v", got, want)
	}
	// ②at 新鲜→实测口径 20bps。
	ins("lifecycle.friction_at.okx", strconv.FormatInt(time.Now().Unix(), 10))
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_strategy SET state='retired' WHERE state='live';
		DELETE FROM fin_eval_reject`); err != nil {
		t.Fatal(err)
	}
	mkStrategy(t, ctx, r, "live", time.Hour)
	r.openLive(ctx)
	if got, want := readCost(), funding.CostAPR(funding.Params{TakerFeeBps: 1, SlippageBps: 5,
		FailCostBps: 2, WindowLossBps: 4, HoldDays: 3}); got != want {
		t.Fatalf("实测新鲜应替声明: got=%v want=%v", got, want)
	}
	// ③at 陈 40d→声明口径复归。
	ins("lifecycle.friction_at.okx", strconv.FormatInt(time.Now().Add(-40*24*time.Hour).Unix(), 10))
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_strategy SET state='retired' WHERE state='live';
		DELETE FROM fin_eval_reject`); err != nil {
		t.Fatal(err)
	}
	mkStrategy(t, ctx, r, "live", time.Hour)
	r.openLive(ctx)
	if got, want := readCost(), funding.CostAPR(funding.Params{TakerFeeBps: 5, SlippageBps: 3,
		FailCostBps: 2, WindowLossBps: 4, HoldDays: 3}); got != want {
		t.Fatalf("实测陈旧应复归声明口径: got=%v want=%v", got, want)
	}
}
