package okxexec

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// T-0044 锚：bills 拉取解析+after 游标透传+缺 billId 拒收。
func TestBillsFetch(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"code":"0","data":[
		  {"billId":"991","subType":"174","instId":"BTC-USDT-SWAP","instType":"SWAP",
		   "ccy":"USDT","ts":"1770000000000","balChg":"1.23","fee":"0","pnl":"1.23","sz":"1","px":"50000"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	bills, err := cli.Bills(context.Background(), "SWAP", "991")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotQuery, "instType=SWAP") || !strings.Contains(gotQuery, "after=991") {
		t.Fatalf("query 缺参: %q", gotQuery)
	}
	if len(bills) != 1 || bills[0].BillID != "991" || bills[0].SubType != "174" ||
		bills[0].BalChg != 1.23 || bills[0].TsMs != 1770000000000 {
		t.Fatalf("解析错: %+v", bills[0])
	}
}

func TestBillsMissingBillIDRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":"0","data":[{"balChg":"1"}]}`))
	}))
	defer srv.Close()
	cli := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	if _, err := cli.Bills(context.Background(), "SWAP", ""); err == nil {
		t.Fatal("缺 billId 应拒收（禁无名账单入账）")
	}
}
