// dialect_face_test.go —— T-0432 方言对表面桩测（范式同 broker_face_test：
// mux HTTP 径+假会话注缝——禁桩真 venue 同律）。锚：①候选机造排序（base
// 滤+score 序+venue 未认弃）；②硬验拒收红桩（ctVal 错配/TradeMode 非
// FullAccess/死价——零落表）；③全环（官名→硬验→落行+审计行单事务）；
// ④官名清单读面。
package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	dashboardv1 "o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// fakeDxSession 方言面会话桩（真形回包喂缝——禁桩真 venue）。
type fakeDxSession struct {
	connErr error
	syms    []string
	specs   map[string]*dxSpec
	quotes  map[string]mt5.QuoteSnapshot
}

type dxSpec struct {
	si *mt5pb.SymbolInfo
	sg *mt5pb.SymGroup
}

func (f *fakeDxSession) Connect(_ context.Context, _ mt5.Credentials) error { return f.connErr }
func (f *fakeDxSession) SymbolList(context.Context) ([]string, error) {
	if f.connErr != nil {
		return nil, f.connErr
	}
	return f.syms, nil
}
func (f *fakeDxSession) SymbolSpec(_ context.Context, symbol string) (*mt5pb.SymbolInfo, *mt5pb.SymGroup, error) {
	sp, ok := f.specs[symbol]
	if !ok {
		return nil, nil, errors.New("Symbol not found（venue 实拍形）")
	}
	return sp.si, sp.sg, nil
}
func (f *fakeDxSession) GetQuoteOne(_ context.Context, symbol string) (mt5.QuoteSnapshot, error) {
	q, ok := f.quotes[symbol]
	if !ok {
		return mt5.QuoteSnapshot{}, errors.New("quote 缺")
	}
	return q, nil
}
func (f *fakeDxSession) Disconnect(context.Context) {}
func (f *fakeDxSession) Close()                     {}

// dxBTC 真形 BTC 语义件（ContractSize=1/ProfitCurrency=USD/FullAccess——
// Exness BTCUSDm 实拍形 T-0143）。
func dxBTC(contractSize float64) *dxSpec {
	return &dxSpec{
		si: &mt5pb.SymbolInfo{ContractSize: contractSize, Digits: 2, Points: 0.01,
			TickSize: 0, Spread: 120, ProfitCurrency: "USD", MarginCurrency: "USD"},
		sg: &mt5pb.SymGroup{MinLots: 0.01, LotsStep: 0.01, TradeMode: mt5pb.TradeMode_TradeMode_FullAccess},
	}
}

// dialectMux 方言面 mux（dialDialect/dialectMeta 注缝——config 种子 gateway_addr）。
func dialectMux(t *testing.T, dial func(string) (dialectSession, error),
	meta dialectMeta) (*config.Client, *pgxpool.Pool, context.Context, http.Handler) {
	t.Helper()
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool, "fin"); err != nil {
		t.Fatal(err)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('mt5.gateway_addr','"mt5grpc3.mtapi.io:443"')
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, coreService{
		pool: pool, cfg: cfg, dialDialect: dial, dialectMeta: meta})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	return cfg, pool, ctx, mux
}

// dxPost RPC POST（Origin 同源真形——同 broker subPost 律）。
func dxPost(t *testing.T, mux http.Handler, proc, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, proc, strings.NewReader(body))
	req.Host = "ui.local"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://ui.local")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

const (
	dxProcCandidates = "/o.dashboard.v1.CoreService/ListSymbolCandidates"
	dxProcMapping    = "/o.dashboard.v1.CoreService/SubmitSymbolMapping"
	dxProcUniverse   = "/o.dashboard.v1.CoreService/ListCanonicalUniverse"
)

