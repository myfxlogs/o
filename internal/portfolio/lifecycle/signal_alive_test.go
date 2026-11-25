package lifecycle

import (
	"context"
	"testing"
)

// O-39955 径B 信号存续锚桩测：probe/live 休眠候选先过「信号死没死」判——
// 信号活着（eval 仍 proposed/率同号/报价守基准）=豁免退役；信号死=
// armed_ttl_expired 照退。判据=entryGate 族判词同函数直调（判据一源律）。

// sigFixture 策略行夹具（direction 自定——dir 族=short_perp/long_perp，
// carry 族=short_perp_long_spot 等复合向）。
func sigFixture(t *testing.T, r *Runner, inst, tmpl, dir, state, enteredAgo string) int64 {
	t.Helper()
	var sid int64
	if err := r.pool.QueryRow(context.Background(), `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,entered_at)
		VALUES($1,$2,$3,100,$4,now()-$5::interval) RETURNING id`,
		inst, tmpl, state, dir, enteredAgo).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

// 锚①：dir_perp_mom probe 零执行零锚——eval 仍 proposed=信号活着→不死
// （实案形：CRCL/MEME/CASHCAT eval proposed 在案仍死于 4h 窗）。
func TestSignalAliveDirProposedSurvives(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sid := sigFixture(t, r, "SIG-USDT-SWAP", "dir_perp_mom", "short_perp", "probe", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('SIG-USDT-SWAP','dir_perp_mom','short_perp',0.1,'proposed','{}','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "probe" {
		t.Fatalf("eval proposed 信号存续应豁免退役 got %s", got)
	}
}

// 锚②a：同条件 eval 翻 rejected=信号死→armed_ttl_expired 照退。
func TestSignalDeadDirRejectedRetires(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sid := sigFixture(t, r, "SIG2-USDT-SWAP", "dir_perp_mom", "short_perp", "probe", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('SIG2-USDT-SWAP','dir_perp_mom','short_perp',0.1,'rejected','{}','{}')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("eval rejected 信号死应退役 got %s", got)
	}
	var reason *string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 ORDER BY id DESC LIMIT 1`, sid).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason == nil || *reason != "armed_ttl_expired" {
		t.Fatalf("退役因应 armed_ttl_expired got %v", reason)
	}
}

// 锚②b：eval proposed 但行陈>72h（dirEntryGate 新鲜度界同判——dedup 冻结
// 的陈 proposed=臂停/断供形，视同信号死不豁免）。
func TestSignalDeadDirStaleRetires(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sid := sigFixture(t, r, "SIG3-USDT-SWAP", "dir_perp_mom", "short_perp", "probe", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence,ts)
		VALUES('SIG3-USDT-SWAP','dir_perp_mom','short_perp',0.1,'proposed','{}','{}',now()-interval '80 hours')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("eval 陈>72h 应视同信号死退役 got %s", got)
	}
}

// 锚③a：carry 族——funding 率同号=信号活着豁免；逆号=信号死照退。
func TestSignalAliveFundingSign(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	live := sigFixture(t, r, "FUND-USDT-SWAP", "funding-hedge", "short_perp_long_spot", "live", "25 hours")
	dead := sigFixture(t, r, "FUND2-USDT-SWAP", "funding-hedge", "short_perp_long_spot", "live", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'FUND-USDT-SWAP',0.001),(now(),'FUND2-USDT-SWAP',-0.001)`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, live); got != "live" {
		t.Fatalf("率同号 short_perp 收租位应豁免 got %s", got)
	}
	if got := stratState(t, r, dead); got != "retired" {
		t.Fatalf("率逆号应信号死退役 got %s", got)
	}
}

// 锚③b：非 perp 向 default 臂平凡放行≠信号证——xvenue-carry cfd 向
// 零判词可核=不豁免照退（防「判词平凡真」变永生豁免）。
func TestSignalAliveNonPerpDirNoExempt(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	sid := sigFixture(t, r, "XV-USDT-SWAP", "xvenue-carry", "short_cfd", "probe", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'XV-USDT-SWAP',0.001)`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("非 perp 向平凡放行不应豁免 got %s", got)
	}
}

// 锚④：vol 族——报价守立项基准×0.5=存续豁免；塌于半值=信号死照退。
func TestSignalAliveVolGate(t *testing.T) {
	r := newRig(t)
	ctx := context.Background()
	// eval→proposal→strategy 全链（proposal_ref 供 refAPR=立项 net_apr）。
	var eid, pid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('VOL-P','dcd_call','dcd_call',0.12,'proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,'VOL-P','dcd_call',0.12) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	var alive, dead int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,proposal_ref,state,notional_usd,direction,entered_at)
		VALUES('VOL-P','dcd_call',$1,'probe',100,'dcd_call',now()-interval '25 hours') RETURNING id`, pid).Scan(&alive); err != nil {
		t.Fatal(err)
	}
	// 死例：refAPR 缺（零 proposal_ref）→volEntryGate 延评=信号死照退。
	dead = sigFixture(t, r, "VOL-D", "dcd_call", "dcd_call", "probe", "25 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_dcd_product(ts,venue,surface,uly,product_id,direction,
		strike,expiry,annualized_yield,spot,raw)
		VALUES(now(),'okx','products','BTC-USDT','VOL-P','C',
		68000,now()+interval '30 days',0.14,88000,'{}')`); err != nil {
		t.Fatal(err)
	}
	if err := r.retireStaleArmed(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, alive); got != "probe" {
		t.Fatalf("报价 0.14≥立项0.12×0.5 应豁免 got %s", got)
	}
	if got := stratState(t, r, dead); got != "retired" {
		t.Fatalf("refAPR 缺应视同信号死退役 got %s", got)
	}
}
