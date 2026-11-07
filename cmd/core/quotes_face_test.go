// quotes_face_test.go —— T-0390 锚②③桩测：ListPriceAlerts 成功径 srows.Err()=nil
// 对 nil error 裸调 .Error()→panic（生产实证每调即炸→前端恒 502）根修钉——
// 零提醒行/有行两态成功径不 panic 且返 200+正常 payload（rows+symbols）。
// 先红后绿：修前裸调形态本断言必红（T-0388 纪律）。
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/internal/platform/config"
	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

func quotesFaceMux(t *testing.T) (http.Handler, *pgxpool.Pool, context.Context) {
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
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, coreService{pool: pool, cfg: cfg})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server)
	return mux, pool, ctx
}

func listPriceAlerts(t *testing.T, mux http.Handler) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/o.dashboard.v1.CoreService/ListPriceAlerts", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req) // 裸调形态=panic 直透本 goroutine（修前红锚）
	var body map[string]any
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("200 载荷非 JSON: %v", err)
		}
	}
	return rec.Code, body
}

// 零提醒行态：fin_price_alert 空表——成功径仍须返 200+symbols 枚举不炸。
func TestListPriceAlertsEmptyOK(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT',65000,64999,65001)`); err != nil {
		t.Fatal(err)
	}
	code, body := listPriceAlerts(t, mux)
	if code != http.StatusOK {
		t.Fatalf("零提醒行成功径应 200: code=%d body=%s", code, body)
	}
	syms, _ := body["symbols"].([]any)
	if len(syms) == 0 {
		t.Fatalf("symbols 枚举应含在采品种: body=%v", body)
	}
	if rows, _ := body["rows"].([]any); len(rows) != 0 {
		t.Fatalf("零提醒行态 rows 应空: body=%v", body)
	}
}

