// probe_test.go —— T-0432⑤ 漂移探针桩测（fixture 同 onboard_test——真 PG
// 独立 schema）。锚：①指纹漂移→ops_event 升件行（esc_class='venue漂移'）；
// ②同签名去重（同态重拍零重发——事件簿防洪）；③死价探活断升件；④无映射
// 行=零面零升件（映射=确认件前置）；⑤ctVal 变=漂移面（单位错配病重演防）。
package onboard

import (
	"context"
	"testing"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
)

// fakeProbeSess 探针会话桩（指纹面可控——drift 场景构造；hash/ctVal 标量
// 出缝同产线形——proto 型不出缝 D-514 同律）。
type fakeProbeSess struct {
	si   *mt5pb.SymbolInfo
	sg   *mt5pb.SymGroup
	bid  float64
	conn error
}

func (f *fakeProbeSess) Connect(context.Context, mt5.Credentials) error { return f.conn }
func (f *fakeProbeSess) SpecHashOf(_ context.Context, _ string) (string, float64, error) {
	return mt5.SpecHash(f.si, f.sg), f.si.GetContractSize(), nil
}
func (f *fakeProbeSess) GetQuoteOne(_ context.Context, _ string) (mt5.QuoteSnapshot, error) {
	return mt5.QuoteSnapshot{Bid: f.bid, Ask: f.bid + 1}, nil
}
func (f *fakeProbeSess) Disconnect(context.Context) {}
func (f *fakeProbeSess) Close()                     {}

// baseSpec 基线语义件（ContractSize=1——与落行基线 hash 同源）。
func baseSpec() (*mt5pb.SymbolInfo, *mt5pb.SymGroup) {
	return &mt5pb.SymbolInfo{ContractSize: 1, Digits: 2, Points: 0.01, ProfitCurrency: "USD"},
		&mt5pb.SymGroup{MinLots: 0.01, LotsStep: 0.01, TradeMode: mt5pb.TradeMode_TradeMode_FullAccess}
}

// probeFixture 探针拍试验台：runner+已落方言行基线+探针缝注假。
func probeFixture(t *testing.T) (*Runner, context.Context, *mt5.DialectRow) {
	t.Helper()
	r, pool, _ := fixture(t)
	ctx := context.Background()
	si, sg := baseSpec()
	store := mt5.NewDialectStore(pool)
	row := mt5.DialectRow{Venue: "exness", Account: "3001157", Canonical: "BTC-USDT-SWAP",
		VenueSymbol: "BTCUSDm", CtVal: 1, TickSz: 0.01, MinSz: 0.01, LotStep: 0.01,
		LotUnit: "lots", Digits: 2, TradeMode: 4, SpecHash: mt5.SpecHash(si, sg)}
	if err := store.UpsertVerified(ctx, row, nil); err != nil {
		t.Fatal(err)
	}
	return r, ctx, &row
}

// ①指纹漂移→升件+②同签名去重。
func TestProbeDriftEscalatesAndDedups(t *testing.T) {
	r, ctx, row := probeFixture(t)
	// venue 改面：ContractSize 1→0.01（ctVal 变=单位错配病重演——指纹漂移面）。
	si, sg := baseSpec()
	si.ContractSize = 0.01
	sess := &fakeProbeSess{si: si, sg: sg, bid: 90000}
	r.dialProbeFn = func(string) (probeSession, error) { return sess, nil }
	oldPace := probePace
	probePace = 0
	defer func() { probePace = oldPace }()

	fam := credFamily{venue: "exness", fields: map[string]string{"login": "3001157"}}
	if err := r.probeVenue(ctx, fam); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.fin.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='方言漂移'
		AND esc_scope='exness:3001157'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("漂移升件行=%d——红桩⑤破", n)
	}
	// ② 同签名重拍=去重零重发。
	if err := r.probeVenue(ctx, fam); err != nil {
		t.Fatal(err)
	}
	if err := r.fin.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='方言漂移'
		AND esc_scope='exness:3001157'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("同签名重发 %d 行——防洪破", n)
	}
	_ = row
}

// ③死价探活断→升件（报价径断面）。
func TestProbeQuoteDeadEscalates(t *testing.T) {
	r, ctx, _ := probeFixture(t)
	si, sg := baseSpec() // 指纹同——纯死价面
	sess := &fakeProbeSess{si: si, sg: sg, bid: 0}
	r.dialProbeFn = func(string) (probeSession, error) { return sess, nil }
	oldPace := probePace
	probePace = 0
	defer func() { probePace = oldPace }()

	fam := credFamily{venue: "exness", fields: map[string]string{"login": "3001157"}}
	if err := r.probeVenue(ctx, fam); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := r.fin.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='方言探活断'
		AND esc_scope='exness:3001157'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("探活断行=%d", n)
	}
}

// ④无映射行=零面（映射=确认件——向导第④步前置，探针空转零升件）。
func TestProbeNoRowsNoop(t *testing.T) {
	r, ctx := func() (*Runner, context.Context) {
		r, _, _ := fixture(t)
		return r, context.Background()
	}()
	calls := 0
	r.dialProbeFn = func(string) (probeSession, error) { calls++; return nil, nil }
	oldPace := probePace
	probePace = 0
	defer func() { probePace = oldPace }()
	fam := credFamily{venue: "exness", fields: map[string]string{"login": "3001157"}}
	if err := r.probeVenue(ctx, fam); err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("无映射行拨号 %d 次——前置判破", calls)
	}
	var n int
	if err := r.fin.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE esc_class='venue漂移'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("无面升件 %d 行", n)
	}
}

// ⑤ SpecHash 判别面：同 spec 同 hash，变任一语义字段变 hash（ctVal/digits/
// TradeMode——漂移探针对拍原点）。
func TestSpecHashDiscriminates(t *testing.T) {
	si, sg := baseSpec()
	base := mt5.SpecHash(si, sg)
	si2, sg2 := baseSpec()
	if mt5.SpecHash(si2, sg2) != base {
		t.Fatal("同 spec 指纹漂移——基线不稳")
	}
	si3, sg3 := baseSpec()
	si3.ContractSize = 100 // ctVal 变
	if mt5.SpecHash(si3, sg3) == base {
		t.Fatal("ctVal 变指纹不变——单位错配病防破")
	}
	si4, sg4 := baseSpec()
	sg4.TradeMode = mt5pb.TradeMode_TradeMode_CloseOnly
	if mt5.SpecHash(si4, sg4) == base {
		t.Fatal("TradeMode 变指纹不变")
	}
}
