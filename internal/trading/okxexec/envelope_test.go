// envelope_test.go —— 信封 code 联合型桩测（T-0102）：dcd 族 `{"code":0,...}`
// 数字型不被 string 解析误杀（坑条 okx-dcd-endpoint-dialect）；既有 string 形
// 回归不破；业务拒归一 APIError（code string 形不变）。
package okxexec

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNumericCodeEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/num-ok":
			_, _ = w.Write([]byte(`{"code":0,"msg":"","data":[{"uly":"BTC-USDT"}]}`))
		case "/num-err":
			_, _ = w.Write([]byte(`{"code":51001,"msg":"no permission","data":[]}`))
		default:
			_, _ = w.Write([]byte(`{"code":"0","msg":"","data":[{"uly":"ETH-USDT"}]}`))
		}
	}))
	defer srv.Close()
	c := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	raw, err := c.Get(context.Background(), "/num-ok")
	if err != nil {
		t.Fatalf("数字 code 信封被误杀: %v", err)
	}
	var rows []struct {
		Uly string `json:"uly"`
	}
	if json.Unmarshal(raw, &rows) != nil || len(rows) != 1 || rows[0].Uly != "BTC-USDT" {
		t.Fatalf("data 解析: %s", raw)
	}
	// 既有 string 形回归。
	if _, err := c.Get(context.Background(), "/str-ok"); err != nil {
		t.Fatalf("string code 回归破: %v", err)
	}
	// 业务拒：APIError code 归一 string 形。
	err = nil
	if _, err = c.Get(context.Background(), "/num-err"); err == nil {
		t.Fatal("业务拒应上抛")
	}
	ae, ok := err.(*APIError)
	if !ok || ae.Code != "51001" {
		t.Fatalf("业务拒应 APIError code=51001: %v", err)
	}
}
