package okxws

import (
	"context"
	"fmt"
	"log"
	"time"
)

// staleWatch 面级断流告警（T-0129）：60s 一拍——订阅标的中 md_ticks
// 全行超龄（>3min 无新鲜行）=行情面死 ALERT 一次，恢复清标可再报；
// 单标的陈≠面死（逐键告警另有 facts_missing 计数面）。
func (c *Collector) staleWatch(ctx context.Context) {
	tick := time.NewTicker(60 * time.Second) // sched-exempt-ok：面级断流看门狗节拍（活性探针非业务轮询）
	defer tick.Stop()
	alerted := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			syms, err := c.symbols(ctx)
			if err != nil || len(syms) == 0 {
				continue
			}
			var fresh int
			if err := c.pool.QueryRow(ctx, `SELECT count(DISTINCT inst_id) FROM md_ticks
				WHERE inst_id = ANY($1) AND ts>now()-interval '3 minutes'`,
				syms).Scan(&fresh); err != nil {
				continue
			}
			if fresh == 0 && !alerted {
				msg := fmt.Sprintf("订阅 %d 标的 md_ticks 全行超龄 3min（断流面非单键陈）", len(syms))
				if c.al != nil {
					kind := c.AlertKind
					if kind == "" {
						kind = "okxws_face_dead"
					}
					c.al.Alert(ctx, kind, msg)
				} else {
					log.Printf("ALERT [okxws] 行情面死——%s", msg)
				}
				alerted = true
			} else if fresh > 0 {
				alerted = false // 恢复清标可再报
			}
		}
	}
}
