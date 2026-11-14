package risk

import (
	"context"
	"testing"

	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Gate) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 隔离硬化（T-0021）：专属 fixture 名空间清场+判据阈远离他包值域——
	// 共享库并发下他包策略行/快照行可能顶过阈翻判（venue 集中度/敞口误伤）。
	for _, q := range []string{
		"DELETE FROM fin_strategy WHERE inst_id LIKE 'RK-%'",
		"DELETE FROM fin_risk_token WHERE true",
		"DELETE FROM fin_risk_decision WHERE intent->>'inst_id' LIKE 'RK-%'",
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	// T-0355：迁移种子 dcd_baseline（BTC 0.0207）入闸后须有价——BTC-USDT
	// tick fixture 供估值径（幂等 upsert；risk 包测不涉其他 md_* 行）。
	if _, err := pool.Exec(ctx,
		`INSERT INTO md_ticks(ts,venue,inst_id,last) VALUES(now(),'okx','BTC-USDT',100000)
		 ON CONFLICT(venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last`); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{
		"risk.kill_switch": "false", "risk.max_notional_usd": "500",
		"risk.max_venue_pct": "0.95", "risk.max_exposure_usd": "1000000000",
		"risk.token_ttl_seconds": "30"} {
		if _, err := pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	return ctx, NewGate(pool, cfg).WithEquity(func(ctx context.Context, v string) (float64, error) { return 1000, nil })
}

var intent = Intent{StrategyRef: 9e6, Venue: "rktest", InstID: "RK-USDT-SWAP", Legs: []string{"spot_long", "perp_short"}, NotionalUSD: 100, Leverage: 1}

func TestAllowDenyAnchors(t *testing.T) {
	ctx, g := setup(t)
	// allow 锚
	d := g.Check(ctx, intent)
	if !d.Allow || d.TokenID == "" {
		t.Fatalf("allow: %+v", d)
	}
	// intent 缺字段 deny 锚
	if d := g.Check(ctx, Intent{Venue: "rktest"}); d.Allow || d.Reason != "intent_incomplete" {
		t.Fatalf("intent_incomplete: %+v", d)
	}
	// 名义上限 deny 锚
	big := intent
	big.NotionalUSD = 999
	if d := g.Check(ctx, big); d.Allow || d.Reason != "notional_cap" {
		t.Fatalf("notional_cap: %+v", d)
	}
	// 每判留痕+reason 分桶
	var n int
	if err := g.pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_risk_decision").Scan(&n); err != nil || n < 3 {
		t.Fatalf("留痕行=%d err=%v", n, err)
	}
	var gen int64
	if err := g.pool.QueryRow(ctx,
		"SELECT config_gen FROM fin_risk_decision ORDER BY id DESC LIMIT 1").Scan(&gen); err != nil || gen == 0 {
		t.Fatalf("代次未留 gen=%d", gen)
	}
}

func TestKillSwitchPriority(t *testing.T) {
	ctx, g := setup(t)
	if _, err := g.pool.Exec(ctx,
		"UPDATE app_config SET value='\"true\"' WHERE key='risk.kill_switch'"); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	// 翻即全拒（含原本合规意图）；此前 allow 不溯及（留痕在）
	d := g.Check(ctx, intent)
	if d.Allow || d.Reason != "kill_switch" {
		t.Fatalf("kill_switch: %+v", d)
	}
}

func TestMissingParamFailClosed(t *testing.T) {
	ctx, g := setup(t)
	if _, err := g.pool.Exec(ctx, "DELETE FROM app_config WHERE key='risk.max_exposure_usd'"); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	d := g.Check(ctx, intent)
	if d.Allow || d.Reason != "missing_param" {
		t.Fatalf("missing_param: %+v", d)
	}
}

func TestTokenMechanics(t *testing.T) {
	ctx, g := setup(t)
	// 无令牌执行拒
	if d := g.ConsumeToken(ctx, "00000000-0000-0000-0000-000000000000", intent); d.Allow || d.Reason != "token_missing" {
		t.Fatalf("token_missing: %+v", d)
	}
	d := g.Check(ctx, intent)
	if !d.Allow {
		t.Fatal("allow 未产令牌")
	}
	// 过期锚：手工把 expires_at 拨过去
	if _, err := g.pool.Exec(ctx,
		"UPDATE fin_risk_token SET expires_at=now()-interval '1 second' WHERE token_id=$1", d.TokenID); err != nil {
		t.Fatal(err)
	}
	if x := g.ConsumeToken(ctx, d.TokenID, intent); x.Allow || x.Reason != "token_expired" {
		t.Fatalf("token_expired: %+v", x)
	}
	// 单次性锚：新令牌用两次
	d2 := g.Check(ctx, intent)
	if x := g.ConsumeToken(ctx, d2.TokenID, intent); !x.Allow {
		t.Fatalf("首用应通: %+v", x)
	}
	if x := g.ConsumeToken(ctx, d2.TokenID, intent); x.Allow || x.Reason != "token_used" {
		t.Fatalf("复用应拒 token_used: %+v", x)
	}
}

