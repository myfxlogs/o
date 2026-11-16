// metric_beats.go —— 量表探群（非升件量表拍——feed_esc.go 棘轮线满,
// 量表拍群整件落此,语义/调用序与 feed_esc 原挂点同）。
package main

import (
	"context"
	"time"
)

// metricBeats 量表拍群——非升件件（量表帧/日行照/账房对账/器官帧),
// 与升件探同拍同序,挂点=feed 拍环末位。
func (c coreService) metricBeats(ctx context.Context, deadSince map[string]time.Time) {
	c.armedDistanceBeat(ctx)         // 量表探（非升件——armed 距阈日行照）
	c.ownerLedgerBeat(ctx)           // 量表探（业主令账+点破率曲线日行照）
	c.ownerModelBeat(ctx, deadSince) // 懂你模型帧(棒4——请托结局谱∪点破域谱+reask_denied 探)
	c.profitDailyBeat(ctx)           // 量表探（盈利报告日行照——业主唯一例行读件）
	c.selfStateBeat(ctx)             // 量表探（自态账——会话冷启动即读件,工单#10358 C）
	c.booksBeat(ctx, deadSince)      // 账房（ΔNAV三桶对账+鬼账升件,业主令11-12立件）
	c.pnlAttrBeat(ctx, deadSince)    // 策略盈亏归因（11-12①——分列+账房咬合,账房器官零改动）
	c.selfAwareBeat(ctx)             // 自我意识器官帧（工单#13540 业主令件——本体觉三问量表）
	c.regimeBeat(ctx)                // 世模环境帧（棒3——市场态量表,世面=冷启动即见件）
	c.seatKpiBeat(ctx)               // 各岗KPI量表（工单#19121——唤知钦定八口径日行照）
}
