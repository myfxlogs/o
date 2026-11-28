package main

// valuation_ccy_beat_test.go —— O-41395 件②发布器锚：FundCcy∪视图
// ccy 集合同步 fin_valuation_ccy（在集 upsert/出集 DELETE/直入币滤除/
// 同值零写幂等）。

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func TestSyncValuationCcy(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	// 清场：测试 ccy 前缀+既有发布行（共享 schema 可能有他测残行——
	// 断言只数 LWPVC% 前缀键，全集合同步语义照旧验证）。
	defer func() {
		_, _ = pool.Exec(ctx, `DELETE FROM fin_valuation_ccy WHERE ccy LIKE 'LWPVC%'`)
		_, _ = pool.Exec(ctx, `DELETE FROM fin_account WHERE account_id LIKE 'okx:lwpvc%'`)
		_, _ = pool.Exec(ctx, `DELETE FROM fin_risk_decision WHERE intent->>'ccy' LIKE 'LWPVC%'`)
	}()
	// 种子①fin_account active 币（FundCcy 种子子）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_account(account_id,venue,ccy,sub_type,state)
		VALUES('okx:lwpvc1','okx','LWPVC1','trading','active')`); err != nil {
		t.Fatal(err)
	}
	// 种子②dcd_baseline allow 凭据（fin_position_view.ccy 源——FundCcy
	// dcd 域同含故 src=双源并集）。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen)
		VALUES('{"op":"dcd_baseline","venue":"okx","ccy":"LWPVC2","amt":"1.5","ordId":"lwpvc-ord"}','allow','allow',1)`); err != nil {
		t.Fatal(err)
	}
	// 预置脏行：直入币+退集币——同步后应消失。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_valuation_ccy(ccy,src) VALUES
		('USDT','dirty'),('LWPVCold','stale')`); err != nil {
		t.Fatal(err)
	}
	if err := syncValuationCcy(ctx, pool); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(ctx,
		`SELECT ccy,src FROM fin_valuation_ccy WHERE ccy LIKE 'LWPVC%' ORDER BY ccy`)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for rows.Next() {
		var c, s string
		if err := rows.Scan(&c, &s); err != nil {
			t.Fatal(err)
		}
		got[c] = s
	}
	rows.Close()
	if len(got) != 2 || got["LWPVC1"] != "fundccy" || got["LWPVC2"] != "fundccy+view" {
		t.Fatalf("同步后 LWPVC 集应={LWPVC1:fundccy,LWPVC2:fundccy+view}，得 %v", got)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_valuation_ccy WHERE ccy IN ('USDT','LWPVCold')`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("直入币/退集币应出集删，残 %d 行", n)
	}
	// 幂等：再同步零变化（updated_at 不动——同值跳写）。
	var before string
	if err := pool.QueryRow(ctx,
		`SELECT max(updated_at)::text FROM fin_valuation_ccy WHERE ccy='LWPVC1'`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if err := syncValuationCcy(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var after string
	if err := pool.QueryRow(ctx,
		`SELECT max(updated_at)::text FROM fin_valuation_ccy WHERE ccy='LWPVC1'`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("同值再同步不应动 updated_at（%s→%s）", before, after)
	}
}
