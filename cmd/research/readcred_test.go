// readcred_test.go —— T-0170 dev 端：fetchReadcred 形面/isAuthErr 判据/
// authClient 401 自重取（治愈/重取败原错上抛/非认证错不触发重取）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"o/internal/trading/okxexec"
)

// 锚①：fetchReadcred 200 JSON→三件套映射。
func TestFetchReadcredOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"api_key":"k","api_secret":"s","passphrase":"p"}`))
	}))
	defer srv.Close()
	c, err := fetchReadcred(context.Background(), srv.Client(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if c.Key != "k" || c.Secret != "s" || c.Passphrase != "p" {
		t.Fatalf("creds=%+v", c)
	}
}

// 锚②：缺件/非 200/坏形→错（fail-closed 不造空凭据）。
func TestFetchReadcredFail(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{"缺件", `{"api_key":"k","api_secret":""}`, 200},
		{"503", "fail-closed", 503},
		{"坏形", `not-json`, 200},
	} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte(tc.body))
		}))
		if _, err := fetchReadcred(context.Background(), srv.Client(), srv.URL); err == nil {
			t.Fatalf("%s: 应错", tc.name)
		}
		srv.Close()
	}
}

// 锚③：isAuthErr 判据——HTTP 401 串/信封 50103 真；他码/他错/nil 假。
func TestIsAuthErr(t *testing.T) {
	if !isAuthErr(fmt.Errorf("http 401: {\"code\":\"50103\"}")) {
		t.Fatal("http 401 应真")
	}
	if !isAuthErr(&okxexec.APIError{Code: "50103", Msg: "unauth"}) {
		t.Fatal("APIError 50103 应真")
	}
	for _, e := range []error{nil, fmt.Errorf("http 500: x"), &okxexec.APIError{Code: "50011"}, fmt.Errorf("ctx canceled")} {
		if isAuthErr(e) {
			t.Fatalf("%v 应假", e)
		}
	}
}

// fakeGetter 凭据件假体（ok=认证过）；makeCli 按钥名判「轮换后新件」。
type fakeGetter struct{ ok bool }

func (f fakeGetter) Get(ctx context.Context, path string) (json.RawMessage, error) {
	if f.ok {
		return json.RawMessage(`["x"]`), nil
	}
	return nil, fmt.Errorf("http 401: unauthorized")
}

func newTestAuthClient(srv *httptest.Server) *authClient {
	a := &authClient{url: srv.URL, httpc: srv.Client(), cli: fakeGetter{}}
	a.makeCli = func(c okxexec.Credentials) authGetter {
		if c.Key == "rotated" {
			return fakeGetter{ok: true}
		}
		return fakeGetter{}
	}
	return a
}

// 锚④：401→重取一次→换件重试治愈（轮换自愈）。
func TestAuthClientRefetchHeals(t *testing.T) {
	var fetches atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		_, _ = w.Write([]byte(`{"api_key":"rotated","api_secret":"s","passphrase":"p"}`))
	}))
	defer srv.Close()
	a := newTestAuthClient(srv)
	raw, err := a.Get(context.Background(), "/api/v5/sfp/dcd/currency-pair")
	if err != nil {
		t.Fatalf("自愈失败: %v", err)
	}
	if string(raw) != `["x"]` {
		t.Fatalf("raw=%s", raw)
	}
	if fetches.Load() != 1 {
		t.Fatalf("fetches=%d want 1", fetches.Load())
	}
}

// 锚⑤：重取败→原 401 上抛 fail-closed（与无凭据现态同形让位）。
func TestAuthClientRefetchFailKeepsErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	a := newTestAuthClient(srv)
	_, err := a.Get(context.Background(), "/x")
	if err == nil || !isAuthErr(err) {
		t.Fatalf("应原 401 上抛，得 %v", err)
	}
}

// 锚⑥：非认证错不触发重取（透传 fail-closed——零 readcred 流量）。
func TestAuthClientNoRefetchOnOtherErr(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("不应发起取")
	}))
	defer srv.Close()
	a := newTestAuthClient(srv)
	a.cli = errGetter{}
	_, err := a.Get(context.Background(), "/x")
	if err == nil || err.Error() != "http 500: x" {
		t.Fatalf("应透传原错，得 %v", err)
	}
}

type errGetter struct{}

func (errGetter) Get(ctx context.Context, path string) (json.RawMessage, error) {
	return nil, fmt.Errorf("http 500: x")
}