// seedDxCreds 凭据族种（<venue>.mt5.* 三件密文——mt5CredsOf 读解正径）。
func seedDxCreds(t *testing.T, ctx context.Context, pool *pgxpool.Pool, mk, venue, login string) {
	t.Helper()
	for f, v := range map[string]string{"login": login, "password": "pw-桩", "server": "Exness-MT5Trial9"} {
		ct, err := cryptoutil.Seal([]byte(mk), v)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO app_secrets(key,ciphertext) VALUES($1,$2)
			ON CONFLICT (key) DO UPDATE SET ciphertext=EXCLUDED.ciphertext`, venue+".mt5."+f, ct); err != nil {
			t.Fatal(err)
		}
	}
}

// ①候选机造：base 滤（BTC* 命中进面）+score 序（quote 字面同>族同>perp 型）
// +venue 未认弃（params 缺不入面）+scanned 全量如实。
func TestRankCandidates(t *testing.T) {
	oldPace := dialectPace
	dialectPace = 0 // 桩测零速
	defer func() { dialectPace = oldPace }()
	sess := &fakeDxSession{
		syms: []string{"BTCUSDm", "BTCUSD.pro", "ETHUSDm", "BTCUSDTm", "GOLD", "XAUUSDm"},
		specs: map[string]*dxSpec{
			"BTCUSDm":    dxBTC(1),
			"BTCUSD.pro": dxBTC(1),
			"BTCUSDTm":   dxBTC(1),
			"ETHUSDm":    dxBTC(1), // base 不中——不入面
		},
	}
	c := coreService{}
	rows, scanned := c.rankCandidates(context.Background(), sess, sess.syms, "BTC", "USDT", 0)
	if scanned != 6 {
		t.Fatalf("scanned=%d——全目录枚举面如实破", scanned)
	}
	if len(rows) != 3 {
		t.Fatalf("rows=%d——base 滤破（BTC* 三符应入）: %v", len(rows), rows)
	}
	if rows[0].GetVenueSymbol() != "BTCUSDTm" {
		t.Fatalf("首行=%s——quote 字面同应最高分", rows[0].GetVenueSymbol())
	}
	for _, r := range rows {
		if r.GetContractSize() != 1 || r.GetMinSz() != 0.01 || r.GetTradeMode() != 4 {
			t.Fatalf("params 面 venue 实返破: %+v", r)
		}
	}
	// 未认符（specs 缺）=弃——GOLD 无 spec 无行。
	if rows[len(rows)-1].GetVenueSymbol() == "GOLD" {
		t.Fatal("venue 未认符入面——禁猜名判据破")
	}
}

// ②硬验拒收红桩（断言红桩）：语义不符候选=拒收非警告零落表——ctVal 错配
// （单位错配病防线）/TradeMode 非 FullAccess/死价探活三面。
func TestSubmitSymbolMappingHardVerifyReject(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{
				syms:   []string{"BTCUSDm"},
				specs:  map[string]*dxSpec{"BTCUSDm": dxBTC(0.01)}, // ctVal 错配（官名 ctVal=1）
				quotes: map[string]mt5.QuoteSnapshot{"BTCUSDm": {Bid: 90000, Ask: 90001}},
			}, nil
		},
		func(context.Context, string) (float64, string, string, error) {
			return 1, "BTC", "live", nil // 官名语义实返形
		})
	seedDxCreds(t, ctx, pool, mk, "exness", "3001157")
	rec := dxPost(t, mux, dxProcMapping,
		`{"venue":"exness","canonical":"BTC-USDT-SWAP","venueSymbol":"BTCUSDm"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("硬验拒收应 400: %d %s", rec.Code, rec.Body.String())
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_symbol_dialect`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("拒收落行 %d——红桩②破（零落表）", n)
	}
}

// ②-b tradeable 拒：TradeMode 非 FullAccess（broker 动态限实拍）。
func TestSubmitSymbolMappingTradeModeReject(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			sp := dxBTC(1)
			sp.sg.TradeMode = mt5pb.TradeMode_TradeMode_CloseOnly
			return &fakeDxSession{
				specs:  map[string]*dxSpec{"BTCUSDm": sp},
				quotes: map[string]mt5.QuoteSnapshot{"BTCUSDm": {Bid: 90000, Ask: 90001}},
			}, nil
		},
		func(context.Context, string) (float64, string, string, error) { return 1, "BTC", "live", nil })
	seedDxCreds(t, ctx, pool, mk, "exness", "3001157")
	rec := dxPost(t, mux, dxProcMapping,
		`{"venue":"exness","canonical":"BTC-USDT-SWAP","venueSymbol":"BTCUSDm"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("tradeable 拒应 400: %d", rec.Code)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_symbol_dialect`).Scan(&n)
	if n != 0 {
		t.Fatalf("零落表破: %d", n)
	}
}

// ③全环：官名→硬验（ctVal/计价币/digits/tradeable/探活全过）→dialect 行落
// （spec_hash/quote_snapshot 在案）+ops_event 审计行单事务。
func TestSubmitSymbolMappingFullLoop(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{
				syms:   []string{"BTCUSDm"},
				specs:  map[string]*dxSpec{"BTCUSDm": dxBTC(1)},
				quotes: map[string]mt5.QuoteSnapshot{"BTCUSDm": {Bid: 90000, Ask: 90001}},
			}, nil
		},
		func(context.Context, string) (float64, string, string, error) { return 1, "BTC", "live", nil })
	seedDxCreds(t, ctx, pool, mk, "exness", "3001157")
	rec := dxPost(t, mux, dxProcMapping,
		`{"venue":"exness","canonical":"BTC-USDT-SWAP","venueSymbol":"BTCUSDm","note":"业主挑定"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("全环应 200: %d %s", rec.Code, rec.Body.String())
	}
	var resp dashboardv1.SubmitSymbolMappingResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.GetVerified() || resp.GetEvidence().GetCtVal() != 1 || resp.GetEvidence().GetSpecHash() == "" {
		t.Fatalf("evidence 在案破: %+v", resp.GetEvidence())
	}
	var vs, hash string
	var snap []byte
	if err := pool.QueryRow(ctx, `SELECT venue_symbol,spec_hash,quote_snapshot FROM fin_symbol_dialect
		WHERE venue='exness' AND account='3001157' AND canonical='BTC-USDT-SWAP'`).Scan(&vs, &hash, &snap); err != nil {
		t.Fatalf("行落破: %v", err)
	}
	if vs != "BTCUSDm" || hash == "" || len(snap) == 0 {
		t.Fatalf("行面破: %s %s %d", vs, hash, len(snap))
	}
	var audits int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event WHERE event='venue符号映射'
		AND esc_scope='exness:3001157'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("审计行=%d——单事务落账破", audits)
	}
}

// ④官名清单读面（okx.symbols owner 册实读升序；缺=空集如实）。
func TestListCanonicalUniverse(t *testing.T) {
	cfg, pool, ctx, mux := dialectMux(t, nil, nil)
	if _, err := pool.Exec(ctx, `INSERT INTO app_config(key,value) VALUES('okx.symbols','["BTC-USDT-SWAP","ETH-USDT-SWAP"]')
		ON CONFLICT (key) DO UPDATE SET value=EXCLUDED.value`); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	rec := dxPost(t, mux, dxProcUniverse, `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("universe 读败: %d", rec.Code)
	}
	var resp dashboardv1.ListCanonicalUniverseResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.GetInsts()) != 2 || resp.GetInsts()[0] != "BTC-USDT-SWAP" {
		t.Fatalf("官名清单破: %v", resp.GetInsts())
	}
}

// ①-RPC 径：ListSymbolCandidates 过 mux（会话拨号+枚举+排序全链）。
func TestListSymbolCandidatesRPC(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{
				syms:  []string{"BTCUSDm", "ETHUSDm"},
				specs: map[string]*dxSpec{"BTCUSDm": dxBTC(1)},
			}, nil
		},
		nil)
	seedDxCreds(t, ctx, pool, mk, "exness", "3001157")
	rec := dxPost(t, mux, dxProcCandidates,
		`{"venue":"exness","canonical":"BTC-USDT-SWAP"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("候选 RPC 败: %d %s", rec.Code, rec.Body.String())
	}
	var resp dashboardv1.ListSymbolCandidatesResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.GetScanned() != 2 || len(resp.GetRows()) != 1 || resp.GetRows()[0].GetVenueSymbol() != "BTCUSDm" {
		t.Fatalf("候选面破: scanned=%d rows=%v", resp.GetScanned(), resp.GetRows())
	}
}

// 防误用锚：dollarPegged 族判据（USD/USDT/USDC 同族——族外拒）。
func TestDollarPeggedFamily(t *testing.T) {
	for _, ok := range []string{"USD", "USDT", "USDC", "usdt"} {
		if !dollarPegged(ok) {
			t.Fatalf("族内 %q 误拒", ok)
		}
	}
	for _, no := range []string{"BTC", "JPY", "", "EUR"} {
		if dollarPegged(no) {
			t.Fatalf("族外 %q 误纳", no)
		}
	}
	if canonicalBase("BTC-USDT-SWAP") != "BTC" || canonicalQuote("BTC-USDT-SWAP") != "USDT" {
		t.Fatal("官名 base/quote 提取破")
	}
}

// 连接败=拒（硬验 session 环——死会话自然拒）。
func TestSubmitSymbolMappingConnFail(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{connErr: errors.New("invalid login")}, nil
		},
		func(context.Context, string) (float64, string, string, error) { return 1, "BTC", "live", nil })
	seedDxCreds(t, ctx, pool, mk, "exness", "3001157")
	rec := dxPost(t, mux, dxProcMapping,
		`{"venue":"exness","canonical":"BTC-USDT-SWAP","venueSymbol":"BTCUSDm"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("连接败应 400: %d", rec.Code)
	}
	var n int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM fin_symbol_dialect`).Scan(&n)
	if n != 0 {
		t.Fatalf("零落表破: %d", n)
	}
}

// O-35250①#尾方言剥：XM #=tradable 变体标——剥后余串进 quote 层；
// 无#同名符（观察面）同剥同 base 不碍；非#尾形零回归。
func TestSplitDialectHash(t *testing.T) {
	cases := []struct{ sym, base, suf string }{
		{"BTCUSD#", "BTC", "usd"},       // #剥→USD剥→base=BTC（spec 钉案）
		{"BTCUSD", "BTC", "usd"},        // 无#同名观察面——同剥同 base
		{"AUDUSD#", "AUD", "usd"},       // #尾+quote 双剥
		{"GOLD#", "GOLD", ""},           // 无 quote 尾——#剥后 base=GOLD
		{"US500Cash#", "US500Cash", ""}, // Cash 尾非报价族——#剥即止
		{"OIL-JAN27", "OIL-JAN27", ""},  // 期货月年形无#无quote——原样
		{"BTCUSD.pro", "BTC", "usd"},    // .PRO marker 层零回归
		{"BTCUSDm", "BTC", "usdm"},      // m 尾方言零回归
	}
	for _, c := range cases {
		b, s := splitDialect(c.sym)
		if b != c.base || s != c.suf {
			t.Fatalf("splitDialect(%q)=(%q,%q) want (%q,%q)", c.sym, b, s, c.base, c.suf)
		}
	}
}

// O-35250②XM 目录候选面：SymbolList 218 符含#尾——canonical BTC-USDT-SWAP
// 候选命中 BTCUSD#/BTCUSD 双形（#=tradable 优先序同分并列如实），
// symbols 原文字段=目录浏览面数据源（rankCandidates 前全量）。
func TestXMHashCandidatesRPC(t *testing.T) {
	oldPace := dialectPace
	dialectPace = 0
	defer func() { dialectPace = oldPace }()
	xmSyms := []string{"BTCUSD#", "BTCUSD", "ETHUSD#", "GOLD#", "SILVER#",
		"AUDUSD", "AUDUSD#", "EURUSD", "US500Cash#", "OILCash#",
		"BRENT-JAN27", "OIL-JAN27"}
	for len(xmSyms) < 218 {
		xmSyms = append(xmSyms, "FILL"+string(rune('A'+len(xmSyms)%26))+string(rune('0'+len(xmSyms)%10)))
	}
	specs := map[string]*dxSpec{"BTCUSD#": dxBTC(1), "BTCUSD": dxBTC(1)}
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{syms: xmSyms, specs: specs}, nil
		}, nil)
	mk := brokerMk(t)
	seedDxCreds(t, ctx, pool, mk, "xm", "88001234")
	rec := dxPost(t, mux, dxProcCandidates,
		`{"venue":"xm","canonical":"BTC-USDT-SWAP"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("候选 RPC 败: %d %s", rec.Code, rec.Body.String())
	}
	var resp dashboardv1.ListSymbolCandidatesResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.GetScanned() != 218 || len(resp.GetSymbols()) != 218 {
		t.Fatalf("目录面破: scanned=%d symbols=%d", resp.GetScanned(), len(resp.GetSymbols()))
	}
	got := map[string]bool{}
	for _, r := range resp.GetRows() {
		got[r.GetVenueSymbol()] = true
	}
	if !got["BTCUSD#"] || !got["BTCUSD"] {
		t.Fatalf("#尾候选未命中: %v", got)
	}
	// 目录浏览数据源=原文全量（含未入候选的观察面/期货形）
	var hasRaw bool
	for _, s := range resp.GetSymbols() {
		if s == "AUDUSD" || s == "OIL-JAN27" {
			hasRaw = true
		}
	}
	if !hasRaw {
		t.Fatal("symbols 原文缺观察面/期货形——目录浏览数据源破")
	}
}

// O-35250③#尾手输落行：候选机外 venue_symbol 同走 SubmitSymbolMapping
// 硬验径（后端零改——GOLD# 实认过全谱验落 fin_symbol_dialect）。
func TestSubmitSymbolMappingHashSymbol(t *testing.T) {
	mk := brokerMk(t)
	_, pool, ctx, mux := dialectMux(t,
		func(string) (dialectSession, error) {
			return &fakeDxSession{
				syms:   []string{"GOLD#", "XAUUSD"},
				specs:  map[string]*dxSpec{"GOLD#": dxBTC(1), "XAUUSD": dxBTC(1)},
				quotes: map[string]mt5.QuoteSnapshot{"GOLD#": {Bid: 2650, Ask: 2650.35}},
			}, nil
		},
		func(context.Context, string) (float64, string, string, error) { return 1, "XAU", "live", nil })
	seedDxCreds(t, ctx, pool, mk, "xm", "88001234")
	rec := dxPost(t, mux, dxProcMapping,
		`{"venue":"xm","canonical":"XAU-USD-SWAP","venueSymbol":"GOLD#","note":"目录浏览手挑"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("#尾手输硬验应过: %d %s", rec.Code, rec.Body.String())
	}
	var vs string
	if err := pool.QueryRow(ctx, `SELECT venue_symbol FROM fin_symbol_dialect
		WHERE venue='xm' AND canonical='XAU-USD-SWAP'`).Scan(&vs); err != nil {
		t.Fatalf("#尾行未落: %v", err)
	}
	if vs != "GOLD#" {
		t.Fatalf("落行 venue_symbol=%s want GOLD#", vs)
	}
}