// 有行态：提醒行×最新价投影——成功径返 200+rows/symbols 载荷不炸。
func TestListPriceAlertsRowsOK(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT',65000,64999,65001),(now(),'TRX-USDT',0.15,0.149,0.151)`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO fin_price_alert(symbol,direction,target,note,created_by)
		VALUES('BTC-USDT','>=',70000,'桩测提醒','t0390')`); err != nil {
		t.Fatal(err)
	}
	code, body := listPriceAlerts(t, mux)
	if code != http.StatusOK {
		t.Fatalf("有行成功径应 200: code=%d body=%s", code, body)
	}
	rows, _ := body["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows 应恰 1 行: body=%v", body)
	}
	row, _ := rows[0].(map[string]any)
	if row["symbol"] != "BTC-USDT" || row["state"] != "active" {
		t.Fatalf("row 投影错: row=%v", row)
	}
	if row["hasQuote"] != true || row["last"] != 65000.0 {
		t.Fatalf("最新价投影错（has_quote/last）: row=%v", row)
	}
	syms, _ := body["symbols"].([]any)
	if len(syms) != 2 {
		t.Fatalf("symbols 应枚举 2 在采品种: body=%v", body)
	}
}

// —— T-0391 自选报价条桩测（票锚②③④——搜索候选域=md_ticks 在采集；选存后
// 瓦片成员价；保存全量替换真值重读+seat=dashboard-ui 审计行）——

func watchCall(t *testing.T, mux http.Handler, proc, body string, withOrigin bool) (int, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		"/o.dashboard.v1.CoreService/"+proc, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if withOrigin {
		req.Host = "ui.local"
		req.Header.Set("Origin", "http://ui.local")
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	var m map[string]any
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
			t.Fatalf("200 载荷非 JSON: %v", err)
		}
	}
	return rec.Code, m
}

// 候选域=md_ticks 在采集（票锚①搜索「trx」面后端正身：Get.symbols 枚举含
// TRX-USDT-SWAP）；Set 全量替换+真值重读+审计行（票锚③）。
func TestQuoteWatchlistSetGetAudit(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT-SWAP',65000,64999,65001),
		      (now(),'TRX-USDT-SWAP',0.15,0.149,0.151)`); err != nil {
		t.Fatal(err)
	}
	// 零成员态：rows 空如实缺
	code, body := watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	if code != http.StatusOK {
		t.Fatalf("Get 零成员应 200: code=%d", code)
	}
	if rows, _ := body["rows"].([]any); len(rows) != 0 {
		t.Fatalf("零成员 rows 应空: %v", body)
	}
	if syms, _ := body["symbols"].([]any); len(syms) != 2 {
		t.Fatalf("候选域应在采 2 员: %v", body)
	}
	// Set 全量替换（序即 ord——TRX 前 BTC 后）
	code, body = watchCall(t, mux, "SetQuoteWatchlist",
		`{"symbols":["TRX-USDT-SWAP","BTC-USDT-SWAP"]}`, true)
	if code != http.StatusOK {
		t.Fatalf("Set 应 200: code=%d body=%v", code, body)
	}
	syms, _ := body["symbols"].([]any)
	if len(syms) != 2 || syms[0] != "TRX-USDT-SWAP" || syms[1] != "BTC-USDT-SWAP" {
		t.Fatalf("Set 真值重读序错: %v", body)
	}
	// Get 成员×最新价投影（ord 序+has_quote+价）
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	rows, _ := body["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("成员 rows 应 2: %v", body)
	}
	r0, _ := rows[0].(map[string]any)
	if r0["symbol"] != "TRX-USDT-SWAP" || r0["hasQuote"] != true || r0["last"] != 0.15 {
		t.Fatalf("首员投影错: %v", r0)
	}
	// 审计行 seat=dashboard-ui
	var seat, event string
	if err := pool.QueryRow(ctx, `SELECT seat,event FROM ops_event
		WHERE event='quote_watchlist_set' ORDER BY id DESC LIMIT 1`).Scan(&seat, &event); err != nil {
		t.Fatalf("审计行缺: %v", err)
	}
	if seat != "dashboard-ui" {
		t.Fatalf("审计行 seat 错: %s", seat)
	}
	// 全量替换语义：二次 Set 覆盖旧集
	_, body = watchCall(t, mux, "SetQuoteWatchlist", `{"symbols":["BTC-USDT-SWAP"]}`, true)
	syms, _ = body["symbols"].([]any)
	if len(syms) != 1 || syms[0] != "BTC-USDT-SWAP" {
		t.Fatalf("二次 Set 应全量覆盖: %v", body)
	}
	// 非在采品种=拒（候选域闸——不开任意 symbol 输入）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"symbols":["NOPE-USDT"]}`, true)
	if code == http.StatusOK {
		t.Fatal("非在采 symbol 应拒")
	}
	// Origin 缺席=拒（同源闸同建行式）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"symbols":["BTC-USDT-SWAP"]}`, false)
	if code == http.StatusOK {
		t.Fatal("Origin 缺席应拒")
	}
}

// —— T-0399 提醒径鲜度闸桩测（票锚①②：活符过/270h 孤行拒/候选死符消失+
// 边界=阈——先红后绿 T-0388 纪律）——

