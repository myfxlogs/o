// dcd_test.go —— T-0207 桩锚：quote POST 三参体+quoteId 解析；trade 唯
// quoteId 体（无 clOrdId 幂等键钉死）。
package okxexec

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDcdQuoteTradeDialect(t *testing.T) {
	var gotPaths []string
	var gotBodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotPaths = append(gotPaths, r.Method+" "+r.URL.Path)
		gotBodies = append(gotBodies, string(b))
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/quote") {
			w.Write([]byte(`{"code":"0","data":[{"quoteId":"q-77"}]}`))
			return
		}
		w.Write([]byte(`{"code":"0","data":[{"ordId":"o-1","sCode":"0"}]}`))
	}))
	defer srv.Close()
	c := withBase(NewClient(Credentials{Key: "k", Secret: "s", Passphrase: "p"}, false), srv.URL)
	ctx := t.Context()
	qid, err := c.DcdQuote(ctx, "BTC-USDT-261113-87500-C", "0.01", "BTC")
	if err != nil || qid != "q-77" {
		t.Fatalf("quoteId=%s err=%v", qid, err)
	}
	if _, err := c.DcdTrade(ctx, qid); err != nil {
		t.Fatal(err)
	}
	if len(gotPaths) != 2 || gotPaths[0] != "POST /api/v5/finance/sfp/dcd/quote" ||
		gotPaths[1] != "POST /api/v5/finance/sfp/dcd/trade" {
		t.Fatalf("paths=%v", gotPaths)
	}
	var qb, tb map[string]string
	_ = json.Unmarshal([]byte(gotBodies[0]), &qb)
	_ = json.Unmarshal([]byte(gotBodies[1]), &tb)
	if qb["productId"] == "" || qb["notionalSz"] != "0.01" || qb["notionalCcy"] != "BTC" {
		t.Fatalf("quote 体=%v", qb)
	}
	if len(tb) != 1 || tb["quoteId"] != "q-77" {
		t.Fatalf("trade 体应唯 quoteId: %v", tb)
	}
}
