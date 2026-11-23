package main

import (
	"context"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0087 锚①：开现货仓后偏移含现货全值（非仅增量），与合约腿增量并轨；
// 价缺腿按 0 加不减；closed 仓不入偏移。
func TestMTMReaderSpotFullValueAndSwapIncrement(t *testing.T) {
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	var xid, pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(inst_id,notional_usd,state)
		VALUES('ETH-USDT',25,'open') RETURNING id`).Scan(&xid); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,inst_id,notional_usd)
		VALUES($1,'ETH-USDT',25) RETURNING id`, xid).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	legs := []struct {
		no         int
		side, inst string
		sz, px     float64
		szUnit     string
		qtyBase    float64
	}{
		{1, "buy", "ETH-USDT", 0.01, 2000, "base", 0.01},       // 现货腿：全值 +0.01×2100=+21（实填币量径）
		{2, "sell", "ETH-USDT-SWAP", 0.1, 2000, "contract", 0}, // 合约腿：增量 −0.1×0.1×(2100−2000)=−1
		{3, "buy", "BTC-USDT", 0.001, 50000, "base", 0.001},    // 价缺腿：无 tick 按 0 加不减
		{4, "buy", "BTC-USDT", 24.13, 0, "quote", 0},           // quote 额腿缺 base 量：跳过不产假偏移（T-0107）
	}
	for _, l := range legs {
		if _, err := pool.Exec(ctx, `INSERT INTO fin_order_leg(execution_id,leg_no,cl_ord_id,venue,side,inst_id,sz,px,state,sz_unit,qty_base)
			VALUES($1,$2,$3,'okx',$4,$5,$6,$7,'filled',$8,NULLIF($9,0))`,
			xid, l.no, "cl-t87", l.side, l.inst, l.sz, l.px, l.szUnit, l.qtyBase); err != nil {
			t.Fatal(err)
		}
	}
	for _, tk := range []struct {
		inst string
		last float64
	}{{"ETH-USDT", 2100}, {"ETH-USDT-SWAP", 2100}} {
		if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last)
			VALUES(now(),'okx',$1,$2)`, tk.inst, tk.last); err != nil {
			t.Fatal(err)
		}
	}
	ctv := func(ctx context.Context, venue, inst string) (float64, bool) {
		if inst == "ETH-USDT-SWAP" {
			return 0.1, true
		}
		return 0, false
	}
	mv, err := mtmReader(pool, ctv)(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// O-34287 验收③：非 SWAP 腿偏移=基币原量入基币键（币量不折 USDT、
	// 不需现价——量纲=币量非估值）；SWAP 增量仍 USDT 域。
	// ETH 买腿 +0.01 ETH；BTC 腿 +0.001 BTC（原币量径无价依赖）；
	// 合约腿 USDT −1；quote 额腿缺 base 量仍跳过。
	if d := mv[[2]string{"okx", "ETH"}]; d != 0.01 {
		t.Fatalf("ETH 基币偏移=%v 应 0.01（qty_base 原币量）", d)
	}
	if d := mv[[2]string{"okx", "BTC"}]; d != 0.001 {
		t.Fatalf("BTC 基币偏移=%v 应 0.001", d)
	}
	if d := mv[[2]string{"okx", "USDT"}]; d < -1.001 || d > -0.999 {
		t.Fatalf("USDT 域应仅合约增量 -1（现货全值已归基币键）: %v", d)
	}
	// closed 仓不入偏移。
	if _, err := pool.Exec(ctx, `UPDATE fin_position SET closed_at=now() WHERE id=$1`, pid); err != nil {
		t.Fatal(err)
	}
	if mv, err = mtmReader(pool, ctv)(ctx); err != nil {
		t.Fatal(err)
	} else if len(mv) != 0 {
		t.Fatalf("closed 仓应零偏移: %v", mv)
	}
}
