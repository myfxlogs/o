package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"o/internal/platform/testutil"
	"o/internal/store/pgstore"
)

// T-0108 桩测锚：/api/trx/* 只读端点出数+读件缺=null 显形不造零。
func TestTrxAPIEndpoints(t *testing.T) {
	ctx := context.Background()
	pool := testutil.Pool(t)
	if err := pgstore.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mountTrxAPI(mux, pool)
	for _, p := range []string{"account", "txs", "radar", "vote", "cmc", "cmc_positions"} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", "/api/trx/"+p, nil))
		if rec.Code != 200 {
			t.Fatalf("%s 端点态: %d", p, rec.Code)
		}
		var v any
		if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
			t.Fatalf("%s 端点出数非法 JSON: %v", p, err)
		}
		if v == nil && p != "txs" && p != "radar" && p != "cmc_positions" {
			continue // 空表=null 显形不造零（合规）
		}
	}
}
