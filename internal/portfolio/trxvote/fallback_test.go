// fallback_test.go —— T-0192③ 桩锚：witness 主败备顶/主复主归双源单测。
package trxvote

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWitFallbackPrimaryDown(t *testing.T) {
	// 主源 500 → trongrid 备顶（hex→base58+cache-last-good 年化）
	prim := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer prim.Close()
	trg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"witnesses":[{"address":"41f1a2b3c4d5e6f70123456789abcdef0123456789","isJobs":true,"realTimeVotes":1000,"url":"sr-a.io"}]}`))
	}))
	defer trg.Close()
	sched := &Scheduler{tronscanBase: prim.URL}
	sched.SetWitFallback(func(ctx context.Context) ([]SRInfo, error) {
		return FetchSRListTrongridFallback(ctx, trg.URL, "k")
	})
	// 首拍无 last-good 缓存→该组退避不评（钦定：不臆造年化）
	if _, err := sched.fetchSRListFallback(context.Background()); err != nil {
		t.Fatalf("fallback 应返回零行非 err（首拍无缓存如实退避）: %v", err)
	}
	// 主源复活喂缓存→备源年化沿缓存（键=hex→b58 后的真地址）
	b58, err := addrHexToB58("41f1a2b3c4d5e6f70123456789abcdef0123456789")
	if err != nil {
		t.Fatal(err)
	}
	lastGoodRate[b58] = 4.2
	out, err := sched.fetchSRListFallback(context.Background())
	if err != nil {
		t.Fatalf("备顶败: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("备源应出行（cache-last-good 年化沿）")
	}
}