// TestAlertFreshnessGate 活/死符建单+候选滤活（含 mt5 源节拍自适——72h 界
// 汇市周末静默非死源：20h 陈 mt5 行过、270h mt5 行拒）。
func TestAlertFreshnessGate(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts) VALUES
		(now(),'okx','BTC-USDT-SWAP',65000,64999,65001,now()),
		(now()-interval '270 hours','okx','TRX-USDT-SWAP',0.15,0.149,0.151,now()-interval '270 hours'),
		(now()-interval '20 hours','mt5','EURUSDm',1.08,1.079,1.081,now()-interval '20 hours'),
		(now()-interval '270 hours','mt5','DEADJPYm',150,149,151,now()-interval '270 hours')`); err != nil {
		t.Fatal(err)
	}
	// ① 活符建单通
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","direction":">=","target":70000}`, true)
	if code != http.StatusOK {
		t.Fatalf("活符建单应 200: code=%d body=%v", code, body)
	}
	// 270h 孤行拒建（业主实证锚）
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"TRX-USDT-SWAP","direction":">=","target":1}`, true)
	if code == http.StatusOK {
		t.Fatal("270h 孤行陈符应拒建")
	}
	// mt5 20h 陈行=源节拍内（周末静默非死）——过；canon 键落库大小写方言不打坏
	code, body = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"EURUSDm","direction":"<=","target":1.0}`, true)
	if code != http.StatusOK {
		t.Fatalf("mt5 源节拍内建单应 200: code=%d", code)
	}
	if row, _ := body["row"].(map[string]any); row["symbol"] != "EURUSDm" {
		t.Fatalf("canon inst_id 落库错（大小写方言）: %v", body)
	}
	// mt5 270h 超源节拍界=死符——拒
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"DEADJPYm","direction":">=","target":200}`, true)
	if code == http.StatusOK {
		t.Fatal("mt5 270h 死符应拒建")
	}
	// ② 候选列表死符消失（ListPriceAlerts+GetQuoteWatchlist 双候选域同口径）
	_, body = listPriceAlerts(t, mux)
	syms, _ := body["symbols"].([]any)
	got := map[string]bool{}
	for _, s := range syms {
		got[s.(string)] = true
	}
	if !got["BTC-USDT-SWAP"] || !got["EURUSDm"] || got["TRX-USDT-SWAP"] || got["DEADJPYm"] {
		t.Fatalf("List 候选域滤活错: %v", syms)
	}
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	syms, _ = body["symbols"].([]any)
	got = map[string]bool{}
	for _, s := range syms {
		got[s.(string)] = true
	}
	if !got["BTC-USDT-SWAP"] || !got["EURUSDm"] || got["TRX-USDT-SWAP"] || got["DEADJPYm"] {
		t.Fatalf("Watchlist 候选域滤活错: %v", syms)
	}
	// 自选写闸同口径——陈符拒入集
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"symbols":["TRX-USDT-SWAP"]}`, true)
	if code == http.StatusOK {
		t.Fatal("自选面陈符应拒入集")
	}
}

// —— T-0412 意图晋级径桩测（票锚：目录符建单→sub_request 行落+候选域并集
// +pending 标+域外拒——端到端前半段 SG 侧）——

// TestIntentPromotePath 目录未订阅符建单→晋升提名同事务落+选符域两域并集
// +pending 标+域外/陈目录仍拒（typo 闸不放松）。
func TestIntentPromotePath(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	// 活符 BTC（在采）+目录符 TRX（粗扫在谱零 tick——僵尸病灶原型）+陈目录
	// DEAD（休拍收缩防陈目录放提名）+域外 NOPE（两域皆无）
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts)
		VALUES(now(),'okx','BTC-USDT-SWAP',65000,64999,65001,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,last,bid,ask,recv_ts) VALUES
		(now(),'okx','TRX-USDT-SWAP',0.15,0.149,0.151,now()),
		(now()-interval '2 hours','okx','DEAD-USDT-SWAP',1,1,1,now()-interval '2 hours')`); err != nil {
		t.Fatal(err)
	}
	// ① 目录符建单=收+同事务提名行（user_intent/open——僵尸不再）
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"trx-usdt-swap","direction":">=","target":0.2}`, true)
	if code != http.StatusOK {
		t.Fatalf("目录符建单应 200: code=%d body=%v", code, body)
	}
	var requester, state, reason string
	if err := pool.QueryRow(ctx, `SELECT requester,state,reason FROM fin_sub_request
		WHERE inst_id='TRX-USDT-SWAP'`).Scan(&requester, &state, &reason); err != nil {
		t.Fatalf("晋升提名行缺（同事务未落）: %v", err)
	}
	if requester != "user_intent" || state != "open" || !strings.Contains(reason, "price_alert#") {
		t.Fatalf("提名行载荷错: requester=%s state=%s reason=%s", requester, state, reason)
	}
	if row, _ := body["row"].(map[string]any); row["symbol"] != "TRX-USDT-SWAP" {
		t.Fatalf("canon 键落库错（小写入应归一原键）: %v", body)
	}
	// ② 候选域两域并集+pending 标（TRX pending、BTC 活符非、DEAD 陈目录不入域）
	_, body = listPriceAlerts(t, mux)
	got := map[string]bool{}
	for _, s := range body["symbols"].([]any) {
		got[s.(string)] = true
	}
	if !got["BTC-USDT-SWAP"] || !got["TRX-USDT-SWAP"] || got["DEAD-USDT-SWAP"] {
		t.Fatalf("候选域并集错: %v", body["symbols"])
	}
	pgot := map[string]bool{}
	for _, s := range body["pendingSymbols"].([]any) {
		pgot[s.(string)] = true
	}
	if !pgot["TRX-USDT-SWAP"] || pgot["BTC-USDT-SWAP"] || pgot["DEAD-USDT-SWAP"] {
		t.Fatalf("pending 标错: %v", body["pendingSymbols"])
	}
	// ③ 域外符仍拒+陈目录符拒（闸没放松到裸奔）
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"NOPE-USDT","direction":">=","target":1}`, true)
	if code == http.StatusOK {
		t.Fatal("域外 typo 符应拒")
	}
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"DEAD-USDT-SWAP","direction":">=","target":1}`, true)
	if code == http.StatusOK {
		t.Fatal("陈目录符应拒（目录休拍收缩）")
	}
	// ④ 自选目录符=收+提名同径（upsert 同键幂等——TRX 已有 alert 提名不重生）
	code, body = watchCall(t, mux, "SetQuoteWatchlist", `{"symbols":["TRX-USDT-SWAP"]}`, true)
	if code != http.StatusOK {
		t.Fatalf("自选目录符应 200: %v", body)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_sub_request
		WHERE inst_id='TRX-USDT-SWAP' AND requester='user_intent' AND state='open'`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("自选提名行应恰 1（同键幂等）: n=%d err=%v", n, err)
	}
	// ⑤ GetQuoteWatchlist pending 同径携标
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	pends, _ := body["pendingSymbols"].([]any)
	if len(pends) != 1 || pends[0] != "TRX-USDT-SWAP" {
		t.Fatalf("watch pending 应恰含 TRX: %v", body)
	}
}

// TestAlertFreshnessBoundary 边界=阈（票锚桩测三件③）：okx 阈内 14min 过、
// 阈外 16min 拒——谓词窗边缘两贴不漂。
func TestAlertFreshnessBoundary(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask,recv_ts) VALUES
		(now()-interval '14 minutes','EDGE-FRESH',1,1,1,now()-interval '14 minutes'),
		(now()-interval '16 minutes','EDGE-DEAD',1,1,1,now()-interval '16 minutes')`); err != nil {
		t.Fatal(err)
	}
	code, _ := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"EDGE-FRESH","direction":">=","target":2}`, true)
	if code != http.StatusOK {
		t.Fatalf("阈内 14min 应过: code=%d", code)
	}
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"EDGE-DEAD","direction":">=","target":2}`, true)
	if code == http.StatusOK {
		t.Fatal("阈外 16min 应拒")
	}
}

