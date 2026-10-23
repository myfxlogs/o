package lifecycle

import (
	"context"
	"testing"

	"phase/internal/platform/config"
	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

func TestDbg4(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
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
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_opp_eval(inst_id,template,direction,verdict,legs,evidence)
		VALUES('LC-USDT-SWAP','funding-hedge','short_perp_long_spot','proposed','{}','{}') RETURNING id`).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		"INSERT INTO fin_proposal(eval_id,inst_id,direction,net_apr) VALUES($1,'LC-USDT-SWAP','short_perp_long_spot',1.0)", eid); err != nil {
		t.Fatal(err)
	}
	r := NewRunner(pool, cfg)
	if err := r.scan(ctx); err != nil {
		var sch string
		_ = pool.QueryRow(ctx, "SHOW search_path").Scan(&sch)
		t.Fatalf("scan err=%v schema=%s", err, sch)
	}
}
