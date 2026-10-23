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
	want := map[string]int{"tickers": 2, "books5": 2, "funding-rate": 1, "mark-price": 1, "index-tickers": 1}
	for ch, n := range want {
		if got[ch] != n {
			t.Fatalf("channel %s=%d want %d（args=%v）", ch, got[ch], n, args)
		}
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

func TestParseBooks5Seq(t *testing.T) {
	m := msg(t, "books5", "BTC-USDT-SWAP", map[string]any{
		"bids": [][]string{{"1", "2"}}, "asks": [][]string{{"3", "4"}},
		"seqId": "42", "checksum": "7", "ts": "1761000000000"})
	r := parseBooks5(m.Arg.InstID, m.Data[0])
	if r == nil || r.seqID != 42 || !r.hasSeq || r.checksum != 7 {
		t.Fatalf("books5 parse: %+v", r)
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
