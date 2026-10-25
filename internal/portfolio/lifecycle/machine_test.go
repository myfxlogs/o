package lifecycle

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"testing"
)

func TestLegal(t *testing.T) {
	for _, ok := range [][3]any{{"proposed", "paper", true}, {"paper", "live", false},
		{"paper", "probe", true}, {"probe", "live", true}, {"retired", "paper", false},
		{"degraded", "paper", true}} {
		if Legal(ok[0].(string), ok[1].(string)) != ok[2].(bool) {
			t.Fatalf("Legal(%s→%s)!=%v", ok[0], ok[1], ok[2])
		}
	}
}
func setup(t *testing.T) (context.Context, *pgxpool.Pool, *config.Client) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"invest.paper.notional_usd": "100", "invest.lifecycle.paper_min_hours": "0",
		"invest.lifecycle.paper_min_net_usd": "0.001", "invest.lifecycle.paper_max_dd_usd": "5",
		"invest.lifecycle.live_switch": "false"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, pool, cfg
}
func TestLifecycleE2E(t *testing.T) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_paper_fill", "DELETE FROM fin_strategy_transition",
		"DELETE FROM fin_strategy", "DELETE FROM fin_proposal WHERE inst_id='LC-USDT-SWAP'",
		"DELETE FROM md_funding WHERE inst_id='LC-USDT-SWAP'"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r := NewRunner(pool, cfg)
	// 提案→paper 开户
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('LC-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}') RETURNING id`).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,'LC-USDT-SWAP','short_perp_long_spot',1.0)", pid); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var sid int64
	var state string
	if err := pool.QueryRow(ctx,
		"SELECT id,state FROM fin_strategy WHERE inst_id='LC-USDT-SWAP'").Scan(&sid, &state); err != nil || state != "paper" {
		t.Fatalf("paper 开户败 state=%s err=%v", state, err)
	}
	// 非法迁移锚：paper→live 越级拒
	if err := Transition(ctx, pool, sid, "live", "越级试", false); err == nil {
		t.Fatal("paper→live 越级未被拒")
	}
	// live 闸：probe→live 无开关 fail-closed
	if _, err := pool.Exec(ctx, "UPDATE fin_strategy SET state='probe' WHERE id=$1", sid); err != nil {
		t.Fatal(err)
	}
	if err := Transition(ctx, pool, sid, "live", "无开关试", false); err == nil {
		t.Fatal("live_switch=false 下 live 迁移未 fail-closed")
	}
	if _, err := pool.Exec(ctx, "UPDATE fin_strategy SET state='paper' WHERE id=$1", sid); err != nil {
		t.Fatal(err)
	}
	// funding 行→虚拟记账+晋升（判据 min_hours=0 即时满）
	if _, err := pool.Exec(ctx, `INSERT INTO md_funding(ts,inst_id,funding_rate)
		VALUES(now(),'LC-USDT-SWAP',0.01)`); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_paper_fill WHERE strategy_id=$1 AND kind='funding'", sid).Scan(&n); err != nil || n == 0 {
		t.Fatalf("虚拟结算未落 fill n=%d err=%v", n, err)
	}
	if err := pool.QueryRow(ctx,
		"SELECT to_state FROM fin_strategy_transition WHERE strategy_id=$1 ORDER BY id DESC LIMIT 1", sid).Scan(&state); err != nil || state != "probe" {
		t.Fatalf("晋升未发生 to=%s err=%v", state, err)
	}
}

// 缺判据键 fail-closed 锚。
func TestMissingCriteriaFailClosed(t *testing.T) {
	ctx, pool, cfg := setup(t)
	if _, err := pool.Exec(ctx, "DELETE FROM app_config WHERE key='invest.lifecycle.paper_min_hours'"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err == nil {
		t.Fatal("缺判据键 scan 应 fail-closed")
	}
	if r.rej.Snapshot()["lifecycle/missing_criteria"] == 0 {
		t.Fatal("missing_criteria 门未计")
	}
}

// T-0022 去重锚：同键连拍两提案→strategy=1（armed 复用不叠仓）；
// 第二 active 行直插拒（部分唯一索引）。
func TestPaperDedupe(t *testing.T) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{
		"DELETE FROM fin_strategy_transition", "DELETE FROM fin_strategy WHERE inst_id='DD-USDT-SWAP'",
		"DELETE FROM fin_proposal WHERE inst_id='DD-USDT-SWAP'"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	r := NewRunner(pool, cfg)
	mkProp := func() {
		var eid int64
		if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
			VALUES('DD-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}')
			ON CONFLICT (inst_id,template,direction) DO UPDATE SET ts=now() RETURNING id`).Scan(&eid); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx,
			"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,'DD-USDT-SWAP','short_perp_long_spot',1.0)", eid); err != nil {
			t.Fatal(err)
		}
	}
	mkProp()
	mkProp() // 连拍两提案
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_strategy WHERE inst_id='DD-USDT-SWAP'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("同键策略行=%d（应=1 armed 复用）", n)
	}
	// T-0069：同键只留最新一行 armed——旧武装退役（被吸收不滞留 proposed
	// 也不淤积 armed）。两提案：1 armed + 1 retired。
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id='DD-USDT-SWAP' AND state='fin_armed'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("armed 提案=%d（应=1 同键只留最新）", n)
	}
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id='DD-USDT-SWAP' AND state='fin_retired'").Scan(&n); err != nil || n != 1 {
		t.Fatalf("retired 提案=%d（应=1 旧武装退役）", n)
	}
	// 部分唯一索引拒第二 active 行
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES('DD-USDT-SWAP','funding-hedge','paper',1,'okx','short_perp_long_spot')`); err == nil {
		t.Fatal("同键第二 active 行未被索引拒")
	}
}

// T-0098 方案 A 锚：degraded 超龄自动 retired+transition 留痕；龄内不波及；
// 同键新提案可重新开 paper（#1 场景桩测——860 被吸径解锁）。
func TestDegradedTTLRetire(t *testing.T) {
	ctx, pool, cfg := setup(t)
	for _, q := range []string{"DELETE FROM fin_strategy_transition", "DELETE FROM fin_strategy",
		"DELETE FROM fin_proposal", "DELETE FROM fin_opp_eval"} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	var oldID, youngID, paperID int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,entered_at)
		VALUES('TTL-USDT-SWAP','funding-hedge','degraded',100,now()-interval '30 hours') RETURNING id`).Scan(&oldID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('TTL2-USDT-SWAP','funding-hedge','degraded',100) RETURNING id`).Scan(&youngID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd)
		VALUES('TTL3-USDT-SWAP','funding-hedge','paper',100) RETURNING id`).Scan(&paperID); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var st, reason string
	if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", oldID).Scan(&st); err != nil || st != "retired" {
		t.Fatalf("超龄 degraded 未退役 state=%s err=%v", st, err)
	}
	if err := pool.QueryRow(ctx, `SELECT reason FROM fin_strategy_transition
		WHERE strategy_id=$1 AND to_state='retired'`, oldID).Scan(&reason); err != nil ||
		reason != "degraded 超龄自动退役——莫环重生窗" {
		t.Fatalf("退役留痕缺/错 reason=%q err=%v", reason, err)
	}
	for id, want := range map[int64]string{youngID: "degraded", paperID: "paper"} {
		if err := pool.QueryRow(ctx, "SELECT state FROM fin_strategy WHERE id=$1", id).Scan(&st); err != nil || st != want {
			t.Fatalf("#%d 波及 state=%s（应 %s）err=%v", id, st, want, err)
		}
	}
	// #1 场景复刻：同键新提案→paper 开户（degraded 退役=键解锁）。
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('TTL-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,'TTL-USDT-SWAP','short_perp_long_spot',1.0)", eid); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var nid int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_strategy
		WHERE inst_id='TTL-USDT-SWAP' AND state='paper'`).Scan(&nid); err != nil {
		t.Fatalf("同键新提案未开 paper（死锁仍存）: %v", err)
	}
}

// T-0124 锚：模板门闸——无 paper 径模板族（dcd_put）提案不建策略行、留
// fin_proposed 待 executor 票；SG 实证 748 DCD 提案被建成 funding-hedge
// paper 的盲晋升不得再演。
func TestTemplateGateBlocksNonPaper(t *testing.T) {
	ctx, pool, cfg := setup(t)
	r := NewRunner(pool, cfg)
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('BTC-USDT-261101-68000-P','dcd_put','dcd_put','proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr)
		VALUES($1,'BTC-USDT-261101-68000-P','dcd_put',0.11) RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := r.scan(ctx); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_strategy
		WHERE inst_id='BTC-USDT-261101-68000-P'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("dcd 提案不应建策略行 n=%d err=%v", n, err)
	}
	var st string
	if err := pool.QueryRow(ctx,
		"SELECT state FROM fin_proposal WHERE id=$1", pid).Scan(&st); err != nil || st != "fin_proposed" {
		t.Fatalf("无 executor 提案应留 fin_proposed: %s %v", st, err)
	}
}
