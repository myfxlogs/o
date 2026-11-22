// Package okxws —— OKX 公开行情 WS 采集器（数据域首岗；员工「行情巡」载体）。
// 纪律面：流为唯一正道——WS 推流入库，禁 REST 轮询补行情；读循环零同步 DB 写
// （异步批量队列）；断线重连；标的热启自 app_config。T-0123：books5 订阅撤除
// （零读者实锤——五档快照落库 1.8GB/日纯烧）。T-0401 钦定例外：scan.go 粗扫
// REST 轮询=发现面筛选层（非执行价源，md_ticks 执行精度仍归 WS 流）。
package okxws

import (
	"time"
)

// row 入库行族（写入协程按类型分桶批量 upsert/insert）。
type tickRow struct {
	instID string
	ts     time.Time
	last   float64
	bid    float64
	ask    float64
	vol24h float64
}
type fundingRow struct {
	instID          string
	ts              time.Time
	fundingRate     float64
	nextFundingTs   time.Time
	nextFundingRate float64
}
type markRow struct {
	instID  string
	channel string
	ts      time.Time
	px      float64
}
type rowItem struct {
	tick    *tickRow
	funding *fundingRow
	mark    *markRow
	depth5  *depth5Row
	kline   *klineRow
	optSum  *optSumRow  // O-32540：opt-summary 希腊字/IV 最新态
	optTick *optTickRow // O-32540：期权活跃链 ticker 最新态
}

// optSumRow opt-summary 频道行（O-32540——instFamily 增量推逐行；
// 数值字段 *float64：venue 空串落 NULL 不落 0，缺席≠真值0 标缺律）。
type optSumRow struct {
	instID   string
	ts       time.Time
	delta    *float64
	gamma    *float64
	theta    *float64
	vega     *float64
	markVol  *float64 // 期权 IV
	bidVol   *float64
	askVol   *float64
	volLv    *float64
	fwdPx    *float64
	distance *float64
	buyApr   *float64
	sellApr  *float64
	lever    *float64
}

// optTickRow 期权 tickers 频道行（O-32540——活跃子集 instId 键；
// last/bid/ask 空串=无成交/无报价落 NULL 非 0）。
type optTickRow struct {
	instID string
	ts     time.Time
	last   *float64
	lastSz *float64
	bidPx  *float64
	bidSz  *float64
	askPx  *float64
	askSz  *float64
	vol24h *float64
}

// klineRow candle* 频道 K 线行（工单#13065 日K数据面——confirm=false 为
// 形成中帧随 tick 覆写；confirm=true 收线帧终态必落）。bar=周期名（1D 首发）。
type klineRow struct {
	instID  string
	bar     string
	ts      time.Time
	o       float64
	h       float64
	l       float64
	c       float64
	vol     float64
	volCcy  float64
	confirm bool
}

// depth5Row books5 蒸馏特征行（T-0139/D-0022：特征值非原始盘口——top5
// 深度 USD 额+spread_bps upsert md_signal kind='depth5'；桥面轻量非全宇宙）。
type depth5Row struct {
	instID   string
	ts       time.Time
	bidUSD   []float64 // top5 bid 档 USD 额（px×sz×ctv——SPOT ctv=1）
	askUSD   []float64
	bestBid  float64
	bestAsk  float64
	spreadBP float64
}
