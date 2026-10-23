package risk

import (
	"context"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func setup(t *testing.T) (context.Context, *Gate) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// 隔离硬化（T-0021）：专属 fixture 名空间清场+判据阈远离他包值域——
	// 共享库并发下他包策略行/快照行可能顶过阈翻判（venue 集中度/敞口误伤）。
	for _, q := range []string{
		"DELETE FROM fin_strategy WHERE inst_id LIKE 'RK-%'",
		"DELETE FROM fin_venue_account_snapshot WHERE ccy='RKCCY'",
		"DELETE FROM fin_risk_token WHERE true",
		"DELETE FROM fin_risk_decision WHERE intent->>'inst_id' LIKE 'RK-%'",
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatal(err)
		}
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
	return ctx, NewGate(pool, cfg)
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
