package main

import (
	"context"
	"testing"

	"phase/internal/platform/testutil"
)

// T-0042 锚：默认策略径（sid=0）回赋真实 strategy id——防 StrategyRef 恒 0
// →intent_incomplete（decision=8 实证）。
func TestResolveStrategyDefaultScansID(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	// fixture 自含：CI 测试实例无基线表（迁移登记表只载增量——fin_strategy 属 0007 未登记段）
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS fin_strategy (
		id bigserial PRIMARY KEY, inst_id text NOT NULL, template_ref text NOT NULL,
		proposal_ref bigint, state text NOT NULL, notional_usd numeric NOT NULL,
		entered_at timestamptz NOT NULL DEFAULT now(), criteria jsonb NOT NULL DEFAULT '{}',
		created_at timestamptz NOT NULL DEFAULT now(), venue text NOT NULL DEFAULT 'okx')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_strategy(inst_id,template_ref,state,notional_usd,venue)
		VALUES('BTC-USDT-SWAP','tpl','probe',100,'okx')`); err != nil {
		t.Fatal(err)
	}
	var wantID int64
	if err := pool.QueryRow(ctx, `SELECT id FROM fin_strategy ORDER BY id DESC LIMIT 1`).Scan(&wantID); err != nil {
		t.Fatal(err)
	}
	sid, instID, notional, venue, err := resolveStrategy(ctx, pool, 0)
	if err != nil {
		t.Fatal(err)
	}
	if sid != wantID || sid == 0 || instID != "BTC-USDT-SWAP" || notional != 100 || venue != "okx" {
		t.Fatalf("默认径 sid 回赋败: sid=%d want=%d inst=%s notional=%v venue=%s", sid, wantID, instID, notional, venue)
	}
}
