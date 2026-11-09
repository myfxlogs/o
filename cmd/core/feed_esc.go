// feed_esc.go —— 鲜度死升件环（业主令「探针只上 UI 不落 ops_event=病灶
// 本体——后端对前端零信任」）：ops_face 同源 healthComps() 注册表 60s 拍评
// dead 态——转死=ops_event 升件进唤知环（esc_class=断流/esc_scope=组件名/
// 24h 同窗去重防重启风暴）；复绿=恢复行携升件#id 自闭（处置席核现况
// 零会话开销——唤知环 dedup 视引证行为已处置）。
package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

// feedEscEvery 升件拍节（ops_face dead 阈最小 2min——60s 拍保证死不超拍
// 级误报又不过频；sched-exempt-ok 期界等待非业务轮询）。
const feedEscEvery = 60 * time.Second

// feedEscDedupHours 同窗去重——同键死 24h 至多一报（进程重启后 DB 同窗
// 兜底内存面丢忆；覆盖失明探针同契）。
const feedEscDedupHours = 24

// feedEscLoop 常驻拍（o-core 装配环起——面死/复绿状态机在内存，
// 24h 同窗 dedup 在库）。ctx 终=进程退。
func (c coreService) feedEscLoop(ctx context.Context) {
	t := time.NewTicker(feedEscEvery) // sched-exempt-ok：鲜度判定期界（dead 阈最小 2min,60s 拍非业务轮询）
	defer t.Stop()
	deadSince := map[string]time.Time{} // comp → 本次死起始（恢复行注死窗）
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		for _, b := range healthComps() {
			var ts *time.Time
			if err := c.pool.QueryRow(ctx, b.query).Scan(&ts); err != nil {
				continue // 读件败本拍让位——非死证据不误报
			}
			isDead := false
			if ts == nil {
				isDead = b.zeroNote == "" // 语义空态面（zeroNote 在案）零行非死
			} else {
				isDead = time.Since(*ts) > b.dead
			}
			if isDead {
				if _, ok := deadSince[b.name]; !ok {
					deadSince[b.name] = time.Now()
					c.feedEscalate(ctx, b, ts)
				}
			} else if since, ok := deadSince[b.name]; ok {
				delete(deadSince, b.name)
				c.feedRecover(ctx, b, since)
			}
		}
	}
}

// feedEscalate 转死升件：同窗 24h dedup 后落 ops_event 升件
// （esc_class=断流 esc_scope=组件名——处置席判类册可辨；
// note=面死定性+fix runbook 指针+末鲜 ts 证据）。
func (c coreService) feedEscalate(ctx context.Context, b compBeat, last *time.Time) {
	var dup int
	if err := c.pool.QueryRow(ctx, `SELECT count(*) FROM ops_event
		WHERE event='升件' AND esc_class=$1 AND esc_scope=$2
		AND ts>now()-make_interval(hours=>$3)`,
		"断流", b.name, feedEscDedupHours).Scan(&dup); err != nil || dup > 0 {
		return // 读败静默下拍再试；同窗有报不暴
	}
	lastTxt := "零行"
	if last != nil {
		lastTxt = fmt.Sprintf("末鲜 %s（陈 %s）", last.UTC().Format("01-02 15:04Z"), time.Since(*last).Round(time.Second))
	}
	note := fmt.Sprintf("面断流 %s：%s——超 dead 阈 %s。%s",
		b.name, lastTxt, b.dead, b.fix)
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','',$1,'断流',$2)`,
		note, b.name); err != nil {
		log.Printf("[feedesc] 升件写败 %s: %v", b.name, err)
		return
	}
	log.Printf("[feedesc] 面死升件 %s——%s", b.name, lastTxt)
}

// feedRecover 复绿恢复行：引 升件#<最近同窗未销 id> 自闭（唤知环 dedup
// 视引证=已处置——自愈案例零认知开销；无同窗升件=静默不留废行）。
func (c coreService) feedRecover(ctx context.Context, b compBeat, since time.Time) {
	var id int64
	if err := c.pool.QueryRow(ctx, `SELECT e.id FROM ops_event e
		WHERE e.event='升件' AND e.esc_class='断流' AND e.esc_scope=$1
		AND e.ts>now()-make_interval(hours=>$2)
		AND NOT EXISTS (SELECT 1 FROM ops_event x WHERE x.ts>e.ts
			AND x.note ~ ('升件#'||e.id::text||'([^0-9]|$)'))
		ORDER BY e.id DESC LIMIT 1`,
		b.name, feedEscDedupHours).Scan(&id); err != nil {
		return // 无在案升件（同窗外/已销）=静默
	}
	note := fmt.Sprintf("升件#%d 复绿——%s 恢复新鲜（死窗 %s 内自愈/重启/人工介入；处置席按 repair 日志与 unit 重启痕定性销账）",
		id, b.name, time.Since(since).Round(time.Second))
	if _, err := c.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note)
		VALUES('探测席','恢复','',$1)`, note); err != nil {
		log.Printf("[feedesc] 恢复写败 %s: %v", b.name, err)
		return
	}
	log.Printf("[feedesc] 面复绿 %s（升件#%d 自闭）", b.name, id)
}
