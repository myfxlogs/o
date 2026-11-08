// open_dcd_reject.go —— T-0259/T-0461 dcd 三步缝注入位：实现体已抽立
// internal/trading/dcdexec（产线 hedgeOpener.dcdOpen 与 simvenue 矩阵
// DCD 演习格同径消费——venue IO 之外全产线同码）。本件仅余缝别名+
// 装配取件——拒因持久化面（fin_eval_reject/fin_execution.venue_err_*）
// 语义随 dcdexec.VenueErr/Mark/MarkVE/QuoteReject 原样在案。
package main

import (
	"o/internal/trading/dcdexec"
)

// dcdVenue dcd 执行三步缝别名（T-0259 桩测注伪位——产线件=*okxexec.Client
// 同签名实现；nil 回退 cli 即产线形，装配面零改）。
type dcdVenue = dcdexec.Venue

func (h hedgeOpener) dcdCli() dcdVenue {
	if h.dcd != nil {
		return h.dcd
	}
	return h.cli
}
