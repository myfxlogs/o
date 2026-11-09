// miner.go —— 教训矿工拍接线（T-0385——同正径不入 hungerLoop 槽）：
// internal/portfolio/hunger 矿工件激活径。独立 ticker 节拍 12h（票面钦
// 慢拍——记忆面非急诊，不挤饥饿 6h 槽亦省协程数面等价）——拍壳语义对形
// hungerLoop：BeatLoop 同款 ctx 界内协程（Run 拉起非野径）、拉起即首挖
// （部署即补当前窗非干等 12h——T-0381 首侦同例）、败=log+升件自报行不
// 拖事件主拍（T-0369 容错必报）。
package lifecycle

import (
	"context"
	"fmt"
	"log"
	"time"
)

// minerBeatEvery 矿工拍周期（var=测试面覆写锚——生产值钦定不动）。
var minerBeatEvery = 12 * time.Hour

// minerLoop 教训矿工拍：hungerLoop 同构 ctx 界内协程——进 ticker 前先
// 跑一次 minerTick（部署即首挖），之后照旧 minerBeatEvery 复拍。
func (r *Runner) minerLoop(ctx context.Context) {
	t := time.NewTicker(minerBeatEvery) // sched-exempt-ok 慢拍记忆面（事件账挖潜非事件径——12h 窗聚类天然期界）
	defer t.Stop()
	r.minerTick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.minerTick(ctx)
		}
	}
}

// minerTick 单拍：MineLessons 挖窗；败情自报不拖主拍——本拍自身终结。
func (r *Runner) minerTick(ctx context.Context) {
	n, err := r.sensor.MineLessons(ctx)
	if err != nil {
		log.Printf("[lifecycle] minertick 败: %v", err)
		r.minerEscalate(ctx, err)
		return
	}
	if n > 0 {
		log.Printf("[lifecycle] minertick 起草工程案 pending ×%d", n)
	}
}

// minerEscalate 败情自报升件行（hungerEscalate 同构——写败/LIFECYCLE-MINER
// 判例键，note 前缀同窗 24h 至多一行；自报行自身写败=静默退）。
func (r *Runner) minerEscalate(ctx context.Context, cause error) {
	if _, err := r.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'写败','LIFECYCLE-MINER'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE $4 AND ts>now()-interval '24 hours')`,
		hungerSeat, "", fmt.Sprintf("minertick 败: %v", cause),
		"minertick 败%"); err != nil {
		log.Printf("[lifecycle] minertick 自报行落败（结构性限度静默退）: %v（原败: %v）", err, cause)
	}
}
