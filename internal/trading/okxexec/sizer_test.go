package okxexec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// T-0046 ②锚：平仓量现读面——availBal（availBal 优先/availEq 兜底/ccy 缺拒）
// + positions 净持仓 abs。
func TestSizerReads(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v5/account/balance" {
			w.Write([]byte(`{"code":"0","data":[{"details":[
			  {"ccy":"BTC","availBal":"0.00030866","availEq":"0.000309"},
			  {"ccy":"ETH","availBal":"","availEq":"2.5"}]}]}`))
			return
		}
		w.Write([]byte(`{"code":"0","data":[{"pos":"-3","instId":"BTC-USDT-SWAP"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	ctx := context.Background()
	if v, err := cli.AvailBal(ctx, "BTC"); err != nil || v != 0.00030866 {
		t.Fatalf("availBal 优先: %v err=%v", v, err)
	}
	if v, err := cli.AvailBal(ctx, "ETH"); err != nil || v != 2.5 {
		t.Fatalf("availEq 兜底: %v err=%v", v, err)
	}
	if _, err := cli.AvailBal(ctx, "DOGE"); err == nil {
		t.Fatal("ccy 缺席应 fail-closed")
	}
	if v, err := cli.ContractPosSz(ctx, "BTC-USDT-SWAP"); err != nil || v != 3 {
		t.Fatalf("净持仓 abs: %v err=%v", v, err)
	}
}
