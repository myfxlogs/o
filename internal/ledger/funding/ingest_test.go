package funding

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
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
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
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
			Ccy: "USDT", TsMs: nowMs + 3, BalChg: 50, Fee: 0, Raw: []byte(`{}`)}, // 未映射子型≠0→残差守恒落 unmapped_bill
		{BillID: "bdust", SubType: "174", InstID: "DOGE-USDT-SWAP", InstType: "SWAP",
			Ccy: "USDT", TsMs: nowMs + 4, BalChg: 0.001, Raw: []byte(`{}`)}, // 无归仓→position_ref NULL
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	// 升件#14671：残差守恒后 bfee(balChg-100.01-fee-0.01=残-100)与
	// bskip(无费 balChg 50 未映射)各产 unmapped_bill 行——静默零覆盖
	// 旧语义正是审计所销病灶，断言随守恒律更新。
	if err != nil || n != 6 {
		t.Fatalf("首拉应记 6 单（174/173/fee+残/dust/残）got=%d err=%v", n, err)
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
		{"unmapped_bill", -100, true},
		{"unmapped_bill", 50, true},
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
	// 未映射子型 balChg≠0：残差守恒落 unmapped_bill（adjustment 待归因）——
	// 「只留源行不入账」旧语义=升件#14671 所销病灶本体。
	var cnt int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_cashflow c JOIN fin_venue_bill b ON c.source_ref=b.id WHERE b.bill_id='bskip' AND c.kind='unmapped_bill'").Scan(&cnt); err != nil || cnt != 1 {
		t.Fatalf("bskip 应落 unmapped_bill 1 行 cnt=%d", cnt)
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

// O-33362 取证锚：/account/bills subType 160-163=type=6 保证金子型
// （/account/subtypes 权威典+真户实证——非双币赢流水）；原 earn_* 映射
// 系误配已摘除——未映射子型落 unmapped_bill 残差域待归因（真 DCD 现金腿
// 在 /asset/bills type 160-163 由 dcdrecon 簿记环按订单枚举直写，本面
// 不重复入账防双计）。
func TestIngestMarginSubTypesResidual(t *testing.T) {
	ctx, pool := setup(t)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "d160", SubType: "160", InstType: "ASSET", Ccy: "USDT", TsMs: nowMs, BalChg: -100, Raw: []byte(`{}`)},
		{BillID: "d161", SubType: "161", InstType: "ASSET", Ccy: "USDT", TsMs: nowMs + 1, BalChg: 100, Raw: []byte(`{}`)},
		{BillID: "d162", SubType: "162", InstType: "ASSET", Ccy: "USDT", TsMs: nowMs + 2, BalChg: 2.5, Raw: []byte(`{}`)},
		{BillID: "d163", SubType: "163", InstType: "ASSET", Ccy: "USDT", TsMs: nowMs + 3, BalChg: 0.5, Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Fatalf("保证金四行应残差落账: %d", n)
	}
	var unmapped, earnOut, earnIn float64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='unmapped_bill'").Scan(&unmapped)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='earn_out'").Scan(&earnOut)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='earn_in'").Scan(&earnIn)
	if unmapped != 3.0 || earnOut != 0 || earnIn != 0 {
		t.Fatalf("unmapped=%v earn_out=%v earn_in=%v 应 3.0/0/0（160-163 非 dcd 不产 earn_*）", unmapped, earnOut, earnIn)
	}
}