// TestDirBookGate 方向书子预算闸（棒3）：方向书成员开仓额外两判——
// 亏完断粮(已实现净亏破阈)先于书名义帽(open+在途 dir 名义+意图>帽)；
// 非方向/查无行原径放行;close 径不拦;键缺 fail-closed missing_param。
func TestDirBookGate(t *testing.T) {
	ctx, g := setup(t)
	for _, q := range []string{ // 本测专属种子面清场（共享库防串）
		`DELETE FROM fin_cashflow WHERE source_table='RKDIR'`,
		`DELETE FROM fin_position WHERE inst_id='RK-DIR'`,
		`DELETE FROM fin_execution WHERE inst_id='RK-DIR'`,
	} {
		if _, err := g.pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	for k, v := range map[string]string{ // 生产种子=百分比基线;测试显式供数(equity 夹具=1000)
		"risk.dir.max_notional_pct": "0.20", "risk.dir.loss_budget_pct": "0.10",
		"risk.dir.trade_loss_pct": "0.005", "lifecycle.dir.stop_frac": "0.15"} {
		if _, err := g.pool.Exec(ctx,
			"INSERT INTO app_config(key,value) VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value", k, v); err != nil {
			t.Fatal(err)
		}
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	var eid, pid, sid int64 // eval→proposal→strategy 链——direction=long_perp 方向书成员
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_opp_eval
		(inst_id,template,direction,net_apr,verdict,legs,evidence)
		VALUES('RK-DIR','dir_perp_mom','long_perp',0.1,'proposed','[]','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_proposal
		(eval_id,inst_id,direction,net_apr,state)
		VALUES($1,'RK-DIR','long_perp',0.1,'fin_proposed') RETURNING id`, eid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_strategy
		(inst_id,template_ref,proposal_ref,state,notional_usd,direction)
		VALUES('RK-DIR','dir_perp_mom',$1,'live',100,'long_perp') RETURNING id`, pid).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	din := intent
	din.StrategyRef = sid
	din.NotionalUSD = 30 // equity=1000:单笔帽=0.005×1000÷0.15=33.3;书帽=200

	if d := g.Check(ctx, din); !d.Allow { // ①书内零仓零亏+意图30×0.15=4.5≤5=放行
		t.Fatalf("方向书空账应放: %+v", d)
	}
	bin := din // ②意图 40×0.15=6>5→dir_trade_loss_cap(判序先于书帽)
	bin.NotionalUSD = 40
	if d := g.Check(ctx, bin); d.Allow || d.Reason != "dir_trade_loss_cap" {
		t.Fatalf("dir_trade_loss_cap: %+v", d)
	}
	var xid, posid int64 // ③open dir 仓 180+意图 30=210>帽 200→dir_notional_cap
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_execution
		(strategy_ref,inst_id,notional_usd,state) VALUES($1,'RK-DIR',180,'open') RETURNING id`, sid).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := g.pool.QueryRow(ctx, `INSERT INTO fin_position
		(execution_id,strategy_ref,inst_id,notional_usd) VALUES($1,$2,'RK-DIR',180) RETURNING id`,
		xid, sid).Scan(&posid); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, din); d.Allow || d.Reason != "dir_notional_cap" {
		t.Fatalf("dir_notional_cap: %+v", d)
	}
	// ④已实现-120<-100(0.10×1000)→dir_budget_exhausted(判序先于两帽)
	if _, err := g.pool.Exec(ctx, `INSERT INTO fin_cashflow
		(kind,venue,ccy,amount,position_ref,source_ref,source_table,occurred_at)
		VALUES('trade_pnl','rktest','USDT',-120,$1,$1,'RKDIR',now())`, posid); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, din); d.Allow || d.Reason != "dir_budget_exhausted" {
		t.Fatalf("dir_budget_exhausted: %+v", d)
	}
	if d := g.Check(ctx, intent); !d.Allow { // ⑤查无行=非方向书成员原径
		t.Fatalf("非方向应原径放: %+v", d)
	}
	cin := din
	cin.Kind = "close" // ⑥close 径不拦(退仓权保留)
	if d := g.Check(ctx, cin); !d.Allow {
		t.Fatalf("close 径应放: %+v", d)
	}
	if _, err := g.pool.Exec(ctx, `DELETE FROM app_config WHERE key LIKE 'risk.dir.%'`); err != nil {
		t.Fatal(err)
	}
	if err := g.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if d := g.Check(ctx, din); d.Allow || d.Reason != "missing_param" { // ⑦键缺 fail-closed
		t.Fatalf("missing_param: %+v", d)
	}
}
