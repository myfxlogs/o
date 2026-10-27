package okxws

import (
	"encoding/json"
	"testing"
)

func msg(t *testing.T, channel, instID string, data map[string]any) *wsMsg {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{"arg": map[string]string{"channel": channel, "instId": instID}, "data": []any{data}})
	var m wsMsg
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	return &m
}

func TestUSDTGate(t *testing.T) {
	for _, s := range []string{"BTC-USDT", "ETH-USDT-SWAP"} {
		if !isUSDTInst(s) {
			t.Fatalf("%s 应过 USDT 闸", s)
		}
	}
	for _, s := range []string{"BTC-USDC-SWAP", "ETH-EUR", "BTCUSD"} {
		if isUSDTInst(s) {
			t.Fatalf("%s 应拒（杂币不采）", s)
		}
	}
}

func TestSubscribeArgs(t *testing.T) {
	args := subscribeArgs([]string{"BTC-USDT-SWAP", "ETH-USDT"})
	got := map[string]int{}
	for _, a := range args {
		got[a["channel"]]++
	}
	want := map[string]int{"funding-rate": 1} // T-0171：REST 替已退役——SWAP 订阅仍只 funding-rate，现价走活跃集 tickers
	for ch, n := range want {
		if got[ch] != n {
			t.Fatalf("channel %s=%d want %d（args=%v）", ch, got[ch], n, args)
		}
	}
	if got["tickers"] != 0 {
		t.Fatalf("tickers 应退出宇宙订阅（活跃集动态加订）: %v", args)
	}
}

func TestParseTicks(t *testing.T) {
	m := msg(t, "tickers", "BTC-USDT-SWAP", map[string]any{
		"last": "67000.1", "bidPx": "67000", "askPx": "67000.5", "vol24h": "1234", "ts": "1761000000000"})
	r := parseTickers(m.Arg.InstID, m.Data[0])
	if r == nil || r.bid != 67000 || r.ask != 67000.5 || r.ts.UnixMilli() != 1761000000000 {
		t.Fatalf("tick parse: %+v", r)
	}
	bad := msg(t, "tickers", "BTC-USDT-SWAP", map[string]any{"bidPx": "0", "askPx": "0"})
	if r := parseTickers(bad.Arg.InstID, bad.Data[0]); r != nil {
		t.Fatal("零价帧应拒")
	}
}

func TestParseFunding(t *testing.T) {
	m := msg(t, "funding-rate", "BTC-USDT-SWAP", map[string]any{
		"fundingRate": "0.0001", "fundingTime": "1761000000000", "nextFundingTime": "1761003600000"})
	r := parseFunding(m.Arg.InstID, m.Data[0])
	if r == nil || r.fundingRate != 0.0001 || r.nextFundingTs.UnixMilli() != 1761003600000 {
		t.Fatalf("funding parse: %+v", r)
	}
}

func TestParseMarkBothForms(t *testing.T) {
	m1 := msg(t, "mark-price", "BTC-USDT-SWAP", map[string]any{"markPx": "67000", "ts": "1761000000000"})
	if r := parseMark(m1.Arg.InstID, "mark-price", m1.Data[0]); r == nil || r.px != 67000 {
		t.Fatalf("mark-price parse: %+v", r)
	}
	m2 := msg(t, "index-tickers", "BTC-USDT", map[string]any{"idxPx": "66999", "ts": "1761000000000"})
	if r := parseMark(m2.Arg.InstID, "index-tickers", m2.Data[0]); r == nil || r.px != 66999 {
		t.Fatalf("index-tickers parse: %+v", r)
	}
}

// T-0139 books5 蒸馏解析锚：USD 额=px×sz×ctv、spread_bps、截断 top5。
func TestParseBooks5(t *testing.T) {
	d := map[string]any{
		"ts": "1700000000000",
		"bids": []any{
			[]any{"100", "2"}, []any{"99", "1"}, []any{"98", "1"},
			[]any{"97", "1"}, []any{"96", "1"}, []any{"95", "1"}, // 第 6 档截断
		},
		"asks": []any{
			[]any{"101", "1"}, []any{"102", "1"},
		},
	}
	raw, _ := json.Marshal(d)
	var m map[string]json.RawMessage
	json.Unmarshal(raw, &m)
	r := parseBooks5("BTC-USDT-SWAP", m, 0.01)
	if r == nil {
		t.Fatal("应解析成行")
	}
	if r.bestBid != 100 || r.bestAsk != 101 {
		t.Fatalf("best=%v/%v", r.bestBid, r.bestAsk)
	}
	if len(r.bidUSD) != 5 {
		t.Fatalf("bid 档应截断 top5 got %d", len(r.bidUSD))
	}
	if r.bidUSD[0] != 100*2*0.01 {
		t.Fatalf("USD 额=px×sz×ctv got %v", r.bidUSD[0])
	}
	want := float64(101-100) / 100.5 * 10000 // 中点=(101+100)/2=100.5
	if r.spreadBP < want-0.01 || r.spreadBP > want+0.01 {
		t.Fatalf("spread_bps=%v want≈%v", r.spreadBP, want)
	}
	if parseBooks5("BTC-USDT-SWAP", m, 0) != nil {
		t.Fatal("缺 ctv 应拒入如实（nil）")
	}
}
