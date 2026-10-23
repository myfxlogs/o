// Package okxws —— OKX 公开行情 WS 采集器（数据域首岗；员工「行情巡」载体）。
// 纪律面：流为唯一正道——WS 推流入库，禁 REST 轮询补行情；读循环零同步 DB 写
// （异步批量队列）；断线重连+books5 seqId 序列完整性判据；标的热启自 app_config。
package okxws

import (
	"encoding/json"
	"time"
)

// row 入库行族（写入协程按类型分桶批量 insert）。
type tickRow struct {
	instID string
	ts     time.Time
	last   float64
	bid    float64
	ask    float64
	vol24h float64
}
type booksRow struct {
	instID   string
	ts       time.Time
	bids     json.RawMessage
	asks     json.RawMessage
	seqID    int64
	checksum int64
	hasSeq   bool
}
type fundingRow struct {
	instID        string
	ts            time.Time
	fundingRate   float64
	nextFundingTs time.Time
}
type markRow struct {
	instID  string
	channel string
	ts      time.Time
	px      float64
}
type rowItem struct {
	tick    *tickRow
	books   *booksRow
	funding *fundingRow
	mark    *markRow
}
