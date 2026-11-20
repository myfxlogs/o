package okxgrid

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// 桩传输面——请求形钉死+信封 data 段回放（签名/信封=okxexec 域已测，
// 本件测字段映射与请求体构形）。

type stubDoer struct {
	getRaw  json.RawMessage
	postRaw json.RawMessage
	getErr  error
	postErr error
	gotGet  []string
	gotPost []struct{ path, body string }
}

func (s *stubDoer) Get(_ context.Context, p string) (json.RawMessage, error) {
	s.gotGet = append(s.gotGet, p)
	return s.getRaw, s.getErr
}
func (s *stubDoer) Post(_ context.Context, p string, b []byte) (json.RawMessage, error) {
	s.gotPost = append(s.gotPost, struct{ path, body string }{p, string(b)})
	return s.postRaw, s.postErr
}

func TestLaunchBodyShape(t *testing.T) {
	st := &stubDoer{postRaw: json.RawMessage(`[{"algoId":"447053782921515008","sCode":"0","sMsg":"","algoClOrdId":"og7","tag":"okxgrid"}]`)}
	ad := NewAdapter(st)
	id, err := ad.Launch(context.Background(), LaunchReq{
		InstID: "BTC-USDT", MaxPx: 70946.7, MinPx: 52351.4,
		GridNum: 20, QuoteSz: 50, AlgoClID: "og7", Tag: "okxgrid",
	})
	if err != nil || id != "447053782921515008" {
		t.Fatalf("launch: id=%q err=%v", id, err)
	}
	if len(st.gotPost) != 1 || st.gotPost[0].path != "/api/v5/tradingBot/grid/order-algo" {
		t.Fatalf("post 面不符: %+v", st.gotPost)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(st.gotPost[0].body), &body); err != nil {
		t.Fatal(err)
	}
	// venue 数值串方言钉（取证在案——全字段 string 形）。
	for k, want := range map[string]string{
		"instId": "BTC-USDT", "algoOrdType": "grid", "maxPx": "70946.7",
		"minPx": "52351.4", "gridNum": "20", "runType": "1", "quoteSz": "50",
	} {
		if body[k] != want {
			t.Fatalf("body[%s]=%v want %q", k, body[k], want)
		}
	}
}

func TestStopBodyShape(t *testing.T) {
	st := &stubDoer{postRaw: json.RawMessage(`[{"algoId":"1","sCode":"0"}]`)}
	ad := NewAdapter(st)
	if err := ad.Stop(context.Background(), "1", "BTC-USDT", "1"); err != nil {
		t.Fatal(err)
	}
	if st.gotPost[0].path != "/api/v5/tradingBot/grid/stop-order-algo" {
		t.Fatalf("path=%s", st.gotPost[0].path)
	}
	var arr []map[string]string
	if err := json.Unmarshal([]byte(st.gotPost[0].body), &arr); err != nil || len(arr) != 1 {
		t.Fatalf("body 非批形: %s", st.gotPost[0].body)
	}
	if arr[0]["stopType"] != "1" || arr[0]["algoOrdType"] != "grid" {
		t.Fatalf("body=%v", arr[0])
	}
}

func TestDetailsParse(t *testing.T) {
	st := &stubDoer{getRaw: json.RawMessage(`[{"algoId":"448965992920907776","algoOrdType":"grid",
		"state":"running","investment":"25","gridProfit":"1.23","floatProfit":"0.5",
		"totalPnl":"1.73","tradeNum":"14","arbitrageNum":"7","runPx":"30089.7",
		"gridNum":"10","maxPx":"5000","minPx":"400","instId":"BTC-USDT"}]`)}
	ad := NewAdapter(st)
	d, err := ad.Details(context.Background(), "448965992920907776")
	if err != nil {
		t.Fatal(err)
	}
	if d.State != "running" || parseF(d.GridProfit) != 1.23 || parseF(d.ArbitrageNum) != 7 {
		t.Fatalf("details=%+v", d)
	}
	if !strings.Contains(st.gotGet[0], "algoOrdType=grid") {
		t.Fatalf("query=%s", st.gotGet[0])
	}
}

func TestDetailsEmptyFailClosed(t *testing.T) {
	st := &stubDoer{getRaw: json.RawMessage(`[]`)}
	if _, err := NewAdapter(st).Details(context.Background(), "x"); err == nil {
		t.Fatal("空 details 未拒")
	}
}

func TestWithdrawIncome(t *testing.T) {
	st := &stubDoer{postRaw: json.RawMessage(`[{"algoId":"1","profit":"100"}]`)}
	p, err := NewAdapter(st).WithdrawIncome(context.Background(), "1")
	if err != nil || p != 100 {
		t.Fatalf("profit=%v err=%v", p, err)
	}
}
