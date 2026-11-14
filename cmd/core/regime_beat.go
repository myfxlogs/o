// regime_beat.go —— 世模环境帧量表探（棒3 世界模型扩域：calib 对「我的
// 信念诚实吗」，regime 答「市场现在是什么态」）。md_klines 1D 确认棒
// 宇宙横切：20d 实现波动（年化）中位+14d 动量中位+多空广度→regime
// 标签，值指纹闸落 量表/regime（6h 拍——环境慢变量，同纹抑）。
// 消费面：dispatch self_narrative「世面」字段/独白档读件/方向族条件化
// 食源。数据薄（足窗品种<5）/陈（末确认棒>96h）=thin/stale 如实报
// 不虚构；INSERT 败=本帧让位同量表惯例。
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

const regimeEvery = 6 * time.Hour

// regimeBeat 世模环境帧（escLoop 量表拍载）。宇宙同 dirmom 口径
// （okx *-SWAP 确认 1D 棒）——环境判断与方向信号同一世界切片。
func (c coreService) regimeBeat(ctx context.Context) {
	var n int
	var medVol, medMom, breadth *float64
	var last *time.Time
	err := c.pool.QueryRow(ctx, `WITH b AS (
		SELECT inst_id, ts, c, row_number() OVER w rn, lag(c) OVER w prev
		FROM md_klines
		WHERE venue='okx' AND bar='1D' AND confirm=true AND inst_id LIKE '%-SWAP'
		WINDOW w AS (PARTITION BY inst_id ORDER BY ts DESC)
	), s AS (
		SELECT inst_id,
			stddev_samp(ln(c/prev)) FILTER (WHERE rn BETWEEN 2 AND 21)*sqrt(365) vol,
			ln(max(c) FILTER (WHERE rn=1)/max(c) FILTER (WHERE rn=15)) mom,
			max(ts) last_ts
		FROM b WHERE rn<=21 GROUP BY inst_id
		HAVING count(*)>=21)
	SELECT count(*),
		percentile_cont(0.5) WITHIN GROUP (ORDER BY vol),
		percentile_cont(0.5) WITHIN GROUP (ORDER BY mom),
		avg((mom>0)::int), max(last_ts) FROM s`).
		Scan(&n, &medVol, &medMom, &breadth, &last)
	var label, note string
	switch {
	case err != nil:
		return // 读件败本帧让位
	case last == nil || time.Since(*last) > 96*time.Hour:
		label = "stale"
		note = fmt.Sprintf("regime=stale n=%d（末确认棒陈——数据面死非市场态）", n)
	case n < 5:
		label = "thin"
		note = fmt.Sprintf("regime=thin n=%d（足窗品种<5——宇宙未足不虚构）", n)
	default:
		trend := "flat"
		switch {
		case *medMom > 0.03:
			trend = "up"
		case *medMom < -0.03:
			trend = "down"
		}
		vol := "mid"
		switch {
		case *medVol > 0.8:
			vol = "hi"
		case *medVol < 0.4:
			vol = "lo"
		}
		label = trend + "_" + vol + "vol"
		note = fmt.Sprintf("regime=%s vol=%.2f mom=%+.3f breadth=%.2f n=%d",
			label, *medVol, *medMom, *breadth, n)
	}
	fp := fpHash([]string{label}) // 纹=类标签非值——类稳则抑至 6h,类变=新现实
	if !gaugeDue(ctx, c, "regime", fp, regimeEvery) {
		return
	}
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES('探测席','量表','regime',$1)`,
		"v="+fp+" "+note); err != nil {
		log.Printf("[beat] regime 写败: %v", err)
	}
}
