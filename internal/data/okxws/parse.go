package okxws

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// wsMsg OKX v5 推送帧头+载荷（arg.channel 分流；data[]=频道载荷行）。
type wsMsg struct {
	Event string `json:"event"`
	Arg   struct {
		Channel string `json:"channel"`
		InstID  string `json:"instId"`
	} `json:"arg"`
	Data []map[string]json.RawMessage `json:"data"`
}

func str(d map[string]json.RawMessage, k string) string {
	var s string
	if v, ok := d[k]; ok {
		_ = json.Unmarshal(v, &s)
	}
	return s
}

func num(d map[string]json.RawMessage, k string) float64 {
	s := str(d, k)
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func intv(d map[string]json.RawMessage, k string) int64 {
	s := str(d, k)
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// tsms OKX ts 字段=毫秒 epoch 字符串 → UTC 时戳（缺省零值，调用方兜底 recv 时刻）。
func tsms(d map[string]json.RawMessage, k string) time.Time {
	ms := intv(d, k)
	if ms == 0 {
		return time.Time{}
	}
	return time.UnixMilli(ms).UTC()
}

// isUSDTInst USDT 计价闸（业主令：杂币不采不用不计——instId 形如 X-USDT[-SWAP]）。
func isUSDTInst(instID string) bool {
	// T-0189：USD 指数键（BTC-USD——index-tickers uly 键域）+到期期货
	// 形（-\d{6}$ 尾——BTC-USD-261225）放行豁免——「杂币不采」禁的是
	// 杂计价币面，USD 指数/币本位期货系 basis 域钦定数据源非杂币。
	if strings.HasSuffix(instID, "-USD") || datedFutRe.MatchString(instID) {
		return true
	}
	return strings.Contains(instID, "-USDT")
}

// parseTickers → tickRow（last/bid/ask 零值=无效帧拒入）。
func parseTickers(instID string, d map[string]json.RawMessage) *tickRow {
	bid, ask := num(d, "bidPx"), num(d, "askPx")
	if bid == 0 && ask == 0 {
		return nil
	}
	ts := tsms(d, "ts")
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &tickRow{instID: instID, ts: ts, last: num(d, "last"), bid: bid, ask: ask, vol24h: num(d, "vol24h")}
}

// parseFunding → fundingRow。
func parseFunding(instID string, d map[string]json.RawMessage) *fundingRow {
	ts := tsms(d, "fundingTime")
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &fundingRow{instID: instID, ts: ts,
		fundingRate: num(d, "fundingRate"), nextFundingTs: tsms(d, "nextFundingTime"),
		nextFundingRate: num(d, "nextFundingRate")}
}

// parseMark → markRow（mark-price/index-tickers 同形：markPx/idxPx）。
func parseMark(instID, channel string, d map[string]json.RawMessage) *markRow {
	px := num(d, "markPx")
	if px == 0 {
		px = num(d, "idxPx")
	}
	if px == 0 {
		return nil
	}
	ts := tsms(d, "ts")
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &markRow{instID: instID, channel: channel, ts: ts, px: px}
}
