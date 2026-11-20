package lifecycle

// shapegate_test.go —— 工单#26436 判例#1本金线形态闸三拉起面契约测：
// intake(admit)/promote(probe→live)/openLive 同消费 execfam.ShapeGateVerdict
// 唯一判词源（钱径闸面双源防散——同 execdrill 全谱判例）。bounded 单标族
// NAV 懒读零触库（shapeGateNavLazy 包封——本金线判据只对杠杆/代管族求值）。

import (
	"testing"
	"time"
)

// intake 面：本金线判据缺（无 NAV 行）→leveraged 族拦留 fin_proposed
// 不建策略；bounded 单标族直过建 paper（不读 NAV——懒读断言=判据缺仍放行）；
// NAV 回补=判据面自解→leveraged 放行。判据缺非越线——升件面零行。
func TestShapeGateAdmitIntake(t *testing.T) {
	ctx, pool, cfg := setup(t)
	if _, err := pool.Exec(ctx, "DELETE FROM md_equity_curve WHERE ccy='NAV'"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	mk := func(inst, tmpl, dir string) int64 {
		var eid, pid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
			VALUES($1,$2,$3,0.08,'proposed','{}','{}') RETURNING id`, inst, tmpl, dir).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
			VALUES($1,$2,$3,1.0) RETURNING id`, eid, inst, dir).Scan(&pid); err != nil {
			t.Fatal(err)
		}
		return pid
	}
	pidLev := mk("SGL-USDT-SWAP", "funding-hedge", "short_perp_long_spot")
	mk("SGB-USDT", "grid_spot", "grid_spot")
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=$1", pidLev).Scan(&st); err != nil || st != "fin_proposed" {
		t.Fatalf("leveraged 判据缺应留 fin_proposed: state=%s err=%v", st, err)
	}
	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_strategy WHERE inst_id='SGB-USDT'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("bounded 直过应建 paper 策略（不读 NAV）: n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='shape_gate_principal'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("判据缺非越线不应升件: n=%d err=%v", n, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_equity_curve(ccy,equity,src) VALUES('NAV',100000,'test')`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=$1", pidLev).Scan(&st); err != nil || st == "fin_proposed" {
		t.Fatalf("NAV 回补 leveraged 应放行: state=%s err=%v", st, err)
	}
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_strategy WHERE inst_id='SGL-USDT-SWAP'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("leveraged 放行应建策略: n=%d err=%v", n, err)
	}
}

// 越线升件面：NAV 不足盖 demand(notional/lev+notional×lossFrac)→拦+落
// ops_event 升件行（ownerEsc=true 越本金线归业主面），proposal 留 fin_proposed。
func TestShapeGateAdmitEscalate(t *testing.T) {
	ctx, pool, cfg := setup(t)
	// NAV 只盖 1.5×notional——demand=2×notional 越线。
	if _, err := pool.Exec(ctx, "UPDATE md_equity_curve SET equity=150 WHERE ccy='NAV'"); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('SGE-USDT-SWAP','funding-hedge','short_perp_long_spot',0.08,'proposed','{}','{}') RETURNING id`).Scan(new(int64)); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		SELECT id,'SGE-USDT-SWAP','short_perp_long_spot',1.0 FROM fin_opp_eval
		WHERE inst_id='SGE-USDT-SWAP' RETURNING id`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_proposal WHERE id=$1", pid).Scan(&st); err != nil || st != "fin_proposed" {
		t.Fatalf("越线应拦留 fin_proposed: state=%s err=%v", st, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='shape_gate_principal' AND event='升件'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("越本金线应落升件行归业主面: n=%d err=%v", n, err)
	}
}

// openLive 面：cell 凭据全绿但 NAV 越线→r.op.Open 未达，策略留 live
// 未开（fakeOpener.calls=0）+升件行落账。
func TestShapeGateOpenLiveBlock(t *testing.T) {
	ctx, r, fo := openerSetup(t)
	seedFlip(t, ctx, r)
	seedFunding(t, ctx, r)
	mkStrategy(t, ctx, r, "live", time.Hour)
	seedAllCellsGreen(t, ctx, r)
	// NAV 150 < demand 2×100 →越线拦在 Open 前。
	if _, err := r.pool.Exec(ctx, "UPDATE md_equity_curve SET equity=150 WHERE ccy='NAV'"); err != nil {
		t.Fatal(err)
	}
	r.openLive(ctx)
	if len(fo.calls) != 0 {
		t.Fatalf("越线应拦在 r.op.Open 前: calls=%d", len(fo.calls))
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='shape_gate_principal'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("openLive 越线应升件: n=%d err=%v", n, err)
	}
}

// promote 面：probe→live 的 Transition 前拦——判据满（演习绿+置信过+
// 开关翻）但 NAV 判据缺→策略留 probe 不晋 live；判据缺非越线不升件。
func TestShapeGatePromoteBlock(t *testing.T) {
	ctx, r, _ := openerSetup(t)
	seedFlip(t, ctx, r)
	seedFunding(t, ctx, r)
	sid := mkStrategy(t, ctx, r, "probe", 2*time.Hour)
	for i := 0; i < 3; i++ {
		if _, err := r.pool.Exec(ctx,
			"INSERT INTO fin_paper_fill(strategy_id,kind,inst_id,amount_usd,ts) VALUES($1,'funding','OP-USDT-SWAP',0.5,now())", sid); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_exec_drill(strategy_ref,inst_id,verdict,rules_version,detail)
		VALUES($1,'OP-USDT-SWAP','pass','simvenue/1','shapegate 闸测凭据')`, sid); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx,
		"UPDATE app_config SET value='\"true\"' WHERE key='invest.lifecycle.live_switch'"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, "DELETE FROM md_equity_curve WHERE ccy='NAV'"); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := r.pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", sid).Scan(&st); err != nil || st != "probe" {
		t.Fatalf("判据缺 probe→live 应拦留 probe: state=%s err=%v", st, err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_scope='shape_gate_principal'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("判据缺非越线不应升件: n=%d err=%v", n, err)
	}
}
