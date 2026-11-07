// quotes_face_test.go —— T-0390 锚②③桩测：ListPriceAlerts 成功径 srows.Err()=nil
// 对 nil error 裸调 .Error()→panic（生产实证每调即炸→前端恒 502）根修钉——
// 零提醒行/有行两态成功径不 panic 且返 200+正常 payload（rows+symbols）。
// 先红后绿：修前裸调形态本断言必红（T-0388 纪律）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	// Set 全量替换（序即 ord——TRX 前 BTC 后；T-0442 新形制 items 携源约束）
	code, body = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"TRX-USDT-SWAP"},{"symbol":"BTC-USDT-SWAP"}]}`, true)
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
	_, body = watchCall(t, mux, "SetQuoteWatchlist", `{"items":[{"symbol":"BTC-USDT-SWAP"}]}`, true)
	syms, _ = body["symbols"].([]any)
	if len(syms) != 1 || syms[0] != "BTC-USDT-SWAP" {
		t.Fatalf("二次 Set 应全量覆盖: %v", body)
	}
	// 非在采品种=拒（候选域闸——不开任意 symbol 输入）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"items":[{"symbol":"NOPE-USDT"}]}`, true)
	if code == http.StatusOK {
		t.Fatal("非在采 symbol 应拒")
	}
	// Origin 缺席=拒（同源闸同建行式）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"items":[{"symbol":"BTC-USDT-SWAP"}]}`, false)
	if code == http.StatusOK {
		t.Fatal("Origin 缺席应拒")
	}
}

// —— T-0442 三件桩测（票锚②③④——先红后绿 T-0388 纪律）——

