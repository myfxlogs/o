package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"
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
	sid := mkStrategy(t, ctx, r, "live", time.Hour)
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
	sid := mkStrategy(t, ctx, r, "probe", 2*time.Hour)
	if _, err := r.pool.Exec(ctx,
		"INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,ts) VALUES($1,'funding','OP-USDT-SWAP',0.5,now())", sid); err != nil {
		t.Fatal(err)
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
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
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

// T-0103 ①锚：预测翻负即开闸拒——next_funding 与开向异号=寿命低于结算窗，
// 不开+fin_opp_eval rejected/lifetime_below_min 落卡（开平判据对偶化）。
func TestOpenLifetimeRejectsFlip(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "live", time.Hour)
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.0001,-0.0005)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("预测翻负仍开——判据分裂未修: %+v", fo.calls)
	}
	var reason, verdict string
	if err := r.pool.QueryRow(ctx, `SELECT verdict,evidence->>'reason' FROM fin_opp_eval
		WHERE inst_id='OP-USDT-SWAP' ORDER BY id DESC LIMIT 1`).Scan(&verdict, &reason); err != nil {
		t.Fatalf("rejected 评估卡未落: %v", err)
	}
	if verdict != "rejected" || reason != "lifetime_below_min" {
		t.Fatalf("拒因卡面: %s/%s", verdict, reason)
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
	for i := 0; i < 3; i++ {
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_position(strategy_ref,inst_id,notional_usd,opened_at,closed_at)
			VALUES($1,'OP-USDT-SWAP',100,now()-interval '30 min',now()-interval '20 min')`, sid); err != nil {
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
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate,next_funding_rate)
		VALUES(now(),'OP-USDT-SWAP',0.0005,0.0004)`); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 1 {
		t.Fatalf("健康仓被误拒: %+v", fo.calls)
	}
}
