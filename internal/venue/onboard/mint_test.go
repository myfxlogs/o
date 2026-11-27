// mint_test.go —— O-41378 桩测锚：①classifyFamily 族分类谱；②pairCensus
// Registry 配对纯函数（命中/未注册/歧义/族不配）；③censusParamCheck 参数
// 对拍；④普查→mint 全链（假会话喂真形 SpecFacts/Quote——census 落行+
// dialect 落行+flag 载明+self-heal）。
package onboard

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
)

// fakeCensusSession 桩普查/mint 会话（SpecFacts/Quote 表驱动——禁桩真 venue）。
type fakeCensusSession struct {
	syms     []string
	facts    map[string]mt5.SpecFacts
	factErr  map[string]error
	quotes   map[string]mt5.QuoteSnapshot
	quoteErr map[string]error
	listCall int
}

func (f *fakeCensusSession) Connect(context.Context, mt5.Credentials) error { return nil }
func (f *fakeCensusSession) SymbolList(context.Context) ([]string, error) {
	f.listCall++
	return f.syms, nil
}
func (f *fakeCensusSession) SymbolSpecFacts(_ context.Context, sym string) (mt5.SpecFacts, error) {
	if err := f.factErr[sym]; err != nil {
		return mt5.SpecFacts{}, err
	}
	ft, ok := f.facts[sym]
	if !ok {
		return mt5.SpecFacts{}, errFakeNoSym
	}
	return ft, nil
}
func (f *fakeCensusSession) GetQuoteOne(_ context.Context, sym string) (mt5.QuoteSnapshot, error) {
	if err := f.quoteErr[sym]; err != nil {
		return mt5.QuoteSnapshot{}, err
	}
	q, ok := f.quotes[sym]
	if !ok {
		return mt5.QuoteSnapshot{Symbol: sym, Bid: 100, Ask: 100.5}, nil // 缺省活价
	}
	return q, nil
}
func (f *fakeCensusSession) Disconnect(context.Context) {}
func (f *fakeCensusSession) Close()                     {}

var errFakeNoSym = errString("venue 未认")

type errString string

func (e errString) Error() string { return string(e) }

func factsOf(sym string, ctVal float64, digits, tm int32, pc, hash string) mt5.SpecFacts {
	return mt5.SpecFacts{Symbol: sym, CtVal: ctVal, TickSz: 0.01, MinSz: 0.01,
		LotStep: 0.01, Digits: digits, TradeMode: tm, ProfitCcy: pc,
		MarginCcy: "USD", SpecHash: hash}
}

// 锚① family_guess 分类谱（参数分拣——fx 双法币/metal 前缀/crypto 宇宙
// base/期约形/index·energy 名形/other 兜底）。
func TestClassifyFamily(t *testing.T) {
	crypto := map[string]bool{"BTC": true, "ETH": true, "DOGE": true}
	cases := []struct {
		base, suffix, want string
	}{
		{"GOLD", "", "metal"},     // GOLD#→base GOLD 单名形
		{"GOLD24-7", "", "metal"}, // GOLD24-7#→前缀 GOLD 族（不误配 GOLD 靠别名层）
		{"GAU", "usd", "metal"},   // GAUUSD#→base GAU
		{"SILVER", "", "metal"},
		{"EUR", "usd", "fx"},     // EURUSD#→EUR/USD 双法币
		{"EUR", "usdm", "fx"},    // EURUSDm→m尾剥后 USD 仍 fx
		{"USD", "jpy", "fx"},     // USDJPY#
		{"BTC", "usd", "crypto"}, // BTCUSD#→crypto（非 fx——BTC 非法币码）
		{"ETH", "btc", "crypto"}, // ETHBTC#→crypto 族（计价币对拍归 mint 判）
		{"OIL-JAN27", "", "future_cfd"},
		{"US100CASH", "", "index"}, // US100Cash#→剥 # 后 base
		{"XAU", "usd", "metal"},    // XAUUSD→金别名族
		{"XXX", "", "other"},       // 判不出=other 不猜
	}
	for _, c := range cases {
		if got := classifyFamily(c.base, c.suffix, crypto); got != c.want {
			t.Fatalf("classifyFamily(%q,%q)=%q want %q", c.base, c.suffix, got, c.want)
		}
	}
}

