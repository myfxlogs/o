package okxexec

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"o/internal/platform/signutil"
)

func TestSignVector(t *testing.T) {
	// 签名串序=ts+method+path+body（okx-exec skill 钉序）。
	got := signutil.OkxSign("sec", "2026-01-01T00:00:00.000Z", "GET", "/api/v5/x?a=1", "")
	if got == "" || len(got) < 20 {
		t.Fatalf("sign: %q", got)
	}
	got2 := signutil.OkxSign("sec", "2026-01-01T00:00:00.000Z", "GET", "/api/v5/x?a=1", "b")
	if got == got2 {
		t.Fatal("body 不入签名=序错")
	}
}

// fake API 应答器：data 数组按 state 返回。
func fakeAPI(t *testing.T, code, state string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("OK-ACCESS-KEY") == "" || r.Header.Get("OK-ACCESS-SIGN") == "" {
			w.WriteHeader(401)
			return
		}
		var rows []any
		if state != "" {
			rows = []any{map[string]any{"state": state, "ordId": "1"}}
		}
		b, _ := json.Marshal(map[string]any{"code": code, "msg": "", "data": rows})
		w.Write(b)
	}))
}

func withBase(c *Client, u string) *Client { c.base = u; return c }

func TestResolveBranches(t *testing.T) {
	ctx := context.Background()
	cred := Credentials{Key: "k", Secret: "s", Passphrase: "p"}
	for state, want := range map[string]OrderState{
		"filled": StateFilled, "live": StateLive, "partially_filled": StatePartial, "": StateAbsent} {
		srv := fakeAPI(t, "0", state)
		cli := withBase(NewClient(cred, false), srv.URL)
		got, err := cli.ResolveOrder(ctx, "BTC-USDT-SWAP", "c1")
		srv.Close()
		if err != nil || got != want {
			t.Fatalf("state=%q got=%v err=%v want=%v", state, got, err, want)
		}
	}
}

func TestResolveUnknownOnQueryFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{}, false), srv.URL)
	st, err := cli.ResolveOrder(context.Background(), "X", "c1")
	if st != StateUnknown || err == nil {
		t.Fatalf("查询败应 unknown+err（禁补单径）got=%v err=%v", st, err)
	}
}

func TestSimHeader(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = r.Header.Get("x-simulated-trading") == "1"
		w.Write([]byte(`{"code":"0","data":[]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, true), srv.URL)
	_, _ = cli.Get(context.Background(), "/api/v5/account/balance")
	if !hit {
		t.Fatal("demo 头未挂")
	}
}

func TestPlaceOrderTimeoutIsTransportErr(t *testing.T) {
	cli := withBase(NewClient(Credentials{}, false), "http://10.255.255.1:1")
	cli.hc.Timeout = 300 * time.Millisecond
	_, err := cli.PlaceOrder(context.Background(), OrderParams{InstID: "X", TdMode: "cash", Side: "buy", OrdType: "market", Sz: "1", ClOrdID: "c1"})
	if err == nil || !strings.Contains(err.Error(), "httpStatusError") {
		t.Fatalf("传输败应标 httpStatusError 供定态分流: %v", err)
	}
}

// T-0040 锚：51603 Order does not exist=确定单未建→absent（非查询败）。
func TestResolve51603Absent(t *testing.T) {
	srv := fakeAPI(t, "51603", "")
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	st, err := cli.ResolveOrder(context.Background(), "BTC-USDT-SWAP", "ph1x1x1")
	if err != nil || st != StateAbsent {
		t.Fatalf("51603 应 absent 无 err（同 clOrdId 可重试）got=%v err=%v", st, err)
	}
}

// T-0040 锚：posMode 实读（long_short_mode）；空值 fail-closed。
func TestAccountPosMode(t *testing.T) {
	ctx := context.Background()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","data":[{"posMode":"long_short_mode","acctLv":"2"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	if pm, err := cli.AccountPosMode(ctx); err != nil || pm != "long_short_mode" {
		t.Fatalf("posMode 实读: got=%q err=%v", pm, err)
	}
	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","data":[{"acctLv":"2"}]}`))
	}))
	defer empty.Close()
	if _, err := withBase(NewClient(Credentials{}, false), empty.URL).AccountPosMode(ctx); err == nil {
		t.Fatal("posMode 空值应 fail-closed 上抛")
	}
}

// T-0101 锚：dcd 族信封 code=数字型（非字符串）——Get 双型兼容拒收 200 真数据。
func TestEnvelopeNumericCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":0,"msg":"","data":[{"ordId":"1","state":"LIVE"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	raw, err := cli.Get(context.Background(), "/api/v5/finance/sfp/dcd/order-history")
	if err != nil || !strings.Contains(string(raw), "11448262") && !strings.Contains(string(raw), "ordId") {
		t.Fatalf("数字 code 信封误杀: raw=%s err=%v", raw, err)
	}
}
