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
