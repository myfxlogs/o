package okxexec

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// retry_test.go —— T-0272④ 锚：私有 REST 读径 429 指数退避——
// 429→200 恢复径达、终败仍 err 上抛不返陈值、非 429 败零重试零新增延迟、
// 写径 429 不重试（mutating 歧态势禁补单语义不破）。

func TestDo429BackoffRecovers(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 { // 前两发 429，第三发放行——限流窗内自愈形
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"code":"0","data":[{"bal":"1"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	if _, err := cli.Get(context.Background(), "/api/v5/asset/balances"); err != nil {
		t.Fatalf("429→200 退避后应达: %v", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("应 3 发（429×2+200）: %d", calls.Load())
	}
}

func TestDo429ExhaustedErr(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	_, err := cli.Get(context.Background(), "/api/v5/asset/balances")
	if err == nil {
		t.Fatal("429 终败应 err 上抛——不假零不返陈值")
	}
	var se *HTTPStatusError
	if !errors.As(err, &se) || se.Code != http.StatusTooManyRequests {
		t.Fatalf("终败应携 HTTPStatusError 429 码位: %v", err)
	}
	if calls.Load() != 4 { // 1 发+≤3 重试
		t.Fatalf("应 4 发（1+3 重试限次）: %d", calls.Load())
	}
}

func TestDoNon429NoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError) // 500 非 429——不重试
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	start := time.Now()
	_, err := cli.Get(context.Background(), "/api/v5/asset/balances")
	if err == nil || calls.Load() != 1 {
		t.Fatalf("非 429 应首发即败零重试: calls=%d err=%v", calls.Load(), err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("非 429 败零新增延迟: %v", time.Since(start))
	}
}

func TestDoPost429NoRetry(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	_, err := cli.Post(context.Background(), "/api/v5/trade/order", []byte("{}"))
	if err == nil || calls.Load() != 1 {
		t.Fatalf("写径 429 应零重试（mutating 歧态势禁补单语义不破）: calls=%d err=%v", calls.Load(), err)
	}
}