// O-33362 取证锚：业主域 dcd 现金腿不入策略账——/account/bills 面已无
// dcd 映射（真腿=/asset/bills 由订单枚举径处理，业主域单不产 counted 腿）；
// 命中域册的保证金子型行同落 unmapped_bill 残差域，owner_in/owner_out
// 不再由本面产出（T-0364 归因径随 bills 误映射一并摘除）。
func TestIngestMarginOwnerDomainResidual(t *testing.T) {
	ctx, pool := setup(t)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "o161", SubType: "161", InstID: "BTC-USDT-261225-95000-C", InstType: "ASSET",
			Ccy: "USDT", TsMs: nowMs, BalChg: 1966.5, Raw: []byte(`{}`)},
		{BillID: "o162", SubType: "162", InstType: "ASSET",
			Ccy: "USDT", TsMs: nowMs + 1, BalChg: 40, Raw: []byte(`{"ordId":"11448262"}`)},
		{BillID: "o160", SubType: "160", InstID: "BTC-USDT-261225-95000-C", InstType: "ASSET",
			Ccy: "USDT", TsMs: nowMs + 2, BalChg: -1966.5, Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil || n != 3 {
		t.Fatalf("业主域三行应残差落账 n=%d err=%v", n, err)
	}
	var unmapped, ownerIn, ownerOut float64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='unmapped_bill'").Scan(&unmapped)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='owner_in'").Scan(&ownerIn)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE kind='owner_out'").Scan(&ownerOut)
	if unmapped != 40.0 || ownerIn != 0 || ownerOut != 0 {
		t.Fatalf("unmapped=%v owner_in=%v owner_out=%v 应 40.0/0/0", unmapped, ownerIn, ownerOut)
	}
}

// O-34287 验收④：SPOT 原生币腿按本币入账——BTC 基币行 subType1=credit
// 记 spot_buy BTC（gross=balChg−fee）、fee 腿按扣款币另记；USDT 报价侧
// 行 spotLike 非 native 仍走旧径零腿（quote 侧现金流=legCash 径不重复）。
// position_ref=NULL——基币量不进仓结算求和域（USDT/USD 报价币域）。
func TestIngestSpotNativeLegs(t *testing.T) {
	ctx, pool := setup(t)
	nowMs := time.Now().UnixMilli()
	bills := []Bill{
		{BillID: "nb1", SubType: "1", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "BTC", TsMs: nowMs, BalChg: 0.001, Fee: -0.00001, Raw: []byte(`{}`)},
		{BillID: "nb2", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "BTC", TsMs: nowMs + 1, BalChg: -0.002, Fee: -0.00002, Raw: []byte(`{}`)},
		{BillID: "nb3", SubType: "2", InstID: "BTC-USDT", InstType: "SPOT",
			Ccy: "USDT", TsMs: nowMs + 2, BalChg: -100.5, Fee: -0.1, Raw: []byte(`{}`)},
	}
	ing := NewIngester(pool, fakeSrc{pages: map[string][]Bill{"": bills}})
	n, err := ing.Ingest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 { // BTC 两行各 spot 腿+fee 腿=4，USDT 行零腿
		t.Fatalf("原生币腿应 4 行入账: %d", n)
	}
	var buy, sell, feeBTC, feeUSDT float64
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE ccy='BTC' AND kind='spot_buy'").Scan(&buy)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE ccy='BTC' AND kind='spot_sell'").Scan(&sell)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE ccy='BTC' AND kind='fee'").Scan(&feeBTC)
	_ = pool.QueryRow(ctx, "SELECT COALESCE(sum(amount),0) FROM fin_cashflow WHERE ccy='USDT' AND kind='fee'").Scan(&feeUSDT)
	// gross=balChg−fee：买 0.001−(−0.00001)=0.00101；卖 −0.002−(−0.00002)=−0.00198
	if buy != 0.00101 || sell != -0.00198 {
		t.Fatalf("BTC spot 腿 buy=%v sell=%v 应 0.00101/-0.00198", buy, sell)
	}
	if feeBTC != -0.00003 || feeUSDT != 0 {
		t.Fatalf("fee 腿 BTC=%v USDT=%v 应 -0.00003/0（USDT 侧 spotLike 旧径零腿）", feeBTC, feeUSDT)
	}
	var posRef *int64
	_ = pool.QueryRow(ctx, "SELECT position_ref FROM fin_cashflow WHERE ccy='BTC' LIMIT 1").Scan(&posRef)
	if posRef != nil {
		t.Fatalf("原生币腿 position_ref 应 NULL（币量不混报价币结算域）: %v", *posRef)
	}
}