// 锚② pairCensusRegistry 配对纯函数（别名键三型/未注册/歧义/族不配）。
func TestPairCensusRegistry(t *testing.T) {
	reg := []mt5.RegistryRow{
		{Canonical: "GOLD", Family: "metal", CtVal: 100, Aliases: []string{"GOLD", "XAU", "XAUUSD"}},
		{Canonical: "EURUSD", Family: "fx", CtVal: 100000, Aliases: []string{"EURUSD"}},
		{Canonical: "BTCUSD", Family: "crypto", CtVal: 1, Aliases: []string{"BTCUSD", "BTC"}},
	}
	idx := registryAliasIndex(reg)
	f64 := func(v float64) *float64 { return &v }
	i32 := func(v int32) *int32 { return &v }
	str := func(v string) *string { return &v }

	// GOLD#→base GOLD 命中；XAUUSDm→base XAU suffix usdm→m剥键命中；
	// EURUSD#→pairKey EURUSD 命中；EURGBP#→pairKey EURGBP 未注册 flag；
	// BTCUSD#→BTC/BTCUSD 双键同 canonical 不歧义。
	ok := []censusMintRow{
		{VenueSymbol: "GOLD#", BaseGuess: "GOLD", SuffixGuess: "", FamilyGuess: "metal",
			CtVal: f64(100), Digits: i32(2), TradeMode: i32(4), ProfitCcy: str("USD"), SpecHash: str("h")},
		{VenueSymbol: "XAUUSDm", BaseGuess: "XAU", SuffixGuess: "usdm", FamilyGuess: "metal",
			CtVal: f64(100), Digits: i32(2), TradeMode: i32(4), ProfitCcy: str("USD"), SpecHash: str("h")},
		{VenueSymbol: "EURUSD#", BaseGuess: "EUR", SuffixGuess: "usd", FamilyGuess: "fx",
			CtVal: f64(100000), Digits: i32(5), TradeMode: i32(4), ProfitCcy: str("USD"), SpecHash: str("h")},
		{VenueSymbol: "BTCUSD#", BaseGuess: "BTC", SuffixGuess: "usd", FamilyGuess: "crypto",
			CtVal: f64(1), Digits: i32(2), TradeMode: i32(4), ProfitCcy: str("USD"), SpecHash: str("h")},
	}
	for _, m := range ok {
		row, reason := pairCensusRegistry(m, idx)
		if row == nil {
			t.Fatalf("%s 应命中 got reason=%s", m.VenueSymbol, reason)
		}
		if reason := censusParamCheck(m, *row); reason != "" {
			t.Fatalf("%s 参数对拍应过 got %s", m.VenueSymbol, reason)
		}
	}
	// 负例：GOLD24-7#（base GOLD24-7 不命中 GOLD 别名）/EURGBP#（pairKey
	// EURGBP 无登记——base EUR 不在 EURUSD aliases 故不误配）/族不配。
	neg := []struct {
		m      censusMintRow
		wantIn string
	}{
		{censusMintRow{VenueSymbol: "GOLD24-7#", BaseGuess: "GOLD24-7", FamilyGuess: "metal"}, "unregistered"},
		{censusMintRow{VenueSymbol: "EURGBP#", BaseGuess: "EUR", SuffixGuess: "gbp", FamilyGuess: "fx"}, "unregistered"},
		{censusMintRow{VenueSymbol: "BTCUSD#", BaseGuess: "BTC", SuffixGuess: "usd", FamilyGuess: "fx"}, "family_mismatch"},
	}
	for _, c := range neg {
		row, reason := pairCensusRegistry(c.m, idx)
		if row != nil || !contains(reason, c.wantIn) {
			t.Fatalf("%s 应 flag[%s] got row=%v reason=%s", c.m.VenueSymbol, c.wantIn, row, reason)
		}
	}
	// 歧义：两 canonical 共享别名→alias_ambiguous 不猜。
	amb := registryAliasIndex([]mt5.RegistryRow{
		{Canonical: "A1", Family: "metal", Aliases: []string{"XAU"}},
		{Canonical: "A2", Family: "metal", Aliases: []string{"XAU"}},
	})
	if row, reason := pairCensusRegistry(censusMintRow{BaseGuess: "XAU"}, amb); row != nil || !contains(reason, "alias_ambiguous") {
		t.Fatalf("歧义应 flag got row=%v reason=%s", row, reason)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// 锚③ censusParamCheck 逐判据（ctVal/计价币/digits/trade_mode/spec_hash）。
func TestCensusParamCheck(t *testing.T) {
	reg := mt5.RegistryRow{Canonical: "GOLD", CtVal: 100}
	f64 := func(v float64) *float64 { return &v }
	i32 := func(v int32) *int32 { return &v }
	str := func(v string) *string { return &v }
	good := censusMintRow{CtVal: f64(100), Digits: i32(2), TradeMode: i32(4),
		ProfitCcy: str("USD"), SpecHash: str("h")}
	if r := censusParamCheck(good, reg); r != "" {
		t.Fatalf("全过应空 got %s", r)
	}
	cases := []struct {
		mut  func(*censusMintRow)
		want string
	}{
		{func(m *censusMintRow) { m.CtVal = f64(0.001) }, "ctval_mismatch"}, // XAU-USDT-SWAP 微型张同名不同券判据锚
		{func(m *censusMintRow) { m.ProfitCcy = str("JPY") }, "profit_ccy_reject"},
		{func(m *censusMintRow) { m.Digits = i32(0) }, "digits_invalid"},
		{func(m *censusMintRow) { m.TradeMode = i32(3) }, "trade_mode_reject"},
		{func(m *censusMintRow) { m.SpecHash = nil }, "params_missing"},
		{func(m *censusMintRow) { m.CtVal = nil }, "params_missing"},
	}
	for _, c := range cases {
		m := good
		c.mut(&m)
		if r := censusParamCheck(m, reg); !contains(r, c.want) {
			t.Fatalf("want %s got %s", c.want, r)
		}
	}
}

// 锚④ 普查→mint 全链（假会话：5 符——3 命中落行+2 未注册 flag；
// census 节流二跑零重拉；flagged→registry 补登记自愈转 mint）。
func TestCensusMintFullChain(t *testing.T) {
	r, pool, _ := fixture(t)
	ctx := context.Background()
	broker := credFamily{venue: "xmtest", fields: map[string]string{
		"login": "12345", "password": "p", "server": "srv"}}
	r.dialCensusFn = func(string) (censusSession, error) {
		return &fakeCensusSession{
			syms: []string{"GOLD#", "XAUUSDm", "EURUSD#", "BTCUSD#", "GOLD24-7#", "XXX#"},
			facts: map[string]mt5.SpecFacts{
				"GOLD#":     factsOf("GOLD#", 100, 2, 4, "USD", "h-gold"),
				"XAUUSDm":   factsOf("XAUUSDm", 100, 2, 4, "USD", "h-xaum"),
				"EURUSD#":   factsOf("EURUSD#", 100000, 5, 4, "USD", "h-eur"),
				"BTCUSD#":   factsOf("BTCUSD#", 1, 2, 4, "USD", "h-btc"),
				"GOLD24-7#": factsOf("GOLD24-7#", 100, 2, 4, "USD", "h-g247"),
				"XXX#":      factsOf("XXX#", 1, 2, 4, "USD", "h-xxx"),
			},
		}, nil
	}
	bases := map[string]bool{"BTC": true}
	if err := r.censusVenue(ctx, broker, bases); err != nil {
		t.Fatal(err)
	}
	// census 落行断言：6 行+family_guess 分拣+params 全谱。
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_venue_symbols
		WHERE venue='xmtest' AND account='12345'`).Scan(&n); err != nil || n != 6 {
		t.Fatalf("census 行数=%d err=%v want 6", n, err)
	}
	var fam string
	pool.QueryRow(ctx, `SELECT family_guess FROM fin_venue_symbols
		WHERE venue='xmtest' AND venue_symbol='GOLD#'`).Scan(&fam)
	if fam != "metal" {
		t.Fatalf("GOLD# family=%s want metal", fam)
	}
	pool.QueryRow(ctx, `SELECT family_guess FROM fin_venue_symbols
		WHERE venue='xmtest' AND venue_symbol='BTCUSD#'`).Scan(&fam)
	if fam != "crypto" {
		t.Fatalf("BTCUSD# family=%s want crypto", fam)
	}
	pool.QueryRow(ctx, `SELECT family_guess FROM fin_venue_symbols
		WHERE venue='xmtest' AND venue_symbol='EURUSD#'`).Scan(&fam)
	if fam != "fx" {
		t.Fatalf("EURUSD# family=%s want fx", fam)
	}
	// 节流锚：二跑 censusFresh=true（censusEvery 内不重拉）。
	fresh, err := r.censusFresh(ctx, broker)
	if err != nil || !fresh {
		t.Fatalf("censusFresh=%v err=%v want true", fresh, err)
	}
	// 撞键锚（O-41379 件②）：BTCUSD# 旧恒等行预置——mint 落行
	// canonical=venue_symbol 同键 ON CONFLICT 幂等 update 不炸不增行。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_symbol_dialect
		(venue,account,canonical,venue_symbol,ct_val,tick_sz,min_sz,lot_step,
		 lot_unit,digits,trade_mode,spec_hash,quote_snapshot,verified_at)
		VALUES('xmtest','12345','BTCUSD#','BTCUSD#',1,0.01,0.01,0.01,'lots',
		 2,4,'h-btc','{}',now())
		ON CONFLICT (venue,account,canonical) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	// mint 拍：registry 种子在案（GOLD/GAUUSD/SILVER/BTCUSD/ETHUSD/EURUSD/
	// USDJPY/GBPUSD——fin_0274 迁移落测试库）。
	idx := mintTestRegIndex(t, ctx, pool)
	if err := r.mintVenue(ctx, broker, idx); err != nil {
		t.Fatal(err)
	}
	// dialect 落行=4（GOLD#/XAUUSDm/EURUSD#/BTCUSD#——GOLD24-7#/XXX# flag）。
	var dn int
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_symbol_dialect
		WHERE venue='xmtest' AND account='12345'`).Scan(&dn)
	if dn != 4 {
		t.Fatalf("dialect 行数=%d want 4", dn)
	}
	// O-41379 恒等式锚：全谱 canonical=venue_symbol（策略 inst_id=venue
	// 原符唯一活命名空间——净名行对 inst_id 永不可达）；registry 命中族
	// 留 census.canonical_ref 作证物位。
	var identBad int
	pool.QueryRow(ctx, `SELECT count(*) FROM fin_symbol_dialect
		WHERE venue='xmtest' AND canonical<>venue_symbol`).Scan(&identBad)
	if identBad != 0 {
		t.Fatalf("恒等式破：canonical<>venue_symbol 行数=%d", identBad)
	}
	var cref string
	pool.QueryRow(ctx, `SELECT canonical_ref FROM fin_venue_symbols
		WHERE venue='xmtest' AND venue_symbol='XAUUSDm'`).Scan(&cref)
	if cref != "GOLD" {
		t.Fatalf("XAUUSDm canonical_ref=%s want GOLD（registry 命中族证物位）", cref)
	}
	// census map_state：4 auto_minted+2 flagged（map_reason 全载明）。
	states := map[string]string{}
	reasons := map[string]string{}
	rows, err := pool.Query(ctx, `SELECT venue_symbol,map_state,map_reason
		FROM fin_venue_symbols WHERE venue='xmtest'`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var s, st string
		var rsn *string
		if err := rows.Scan(&s, &st, &rsn); err != nil {
			t.Fatal(err)
		}
		states[s] = st
		if rsn != nil {
			reasons[s] = *rsn
		}
	}
	rows.Close()
	for _, s := range []string{"GOLD#", "XAUUSDm", "EURUSD#", "BTCUSD#"} {
		if states[s] != "auto_minted" {
			t.Fatalf("%s map_state=%s want auto_minted", s, states[s])
		}
	}
	for _, s := range []string{"GOLD24-7#", "XXX#"} {
		if states[s] != "flagged" || reasons[s] == "" {
			t.Fatalf("%s 应 flagged+reason got state=%s reason=%q", s, states[s], reasons[s])
		}
	}
	// 审计行 actor=auto-mint。
	var actor string
	pool.QueryRow(ctx, `SELECT seat FROM ops_event WHERE event='venue符号映射'
		AND esc_scope='xmtest:12345' ORDER BY id DESC LIMIT 1`).Scan(&actor)
	if actor != "auto-mint" {
		t.Fatalf("审计 actor=%s want auto-mint", actor)
	}
	// 自愈锚：registry 补登记 XXX 族→下拍 flagged 转 mint。
	if _, err := pool.Exec(ctx, `INSERT INTO fin_canonical_registry
		(canonical,family,ct_val,quote_ccy,aliases,note)
		VALUES('XXXCOIN','crypto',1,'USD',ARRAY['XXXCOIN','XXX'],'test 补登记')
		ON CONFLICT (canonical) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	// XXX# census 行 base=XXX 命中别名 XXX——family_guess=other 兼容
	// （other 不收窄）→ctVal 1==1→minted。
	if err := r.mintVenue(ctx, broker, mintTestRegIndex(t, ctx, pool)); err != nil {
		t.Fatal(err)
	}
	var xxxState string
	pool.QueryRow(ctx, `SELECT map_state FROM fin_venue_symbols
		WHERE venue='xmtest' AND venue_symbol='XXX#'`).Scan(&xxxState)
	if xxxState != "auto_minted" {
		t.Fatalf("XXX# 自愈后 map_state=%s want auto_minted", xxxState)
	}
}

// mintTestRegIndex 测试径 registry 全载建别名索引（RegistryStore.Enabled
// 同径——种子+用例补登全收）。
func mintTestRegIndex(t *testing.T, ctx context.Context, pool *pgxpool.Pool) map[string][]mt5.RegistryRow {
	t.Helper()
	reg, err := mt5.NewRegistryStore(pool).Enabled(ctx)
	if err != nil {
		t.Fatalf("registry Enabled: %v", err)
	}
	return registryAliasIndex(reg)
}
