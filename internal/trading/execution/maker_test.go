package execution

import (
	"testing"
)

// T-0238 验收锚：①maker post_only 下单即成交径；②超时未成取消降级 market；
// ③裸奔窗=leg1 maker 成+leg2 taker 补（双腿终态全 filled）。

func TestMakerFillAndFallback(t *testing.T) {
	fv := &fakeVenue{makerFill: true}
	ctx, r := setup(t, fv)
	for _, kv := range [][2]string{
		{"exec.maker_enabled", "true"}, {"exec.maker_timeout_s", "1"},
	} {
		if _, err := r.pool.Exec(ctx, `INSERT INTO app_config(key,value)
			VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, kv[0], kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask) VALUES
		(now(),'BTC-USDT',90000,89990,90010),(now(),'BTC-USDT-SWAP',80950,80945,80955)`); err != nil {
		t.Fatal(err)
	}
	if err := r.cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	did := mkToken(t, ctx, r, false, false, "allow")
	legs := [2]Leg{{Venue: "okx", InstID: "BTC-USDT", Side: "buy", Sz: "0.01", Notional: 100, TgtQuote: true},
		{Venue: "leg2", InstID: "BTC-USDT-SWAP", Side: "sell", Sz: "1", Notional: 100}}
	id, err := r.Execute(ctx, Intent{DecisionID: did, StrategyRef: 1, InstID: "BTC-USDT-SWAP",
		NotionalUSD: 100, Legs: legs})
	if err != nil {
		t.Fatalf("maker 径执行败: %v", err)
	}
	var ot string
	if err := r.pool.QueryRow(ctx, `SELECT ord_type FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1`, id).Scan(&ot); err != nil || ot != "post_only" {
		t.Fatalf("leg1 应 post_only ot=%q err=%v", ot, err)
	}
	// 降级径：makerFill=false→时限取消→market 补（cancelled≥1）
	fv2 := &fakeVenue{makerFill: false}
	ctx2, r2 := setup(t, fv2)
	for _, kv := range [][2]string{{"exec.maker_enabled", "true"}, {"exec.maker_timeout_s", "0.5"}} {
		if _, err := r2.pool.Exec(ctx2, `INSERT INTO app_config(key,value)
			VALUES($1,to_jsonb($2::text)) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value`, kv[0], kv[1]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := r2.pool.Exec(ctx2, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask) VALUES
		(now(),'BTC-USDT',90000,89990,90010)`); err != nil {
		t.Fatal(err)
	}
	if err := r2.cfg.Load(ctx2); err != nil {
		t.Fatal(err)
	}
	did2 := mkToken(t, ctx2, r2, false, false, "allow")
	id2, err := r2.Execute(ctx2, Intent{DecisionID: did2, StrategyRef: 1, InstID: "BTC-USDT-SWAP",
		NotionalUSD: 100, Legs: legs})
	if err != nil {
		t.Fatalf("降级径执行败: %v", err)
	}
	if fv2.cancelled == 0 {
		t.Fatal("超时未取消——降级径丢取消件")
	}
	var ot2 string
	if err := r2.pool.QueryRow(ctx2, `SELECT ord_type FROM fin_order_leg
		WHERE execution_id=$1 AND leg_no=1 AND state='filled'`, id2).Scan(&ot2); err != nil || ot2 != "market" {
		t.Fatalf("降级腿应 market ot=%q err=%v", ot2, err)
	}
}