// TestIntentPromoteFamilySpot T-0427 粗扫扩全宇宙消费面对拍：SPOT/FUTURES
// 族符落 md_tick_scan 后，选符域谓词（candDomainSQL/symDomain）零改动自动
// 见新族符；现货目录符建单同事务提名晋升（SubRequestArm 族兼容前半段——
// 现货腿 tickers WS 通道生产已实证：dcd 现货腿 T-0171 同源）。
func TestIntentPromoteFamilySpot(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,inst_type,last,bid,ask,recv_ts) VALUES
		(now(),'okx','ETH-USDT','spot',3000,2999,3001,now()),
		(now(),'okx','QQQ-USD-260327','futures',7,7,7,now())`); err != nil {
		t.Fatal(err)
	}
	// ① 候选域零改动见新族符（SPOT/FUTURES 都入域——谓词无族过滤）
	_, body := listPriceAlerts(t, mux)
	got := map[string]bool{}
	for _, s := range body["symbols"].([]any) {
		got[s.(string)] = true
	}
	if !got["ETH-USDT"] || !got["QQQ-USD-260327"] {
		t.Fatalf("族符应零改动入选符域: %v", body["symbols"])
	}
	// ② 现货目录符建单=收+同事务提名晋升（user_intent/open）
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"eth-usdt","direction":">=","target":3200}`, true)
	if code != http.StatusOK {
		t.Fatalf("现货目录符建单应 200: code=%d body=%v", code, body)
	}
	var requester, state string
	if err := pool.QueryRow(ctx, `SELECT requester,state FROM fin_sub_request
		WHERE inst_id='ETH-USDT'`).Scan(&requester, &state); err != nil {
		t.Fatalf("现货符晋升提名行缺: %v", err)
	}
	if requester != "user_intent" || state != "open" {
		t.Fatalf("提名行载荷错: requester=%s state=%s", requester, state)
	}
}
