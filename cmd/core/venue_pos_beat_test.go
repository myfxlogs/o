package main

// venue_pos_beat_test.go —— O-33207 B 面桩测：OKX 手动仓快照→
// fin_venue_position upsert/O 单 instId 排除（不双簿）/stale 清场/
// mtm_off 直给值域/账户归户注册表解析。

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func venuePosPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM fin_venue_position WHERE venue='okx'`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func okxPos(inst, posSide, pos, posID, upl string) okxexec.Position {
	return okxexec.Position{InstID: inst, InstType: "SWAP", PosSide: posSide,
		Pos: pos, PosID: posID, AvgPx: "65000", Upl: upl, Ccy: "USDT", CTime: "1764000000000"}
}

// TestVenuePosManualUpsert 手动仓（instId 无 O 在持）→open 行+mtm_off
// 直给；二拍 upl 变→行更新不重键。
func TestVenuePosManualUpsert(t *testing.T) {
	pool := venuePosPool(t)
	ctx := context.Background()
	fn := func(context.Context) ([]okxexec.Position, error) {
		return []okxexec.Position{okxPos("BTC-USDT-SWAP", "net", "0.5", "pos-1", "12.5")}, nil
	}
	if err := venuePosOnce(ctx, pool, fn); err != nil {
		t.Fatal(err)
	}
	var st, side, ccy string
	var off float64
	if err := pool.QueryRow(ctx,
		`SELECT state,side,ccy,mtm_off FROM fin_venue_position
		 WHERE account_id='okx:trading' AND ticket='pos-1'`).
		Scan(&st, &side, &ccy, &off); err != nil {
		t.Fatalf("手动仓行缺: %v", err)
	}
	if st != "open" || side != "buy" || ccy != "USDT" || off != 12.5 {
		t.Fatalf("行域错 st=%s side=%s ccy=%s off=%v", st, side, ccy, off)
	}
	// upl 漂移→同票 upsert 更新不新增
	fn2 := func(context.Context) ([]okxexec.Position, error) {
		return []okxexec.Position{okxPos("BTC-USDT-SWAP", "net", "0.5", "pos-1", "-3.0")}, nil
	}
	if err := venuePosOnce(ctx, pool, fn2); err != nil {
		t.Fatal(err)
	}
	var n int
	var off2 float64
	if err := pool.QueryRow(ctx,
		`SELECT count(*),max(mtm_off) FROM fin_venue_position WHERE ticket='pos-1'`).
		Scan(&n, &off2); err != nil || n != 1 || off2 != -3.0 {
		t.Fatalf("upsert 失效 n=%d off=%v err=%v", n, off2, err)
	}
}

// TestVenuePosOwnExcluded O 在持同 instId→venue 仓判 O 的不入册
// （双簿防线——合并仓不可分保守方向）。
func TestVenuePosOwnExcluded(t *testing.T) {
	pool := venuePosPool(t)
	ctx := context.Background()
	var eid int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO fin_execution(inst_id,notional_usd,state) VALUES('BTC-USDT-SWAP',100,'open') RETURNING id`).
		Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO fin_position(execution_id,inst_id,notional_usd,state) VALUES($1,'BTC-USDT-SWAP',100,'open')`, eid); err != nil {
		t.Fatal(err)
	}
	fn := func(context.Context) ([]okxexec.Position, error) {
		return []okxexec.Position{okxPos("BTC-USDT-SWAP", "net", "0.5", "pos-o", "9.9")}, nil
	}
	if err := venuePosOnce(ctx, pool, fn); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM fin_venue_position WHERE ticket='pos-o'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("O 单误入手动仓册 n=%d err=%v", n, err)
	}
}

// TestVenuePosStaleClear 快照消失→stale 清出 mtm；读败跳拍不清场。
func TestVenuePosStaleClear(t *testing.T) {
	pool := venuePosPool(t)
	ctx := context.Background()
	mk := func(upl string) func(context.Context) ([]okxexec.Position, error) {
		return func(context.Context) ([]okxexec.Position, error) {
			return []okxexec.Position{okxPos("ETH-USDT-SWAP", "net", "1", "pos-2", upl)}, nil
		}
	}
	if err := venuePosOnce(ctx, pool, mk("1")); err != nil {
		t.Fatal(err)
	}
	// 空快照=全平→stale
	empty := func(context.Context) ([]okxexec.Position, error) { return nil, nil }
	if err := venuePosOnce(ctx, pool, empty); err != nil {
		t.Fatal(err)
	}
	var st string
	if err := pool.QueryRow(ctx,
		`SELECT state FROM fin_venue_position WHERE ticket='pos-2'`).Scan(&st); err != nil || st != "stale" {
		t.Fatalf("消失仓未 stale st=%s err=%v", st, err)
	}
	// 读败→err 上抛且不清场（fail-closed——在册行原样）
	bad := func(context.Context) ([]okxexec.Position, error) {
		return nil, context.DeadlineExceeded
	}
	if err := venuePosOnce(ctx, pool, bad); err == nil {
		t.Fatal("读败未上抛")
	}
}
