package lifecycle

import (
	"context"
	"testing"
)

// seedDir 方向族 paper 策略夹具（dir_perp_mom——PaperOnly 册唯一员；
// direction 直给——stratFixture 钉死 short_perp_long_spot 不可复用）。
func seedDir(t *testing.T, r *Runner, inst, dir string, enteredAgo string) int64 {
	t.Helper()
	ctx := context.Background()
	var sid int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,direction,entered_at)
		VALUES($1,'dir_perp_mom','paper',1000,$2,now()-$3::interval) RETURNING id`,
		inst, dir, enteredAgo).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func setTick(t *testing.T, r *Runner, inst string, px float64) {
	t.Helper()
	if _, err := r.pool.Exec(context.Background(), `INSERT INTO md_ticks(venue,inst_id,last,ts)
		VALUES('okx',$1,$2,now()) ON CONFLICT (venue,inst_id) DO UPDATE SET last=EXCLUDED.last,ts=EXCLUDED.ts`,
		inst, px); err != nil {
		t.Fatal(err)
	}
}

// dirRig 方向面判据种子（mark_iv 趋零=每拍可摊；停/盈/TTL 阈小直给）。
func dirRig(t *testing.T) *Runner {
	t.Helper()
	r := newRig(t)
	ctx := context.Background()
	for k, v := range map[string]string{
		"lifecycle.dir.mark_iv_min": "0.001", "lifecycle.dir.stop_frac": "0.15",
		"lifecycle.dir.take_frac": "0.10", "lifecycle.dir.ttl_h": "240",
	} {
		if _, err := r.pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return r
}

// 棒1 锚：dir paper 链三段——①开户锚 open fill(px 载 detail) ②MTM 摊提
// pnl delta=sign·notional·(px/prev−1) ③止损出场→close fill+retired 留痕。
func TestDirPaperStopLoss(t *testing.T) {
	r := dirRig(t)
	ctx := context.Background()
	sid := seedDir(t, r, "DIRT-USDT-SWAP", "long_perp", "1 hours")
	setTick(t, r, "DIRT-USDT-SWAP", 110)
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var kind string
	var px float64
	if err := r.pool.QueryRow(ctx, `SELECT kind,(detail#>>'{px}')::float8 FROM fin_paper_fill
		WHERE strategy_id=$1`, sid).Scan(&kind, &px); err != nil || kind != "open" || px != 110 {
		t.Fatalf("开户锚 kind=%s px=%v err=%v 期 open/110", kind, px, err)
	}
	// -20% 价移：delta=1000·(88/110−1)=−200≤−150 止损阈→dir_stop_loss 退役。
	// 摊提节距闸绕（open 锚戳回 10min——粒度闸非事实缺，拍内时差<阈）。
	if _, err := r.pool.Exec(ctx, `UPDATE fin_paper_fill SET ts=now()-interval '10 minutes' WHERE strategy_id=$1`, sid); err != nil {
		t.Fatal(err)
	}
	setTick(t, r, "DIRT-USDT-SWAP", 88)
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var delta float64
	if err := r.pool.QueryRow(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='pnl'`, sid).Scan(&delta); err != nil {
		t.Fatalf("pnl 摊提笔缺: %v", err)
	}
	if delta < -205 || delta > -195 {
		t.Fatalf("pnl delta=%v 期≈-200", delta)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("止损后 state=%s 期 retired", got)
	}
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND to_state='retired'`, sid).Scan(&reason); err != nil || reason != "dir_stop_loss" {
		t.Fatalf("退役因=%q err=%v 期 dir_stop_loss", reason, err)
	}
	var kinds int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='close'`, sid).Scan(&kinds); err != nil || kinds != 1 {
		t.Fatalf("close fill 数=%v err=%v 期 1", kinds, err)
	}
}

// 棒1 锚：信号翻面出场——本向最新 eval 已拒=持仓信念消亡即退（belief
// 续存判据与 TTL 机械闸互补）。
func TestDirPaperSignalFlip(t *testing.T) {
	r := dirRig(t)
	ctx := context.Background()
	sid := seedDir(t, r, "DIRS-USDT-SWAP", "short_perp", "1 hours")
	setTick(t, r, "DIRS-USDT-SWAP", 100)
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('DIRS-USDT-SWAP','dir_perp_mom','short_perp',0,'rejected','{}','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_paper_fill SET ts=now()-interval '10 minutes' WHERE strategy_id=$1`, sid); err != nil {
		t.Fatal(err)
	}
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("翻面后 state=%s 期 retired", got)
	}
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND to_state='retired'`, sid).Scan(&reason); err != nil || reason != "dir_signal_flip" {
		t.Fatalf("退役因=%q err=%v 期 dir_signal_flip", reason, err)
	}
}

// 棒1 锚：止盈出场——+18% 价移 delta=+180≥take 阈 100→dir_take_profit。
func TestDirPaperTakeProfit(t *testing.T) {
	r := dirRig(t)
	ctx := context.Background()
	sid := seedDir(t, r, "DIRU-USDT-SWAP", "long_perp", "1 hours")
	setTick(t, r, "DIRU-USDT-SWAP", 100)
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_paper_fill SET ts=now()-interval '10 minutes' WHERE strategy_id=$1`, sid); err != nil {
		t.Fatal(err)
	}
	setTick(t, r, "DIRU-USDT-SWAP", 118)
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	if got := stratState(t, r, sid); got != "retired" {
		t.Fatalf("止盈后 state=%s 期 retired", got)
	}
	var reason string
	if err := r.pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND to_state='retired'`, sid).Scan(&reason); err != nil || reason != "dir_take_profit" {
		t.Fatalf("退役因=%q err=%v 期 dir_take_profit", reason, err)
	}
	var delta float64
	if err := r.pool.QueryRow(ctx, `SELECT amount_usd FROM fin_paper_fill
		WHERE strategy_id=$1 AND kind='pnl'`, sid).Scan(&delta); err != nil || delta < 175 || delta > 185 {
		t.Fatalf("pnl delta=%v err=%v 期≈+180", delta, err)
	}
}

// 棒1 锚：价陈=让位不造 fill（fail-closed 同 T-0171——非 fatal 零 fill）。
func TestDirPaperStaleMark(t *testing.T) {
	r := dirRig(t)
	ctx := context.Background()
	sid := seedDir(t, r, "DIRST-USDT-SWAP", "long_perp", "1 hours")
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(venue,inst_id,last,ts)
		VALUES('okx','DIRST-USDT-SWAP',100,now()-interval '1 hour')
		ON CONFLICT (venue,inst_id) DO UPDATE SET last=EXCLUDED.last,ts=EXCLUDED.ts`); err != nil {
		t.Fatal(err)
	}
	if err := r.dirPaperAccounting(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM fin_paper_fill WHERE strategy_id=$1`, sid).Scan(&n); err != nil || n != 0 {
		t.Fatalf("陈价仍造 fill n=%d err=%v 期 0", n, err)
	}
}
