package okxws

import (
	"encoding/json"
	"strconv"
	"time"

	"o/internal/platform/legset"
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

// isUSDTInst USDT 计价闸——T-0394 归并 legset.IsUSDTInst 唯一真源
// （订阅闸/依赖对枚举探针/腿派生同判据，本件=包内薄壳委托）。
func isUSDTInst(instID string) bool { return legset.IsUSDTInst(instID) }

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
