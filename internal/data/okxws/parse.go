package okxws

import (
	"encoding/json"
	"strconv"
	"time"

	"o/internal/platform/legset"
)

// wsMsg OKX v5 推送帧头+载荷（arg.channel 分流；data[]=频道载荷行——
// 行形频道而异：tickers/funding/mark/books5=对象，candle*=数组，
// 故 Data 只载 RawMessage 由各频道解析面自解）。
type wsMsg struct {
	Event string `json:"event"`
	Arg   struct {
		Channel    string `json:"channel"`
		InstID     string `json:"instId"`
		InstFamily string `json:"instFamily"` // O-32540：opt-summary 订阅键（instId 形无 ack 拒——族键唯一受理面）
	} `json:"arg"`
	Data []json.RawMessage `json:"data"`
}

// obj 对象形载荷行解码（tickers/funding/mark/books5 各频道共用）。
func obj(raw json.RawMessage) map[string]json.RawMessage {
	var d map[string]json.RawMessage
	_ = json.Unmarshal(raw, &d)
	return d
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

// numPtr 可空数值解析（O-32540 期权面——venue 空串/缺字段落 nil→
// NULL 入库，缺席≠真值 0 标缺律；区别于 num() 的 0 缺省语义）。
func numPtr(d map[string]json.RawMessage, k string) *float64 {
	s := str(d, k)
	if s == "" {
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &f
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

// isUSDTInst USDT 计价闸——T-0392 归并 legset.IsUSDTInst 唯一真源
// （订阅闸/覆盖探针/腿派生同判据，本件=包内薄壳委托）。
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

// parseOptSummary → optSumRow（O-32540——opt-summary 增量推行；
// instId 在载荷行内非 arg；ts 缺省回落 recv 时刻）。
func parseOptSummary(d map[string]json.RawMessage) *optSumRow {
	instID := str(d, "instId")
	if instID == "" {
		return nil
	}
	ts := tsms(d, "ts")
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &optSumRow{instID: instID, ts: ts,
		delta: numPtr(d, "delta"), gamma: numPtr(d, "gamma"),
		theta: numPtr(d, "theta"), vega: numPtr(d, "vega"),
		markVol: numPtr(d, "markVol"), bidVol: numPtr(d, "bidVol"),
		askVol: numPtr(d, "askVol"), volLv: numPtr(d, "volLv"),
		fwdPx: numPtr(d, "fwdPx"), distance: numPtr(d, "distance"),
		buyApr: numPtr(d, "buyApr"), sellApr: numPtr(d, "sellApr"),
		lever: numPtr(d, "lever")}
}

// parseOptTickers → optTickRow（O-32540——期权 tickers 行形与
// SWAP 同构但全字段可空：last="" 无成交、bid/ask="" 无报价）。
func parseOptTickers(instID string, d map[string]json.RawMessage) *optTickRow {
	ts := tsms(d, "ts")
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	return &optTickRow{instID: instID, ts: ts,
		last: numPtr(d, "last"), lastSz: numPtr(d, "lastSz"),
		bidPx: numPtr(d, "bidPx"), bidSz: numPtr(d, "bidSz"),
		askPx: numPtr(d, "askPx"), askSz: numPtr(d, "askSz"),
		vol24h: numPtr(d, "vol24h")}
}

// parseLiquidation → []liqRow（O-35587②——liquidation-orders 帧级对象
// {instId,instFamily,instType,uly,details[]}→details 逐条展开；ts=毫秒
// epoch 字符串缺省回落 recv 时刻；空 details/缺 instId=拒入 nil）。
func parseLiquidation(d map[string]json.RawMessage) []liqRow {
	instID := str(d, "instId")
	if instID == "" {
		return nil
	}
	var dets []map[string]json.RawMessage
	if err := json.Unmarshal(d["details"], &dets); err != nil || len(dets) == 0 {
		return nil
	}
	out := make([]liqRow, 0, len(dets))
	for _, det := range dets {
		ts := tsms(det, "ts")
		if ts.IsZero() {
			ts = time.Now().UTC()
		}
		out = append(out, liqRow{instID: instID, instFam: str(d, "instFamily"),
			uly: str(d, "uly"), ts: ts, posSide: str(det, "posSide"),
			side: str(det, "side"), sz: num(det, "sz"), bkPx: num(det, "bkPx"),
			bkLoss: num(det, "bkLoss"), ccy: str(det, "ccy")})
	}
	return out
}

// parseCandle → klineRow（candle* 数组载荷：[ts,o,h,l,c,vol,volCcy,
// volCcyQuote,confirm]——confirm "0"=形成中 "1"=已收线；ts=开盘时刻毫秒）。
func parseCandle(instID, bar string, raw json.RawMessage) *klineRow {
	var a []string
	if err := json.Unmarshal(raw, &a); err != nil || len(a) < 9 {
		return nil
	}
	ms, _ := strconv.ParseInt(a[0], 10, 64)
	if ms == 0 {
		return nil
	}
	f := func(i int) float64 { v, _ := strconv.ParseFloat(a[i], 64); return v }
	return &klineRow{instID: instID, bar: bar, ts: time.UnixMilli(ms).UTC(),
		o: f(1), h: f(2), l: f(3), c: f(4), vol: f(5), volCcy: f(6),
		confirm: a[8] == "1"}
}
