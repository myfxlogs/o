// novelty_esc.go —— 惊异唤面（「知道自己不知道」器官首探——新面进管道
// 时主动想）。新奇度半翼：fin_opp_hypothesis 是 insert-only 卡表
// （UNIQUE(venue,surface)）——新行=假想官起草了从没卡过的面=生成面
// 扩张的物证，唤会话看一眼再走 decide 窄门。
// 不确定度半翼暂缓：无置信列可测（self_model 是帧文非结构列），
// 信念-兑现分叉已由 calib_drift 探管着——不建第二张对账面。
package main

import (
	"context"
	"fmt"
	"time"
)

// noveltyEscCheck 新奇度探：24h 内新增假想卡（state='proposed' 未裁）
// →升件（class='novelty' scope='novelty_<venue>/<surface>'，
// 一卡一 scope 各火各销；fp=卡 id——销后同卡不重火，30d 纹期后
// 同名再卡=新案照火）。无新卡=常静默（本探不报族谱面净态——
// 生成面停滞归 oppSurfaceStallEscCheck 管，不叠探）。
func (c coreService) noveltyEscCheck(ctx context.Context, deadSince map[string]time.Time) {
	rows, err := c.pool.Query(ctx, `SELECT id,venue,surface,family
		FROM fin_opp_hypothesis
		WHERE ts>now()-interval '24 hours' AND state='proposed'
		ORDER BY id DESC LIMIT 10`)
	if err != nil {
		return // 读件败本拍让位
	}
	defer rows.Close()
	type nv struct {
		id                     int64
		venue, surface, family string
	}
	var news []nv
	for rows.Next() {
		var v nv
		if rows.Scan(&v.id, &v.venue, &v.surface, &v.family) == nil {
			news = append(news, v)
		}
	}
	for _, v := range news {
		vv := v
		c.escEdgeUV(ctx, deadSince, "novelty", vv.venue+"/"+vv.surface,
			fmt.Sprint(vv.id), true, func() string {
				return fmt.Sprintf(
					"novelty: 假想卡 #%d %s/%s(family=%s)进管道——O 踏进没踩过的面:裁 adopt/park/dismiss 归 decide 窄门;禁裸销",
					vv.id, vv.venue, vv.surface, vv.family)
			})
	}
}
