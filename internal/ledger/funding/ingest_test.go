package funding

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/testutil"
	"phase/internal/store/pgstore"
)

// fakeSrc 可编程账单源（pages 逐页弹夹——after 游标推进）。
type fakeSrc struct{ pages map[string][]Bill }

func (f fakeSrc) Bills(ctx context.Context, instType, after string) ([]Bill, error) {
	return f.pages[after], nil
}

func setup(t *testing.T) (context.Context, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	ctx := context.Background()
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE fin_cashflow, fin_venue_bill"); err != nil {
		t.Fatal(err)
	}
	return ctx, pool
}

// mkPos 造仓行（归仓锚——inst_id+时窗匹配）。
func mkPos(t *testing.T, ctx context.Context, pool *pgxpool.Pool, instID string) int64 {
	var eid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_execution(strategy_ref,decision_id,token_id,inst_id,notional_usd,state)
		VALUES(1,1,gen_random_uuid(),$1,100,'closed') RETURNING id`, instID).Scan(&eid); err != nil {
		t.Fatal(err)
	}
	var pid int64
	if err := pool.QueryRow(ctx, `INSERT INTO fin_position(execution_id,strategy_ref,inst_id,notional_usd,opened_at,closed_at)
		VALUES($1,1,$2,100,now()-interval '2 hours',now()+interval '2 hours') RETURNING id`, eid, instID).Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}

// T-0044 锚：bill 事件→funding_settle/fee 行落账+幂等（同 billId 重发不双记）。
func TestIngestFundingAndFeeIdempotent(t *testing.T) {
	ctx, pool := setup(t)
	mkPos(t, ctx, pool, "BTC-USDT-SWAP")
	nowMs := time.Now().Add(-time.Minute).UnixMilli() // 仓时窗内（now±2h）
	bills := []Bill{
		{BillID: "b174", SubType: "174", InstID: "BTC-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs, BalChg: 1.23, Raw: []byte(`{"billId":"b174"}`)},
		{BillID: "b173", SubType: "173", InstID: "BTC-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs + 1, BalChg: 0, Pnl: -0.5, Raw: []byte(`{}`)}, // balChg 空→pnl 兜底
		{BillID: "bfee", SubType: "5", InstID: "BTC-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs + 2, BalChg: -100.01, Fee: -0.01, Raw: []byte(`{}`)},
		{BillID: "bskip", SubType: "1", InstID: "BTC-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs + 3, BalChg: 50, Fee: 0, Raw: []byte(`{}`)}, // 非资金费无费=只留源
		{BillID: "bdust", SubType: "174", InstID: "DOGE-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs + 4, BalChg: 0.001, Raw: []byte(`{}`)}, // 无归仓→position_ref NULL
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil || n != 4 {
		t.Fatalf("首拉应记 4 单（174/173/fee/dust）got=%d err=%v", n, err)
	}
	// 幂等：同批重放（重启重放径）——零新记。
	n, err = ing.Ingest(ctx)
	if err != nil || n != 0 {
		t.Fatalf("重放应 0 新记 got=%d err=%v", n, err)
	}
	for _, tc := range []struct {
		kind    string
		amt     float64
		wantPos bool
	}{
		{"funding_settle", 1.23, true},
		{"funding_settle", -0.5, true},
		{"fee", -0.01, true},
		{"funding_settle", 0.001, false},
	} {
		var amt float64
		var pos *int64
		var cnt int
		if err := pool.QueryRow(ctx,
			"SELECT amount,position_ref,count(*) FROM fin_cashflow WHERE kind=$1 AND amount=$2 GROUP BY amount,position_ref",
			tc.kind, tc.amt).Scan(&amt, &pos, &cnt); err != nil || cnt != 1 {
			t.Fatalf("%s %.3f: 行数/读败 err=%v cnt=%d（幂等双记?）", tc.kind, tc.amt, err, cnt)
		}
		if tc.wantPos == (pos == nil) {
			t.Fatalf("%s %.3f 归仓位错: pos=%v", tc.kind, tc.amt, pos)
		}
	}
	// 非资金费无费单：只留源行不入账。
	var cnt int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_cashflow c JOIN fin_venue_bill b ON c.source_ref=b.id WHERE b.bill_id='bskip'").Scan(&cnt); err != nil || cnt != 0 {
		t.Fatalf("bskip 不应入账 cnt=%d", cnt)
	}
	// 源表 5 行全留档（对拍面）。
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_venue_bill").Scan(&cnt); err != nil || cnt != 5 {
		t.Fatalf("源行=5 got=%d", cnt)
	}
}

// 分页推进+旧账单切口：第二页首行已入库即止。
func TestIngestPaginationStopsAtKnown(t *testing.T) {
	ctx, pool := setup(t)
	mkPos(t, ctx, pool, "ETH-USDT-SWAP")
	nowMs := time.Now().Add(-time.Minute).UnixMilli()
	page1 := []Bill{{BillID: "n1", SubType: "174", InstID: "ETH-USDT-SWAP", InstType: "SWAP", Ccy: "USDT", TsMs: nowMs + 2, BalChg: 1}}
	page2 := []Bill{{BillID: "n2", SubType: "174", InstID: "ETH-USDT-SWAP", InstType: "SWAP", Ccy: "USDT", TsMs: nowMs + 1, BalChg: 2}}
	src := fakeSrc{pages: map[string][]Bill{"": page1, "n1": page2}}
	ing := NewIngester(pool, src)
	if n, _ := ing.Ingest(ctx); n != 1 {
		t.Fatalf("页1 应记 1 got=%d", n)
	}
	// 第二轮：新页1=n2 且尾行已入库（<100 行早停亦可）——增量零重。
	src.pages[""] = page2
	if n, _ := ing.Ingest(ctx); n != 1 {
		t.Fatalf("n2 应新记 1 got=%d", n)
	}
	if n, _ := ing.Ingest(ctx); n != 0 {
		t.Fatalf("重放应 0 got=%d", n)
	}
}