// TestAlertDualThreshold 件②：双阈单事务 1-2 行原子落（落行层仍 direction
// 行——零 schema 破）+双空禁建+单阈兼容+负/零值拒。
func TestAlertDualThreshold(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT-SWAP',65000,64999,65001)`); err != nil {
		t.Fatal(err)
	}
	// ① 双填=两行（>= 上限 + <= 下限——落行层 direction 行写）
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","hasUpper":true,"upperTarget":70000,
		"hasLower":true,"lowerTarget":60000}`, true)
	if code != http.StatusOK {
		t.Fatalf("双阈建单应 200: code=%d body=%v", code, body)
	}
	rows, _ := body["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("双阈应落 2 行: %v", body)
	}
	dirs := map[string]float64{}
	for _, r := range rows {
		m, _ := r.(map[string]any)
		dirs[m["direction"].(string)] = m["target"].(float64)
	}
	if dirs[">="] != 70000 || dirs["<="] != 60000 {
		t.Fatalf("双阈落行错: %v", dirs)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM fin_price_alert
		WHERE symbol='BTC-USDT-SWAP' AND state='active'`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("双阈行应齐 2: n=%d err=%v", n, err)
	}
	// ② 双空=禁建
	code, _ = watchCall(t, mux, "CreatePriceAlert", `{"symbol":"BTC-USDT-SWAP"}`, true)
	if code == http.StatusOK {
		t.Fatal("双空应禁建")
	}
	// ③ 单阈兼容（只填下限）
	code, body = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","hasLower":true,"lowerTarget":50000}`, true)
	if code != http.StatusOK {
		t.Fatalf("单下限应 200: code=%d body=%v", code, body)
	}
	if rows, _ = body["rows"].([]any); len(rows) != 1 {
		t.Fatalf("单阈应恰 1 行: %v", body)
	}
	// ④ 在位但非法值=拒（0/负——在位标判填非零值判填的反向锚）
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","hasUpper":true,"upperTarget":-5}`, true)
	if code == http.StatusOK {
		t.Fatal("负上限应拒")
	}
	// ⑤ 审计行一单一条携双条件
	var note string
	if err := pool.QueryRow(ctx, `SELECT note FROM ops_event
		WHERE event='price_alert_build' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("审计行缺: %v", err)
	}
	if !strings.Contains(note, "BTC-USDT-SWAP") {
		t.Fatalf("审计行缺品种: %s", note)
	}
}

// TestAlertSoftDelete 件①：删除径=软删 disabled 留痕（行不物理删）——active
// 翻 disabled 可查；fired 终态禁手翻（幂等回真值）——史行只读。
func TestAlertSoftDelete(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,inst_id,last,bid,ask)
		VALUES(now(),'BTC-USDT-SWAP',65000,64999,65001)`); err != nil {
		t.Fatal(err)
	}
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","hasUpper":true,"upperTarget":70000}`, true)
	if code != http.StatusOK {
		t.Fatalf("建单应 200: %v", body)
	}
	rows, _ := body["rows"].([]any)
	row, _ := rows[0].(map[string]any)
	// proto3 JSON int64=string——id 解串走 strconv（float 断言=panic 病灶）
	id, err2 := strconv.ParseInt(row["id"].(string), 10, 64)
	if err2 != nil {
		t.Fatalf("id 解串败: %v", err2)
	}
	// 软删：active→disabled——行留痕（count 不减、state=disabled）
	code, body = watchCall(t, mux, "SetPriceAlertState",
		fmt.Sprintf(`{"id":%d,"state":"disabled"}`, id), true)
	if code != http.StatusOK {
		t.Fatalf("软删应 200: %v", body)
	}
	var state string
	var cnt int
	if err := pool.QueryRow(ctx, `SELECT state,count(*) OVER () FROM fin_price_alert
		WHERE id=$1`, id).Scan(&state, &cnt); err != nil || state != "disabled" || cnt != 1 {
		t.Fatalf("软删应留痕 disabled 行不减: state=%s cnt=%d err=%v", state, cnt, err)
	}
	// 恢复径回 active（软删可逆——已停用区恢复钮后端径）
	code, _ = watchCall(t, mux, "SetPriceAlertState",
		fmt.Sprintf(`{"id":%d,"state":"active"}`, id), true)
	if code != http.StatusOK {
		t.Fatal("恢复应 200")
	}
	// fired 终态禁手翻：手工翻 fired 后翻位请求=幂等回真值不报错
	if _, err := pool.Exec(ctx, `UPDATE fin_price_alert SET state='fired',
		fired_at=now() WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	code, body = watchCall(t, mux, "SetPriceAlertState",
		fmt.Sprintf(`{"id":%d,"state":"disabled"}`, id), true)
	if code != http.StatusOK {
		t.Fatalf("fired 翻位应幂等 200: %v", body)
	}
	if err := pool.QueryRow(ctx, `SELECT state FROM fin_price_alert
		WHERE id=$1`, id).Scan(&state); err != nil || state != "fired" {
		t.Fatalf("fired 终态禁手翻——史行只读: state=%s err=%v", state, err)
	}
}

// TestWatchVenueConstraint 件③：源约束取价过滤（④「选源后报价行只显所选
// 源价」——okx/mt5 双源异价，约束切源价随切）+空=全部源最新跨源+quote_venue
// 来源实证+src_sets 实采派生+源约束闸（谱外源拒/冒号拒）+源感知晋升（约束
// 谱上未采源逐源提名）。
func TestWatchVenueConstraint(t *testing.T) {
	mux, pool, ctx := quotesFaceMux(t)
	// BTC 双源实采异价（mt5 价新——空约束应取 mt5 跨源最新）；ETH 单源 okx
	if _, err := pool.Exec(ctx, `INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask,recv_ts) VALUES
		(now()-interval '1 minute','okx','BTC-USDT-SWAP',65000,64999,65001,now()-interval '1 minute'),
		(now(),'mt5','BTC-USDT-SWAP',65100,65099,65101,now()),
		(now(),'okx','ETH-USDT',3000,2999,3001,now())`); err != nil {
		t.Fatal(err)
	}
	// ① 约束 okx → 只显 okx 价（quote_venue=okx 实证）
	code, body := watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"BTC-USDT-SWAP","venues":["okx"]}]}`, true)
	if code != http.StatusOK {
		t.Fatalf("约束建单应 200: %v", body)
	}
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	rows, _ := body["rows"].([]any)
	r0, _ := rows[0].(map[string]any)
	if r0["quoteVenue"] != "okx" || r0["last"] != 65000.0 {
		t.Fatalf("约束 okx 应只显 okx 价: %v", r0)
	}
	// ② 约束切 mt5 → 价随切（65100）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"BTC-USDT-SWAP","venues":["mt5"]}]}`, true)
	if code != http.StatusOK {
		t.Fatal("切源应 200")
	}
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	rows, _ = body["rows"].([]any)
	r0, _ = rows[0].(map[string]any)
	if r0["quoteVenue"] != "mt5" || r0["last"] != 65100.0 {
		t.Fatalf("约束 mt5 应只显 mt5 价: %v", r0)
	}
	// ③ 空约束=全部源（最新跨源——mt5 价新取 mt5，语义同旧 DISTINCT ON）
	code, _ = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"BTC-USDT-SWAP"},{"symbol":"ETH-USDT"}]}`, true)
	if code != http.StatusOK {
		t.Fatal("空约束应 200")
	}
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	rows, _ = body["rows"].([]any)
	r0, _ = rows[0].(map[string]any)
	if r0["quoteVenue"] != "mt5" || r0["last"] != 65100.0 {
		t.Fatalf("空约束应最新跨源: %v", r0)
	}
	// ④ src_sets=实采派生（BTC:[mt5 okx] 有序、ETH:[okx]——零硬编码词表）
	got := map[string][]any{}
	for _, s := range body["srcSets"].([]any) {
		m, _ := s.(map[string]any)
		got[m["symbol"].(string)] = m["venues"].([]any)
	}
	if len(got["BTC-USDT-SWAP"]) != 2 || got["BTC-USDT-SWAP"][0] != "mt5" ||
		got["BTC-USDT-SWAP"][1] != "okx" || len(got["ETH-USDT"]) != 1 {
		t.Fatalf("src_sets 派生错: %v", got)
	}
	// ⑤ 源约束闸：谱外源拒/冒号裸值拒
	code, _ = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"ETH-USDT","venues":["binance"]}]}`, true)
	if code == http.StatusOK {
		t.Fatal("谱外源应拒（chip 候选同谓词）")
	}
	code, _ = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"ETH-USDT","venues":["okx:spot"]}]}`, true)
	if code == http.StatusOK {
		t.Fatal("冒号 venue 应拒（D-496 裸值闸）")
	}
	// ⑥ 源感知晋升：目录符 ETH 谱上 okx 未采（md_tick_scan 有谱零 tick）+
	// 约束 mt5（谱外）——mt5 拒不变，约束 okx → 提名 (okx,ETH) user_intent 行
	if _, err := pool.Exec(ctx, `INSERT INTO md_tick_scan(ts,venue,inst_id,inst_type,last,bid,ask,recv_ts)
		VALUES(now(),'okx','SOL-USDT','swap',100,99,101,now())`); err != nil {
		t.Fatal(err)
	}
	code, _ = watchCall(t, mux, "SetQuoteWatchlist",
		`{"items":[{"symbol":"SOL-USDT","venues":["okx"]}]}`, true)
	if code != http.StatusOK {
		t.Fatal("目录符约束源应 200")
	}
	var venue, requester, state string
	if err := pool.QueryRow(ctx, `SELECT venue,requester,state FROM fin_sub_request
		WHERE inst_id='SOL-USDT'`).Scan(&venue, &requester, &state); err != nil {
		t.Fatalf("源感知晋升提名行缺: %v", err)
	}
	if venue != "okx" || requester != "user_intent" || state != "open" {
		t.Fatalf("提名行载荷错: venue=%s requester=%s state=%s", venue, requester, state)
	}
	// ⑦ 约束源在谱待采 → Get 行 pending=true（源感知晋升中标）
	_, body = watchCall(t, mux, "GetQuoteWatchlist", `{}`, false)
	rows, _ = body["rows"].([]any)
	for _, r := range rows {
		m, _ := r.(map[string]any)
		if m["symbol"] == "SOL-USDT" && m["pending"] != true {
			t.Fatalf("约束谱上待采应标 pending: %v", m)
		}
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
	// ① 活符建单通（T-0442 件②新形制——hasUpper/upperTarget）
	code, body := watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"BTC-USDT-SWAP","hasUpper":true,"upperTarget":70000}`, true)
	if code != http.StatusOK {
		t.Fatalf("活符建单应 200: code=%d body=%v", code, body)
	}
	// 270h 孤行拒建（业主实证锚）
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"TRX-USDT-SWAP","hasUpper":true,"upperTarget":1}`, true)
	if code == http.StatusOK {
		t.Fatal("270h 孤行陈符应拒建")
	}
	// mt5 20h 陈行=源节拍内（周末静默非死）——过；canon 键落库大小写方言不打坏
	code, body = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"EURUSDm","hasLower":true,"lowerTarget":1.0}`, true)
	if code != http.StatusOK {
		t.Fatalf("mt5 源节拍内建单应 200: code=%d", code)
	}
	rows0, _ := body["rows"].([]any)
	if len(rows0) != 1 {
		t.Fatalf("单阈建行应恰 1 行: %v", body)
	}
	if row, _ := rows0[0].(map[string]any); row["symbol"] != "EURUSDm" {
		t.Fatalf("canon inst_id 落库错（大小写方言）: %v", body)
	}
	// mt5 270h 超源节拍界=死符——拒
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"DEADJPYm","hasUpper":true,"upperTarget":200}`, true)
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
	code, _ = watchCall(t, mux, "SetQuoteWatchlist", `{"items":[{"symbol":"TRX-USDT-SWAP"}]}`, true)
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
		`{"symbol":"trx-usdt-swap","hasUpper":true,"upperTarget":0.2}`, true)
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
	rows1, _ := body["rows"].([]any)
	if row, _ := rows1[0].(map[string]any); row["symbol"] != "TRX-USDT-SWAP" {
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
		`{"symbol":"NOPE-USDT","hasUpper":true,"upperTarget":1}`, true)
	if code == http.StatusOK {
		t.Fatal("域外 typo 符应拒")
	}
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"DEAD-USDT-SWAP","hasUpper":true,"upperTarget":1}`, true)
	if code == http.StatusOK {
		t.Fatal("陈目录符应拒（目录休拍收缩）")
	}
	// ④ 自选目录符=收+提名同径（upsert 同键幂等——TRX 已有 alert 提名不重生）
	code, body = watchCall(t, mux, "SetQuoteWatchlist", `{"items":[{"symbol":"TRX-USDT-SWAP"}]}`, true)
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
		`{"symbol":"EDGE-FRESH","hasUpper":true,"upperTarget":2}`, true)
	if code != http.StatusOK {
		t.Fatalf("阈内 14min 应过: code=%d", code)
	}
	code, _ = watchCall(t, mux, "CreatePriceAlert",
		`{"symbol":"EDGE-DEAD","hasUpper":true,"upperTarget":2}`, true)
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
		`{"symbol":"eth-usdt","hasUpper":true,"upperTarget":3200}`, true)
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
