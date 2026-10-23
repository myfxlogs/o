package lifecycle

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
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
			VALUES('DD-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
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
	// 两提案都 armed（被吸收不滞留 proposed）
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_proposal WHERE inst_id='DD-USDT-SWAP' AND state='fin_armed'").Scan(&n); err != nil || n != 2 {
		t.Fatalf("armed 提案=%d", n)
	}
	// 部分唯一索引拒第二 active 行
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue,direction)
		VALUES('DD-USDT-SWAP','funding-hedge','paper',1,'okx','short_perp_long_spot')`); err == nil {
		t.Fatal("同键第二 active 行未被索引拒")
	}
}
