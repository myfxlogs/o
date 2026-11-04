// earn_face_test.go —— T-0342 锚②桩测：Origin 同源闸（缺席/跨源/scheme 坏=
// PermissionDenied；同源=放行翻位）+写后 app_config 值/updated_by 审计列留痕+
// ops_event 翻位行+GetEarnAutoPurchase 回读真态（缺键=enabled=false 停用显形）。
// 全走 testutil 独立 schema+connecthttp 真挂载——不假数据不打生产。
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

func earnFaceMux(t *testing.T) (http.Handler, *config.Client, *pgxpool.Pool, context.Context) {
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
	return mux, cfg, pool, ctx
}

// earnPost connect unary JSON POST（connect-web 同形——浏览器同源 POST 恒带
// Origin；host 缺席回退 example.com 使闸面可测）。
func earnPost(t *testing.T, mux http.Handler, proc, body, origin, host string) *httptest.ResponseRecorder {
	t.Helper()
	if host == "" {
		host = "example.com"
	}
	req := httptest.NewRequest(http.MethodPost, proc, strings.NewReader(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestEarnAutoPurchaseOriginGate(t *testing.T) {
	mux, cfg, pool, ctx := earnFaceMux(t)
	const set = "/o.dashboard.v1.CoreService/SetEarnAutoPurchase"
	const body = `{"enabled":false,"reason":"桩测翻位"}`

	// Origin 缺席=拒（非页面径——curl/脚本面裸调不可写）。
	if rec := earnPost(t, mux, set, body, "", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("Origin 缺席应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	// 跨源=拒（host 不等）。
	if rec := earnPost(t, mux, set, body, "https://evil.example", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("跨源应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	// scheme 非 http/https=拒。
	if rec := earnPost(t, mux, set, body, "chrome-extension://example.com", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("坏 scheme 应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	// Origin 不可解析/缺 host=拒。
	if rec := earnPost(t, mux, set, body, "notaurl", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("坏 Origin 应 403: code=%d body=%s", rec.Code, rec.Body)
	}
	// 上面全拒——键应仍=播种真值 true（零误写）。
	if on, err := cfg.GetBool("earn.auto_purchase_enabled"); err != nil || !on {
		t.Fatalf("全拒后键应仍 true: on=%v err=%v", on, err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='earn_auto_purchase'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("全拒后审计行应零: n=%d err=%v", n, err)
	}
}

func TestEarnAutoPurchaseWriteAndReadback(t *testing.T) {
	mux, cfg, pool, ctx := earnFaceMux(t)
	const set = "/o.dashboard.v1.CoreService/SetEarnAutoPurchase"
	const get = "/o.dashboard.v1.CoreService/GetEarnAutoPurchase"
	// 同源写=放行：Origin http://example.com == Host example.com。
	rec := earnPost(t, mux, set, `{"enabled":false,"reason":"业主前端手动翻位"}`,
		"http://example.com", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("同源写应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var out struct {
		Enabled   bool   `json:"enabled"`
		UpdatedBy string `json:"updatedBy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("写回包解析败: %v body=%s", err, rec.Body)
	}
	if out.Enabled || out.UpdatedBy != "dashboard-ui" {
		t.Fatalf("写回包应 enabled=false by=dashboard-ui: %+v", out)
	}
	// 库面审计列留痕（钦定锚②）。
	var by string
	if err := pool.QueryRow(ctx,
		`SELECT updated_by FROM app_config WHERE key='earn.auto_purchase_enabled'`).Scan(&by); err != nil {
		t.Fatal(err)
	}
	if by != "dashboard-ui" {
		t.Fatalf("updated_by 应 dashboard-ui: %q", by)
	}
	// ops_event 翻位审计行（reason 落 note）。
	var note string
	if err := pool.QueryRow(ctx,
		`SELECT note FROM ops_event WHERE event='earn_auto_purchase' ORDER BY id DESC LIMIT 1`).Scan(&note); err != nil {
		t.Fatalf("ops_event 翻位行缺: %v", err)
	}
	if !strings.Contains(note, "业主前端手动翻位") {
		t.Fatalf("审计 note 应含 reason: %q", note)
	}
	// cfg 热载后 GetBool=false（worker 闸真值同径）。
	if err := cfg.Load(ctx); err != nil {
		t.Fatal(err)
	}
	if on, err := cfg.GetBool("earn.auto_purchase_enabled"); err != nil || on {
		t.Fatalf("写后热读应 false: on=%v err=%v", on, err)
	}
	// GetEarnAutoPurchase 回读真态——enabled=false+updated_by+note 三件齐。
	rec = earnPost(t, mux, get, `{}`, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("读面应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var rd struct {
		Enabled   bool   `json:"enabled"`
		UpdatedBy string `json:"updatedBy"`
		Note      string `json:"note"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rd); err != nil {
		t.Fatalf("读包解析败: %v body=%s", err, rec.Body)
	}
	if rd.Enabled || rd.UpdatedBy != "dashboard-ui" || !strings.Contains(rd.Note, "停用") {
		t.Fatalf("回读应停用显形+审计痕: %+v", rd)
	}
	// 同源恢复=true（往返可翻——开关非单态闸）。
	rec = earnPost(t, mux, set, `{"enabled":true,"reason":"桩测恢复"}`,
		"https://example.com", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("同源恢复应 200: code=%d body=%s", rec.Code, rec.Body)
	}
}

// 缺键=enabled=false fail-closed 显形（不假开——worker 同口径）。
func TestEarnAutoPurchaseAbsentKey(t *testing.T) {
	mux, _, pool, ctx := earnFaceMux(t)
	if _, err := pool.Exec(ctx,
		"DELETE FROM app_config WHERE key='earn.auto_purchase_enabled'"); err != nil {
		t.Fatal(err)
	}
	rec := earnPost(t, mux, "/o.dashboard.v1.CoreService/GetEarnAutoPurchase", `{}`, "", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("缺键读面应 200: code=%d body=%s", rec.Code, rec.Body)
	}
	var rd struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &rd); err != nil || rd.Enabled {
		t.Fatalf("缺键应 enabled=false: %+v err=%v", rd, err)
	}
}
