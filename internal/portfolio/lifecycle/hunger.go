// hunger.go —— 饥饿感知拍接线（T-0378——T-0376 自治环末件入岗）：
// internal/portfolio/hunger 死件激活径。独立 ticker 节拍非挂 scan——
// 饥饿态=零提案零策略变=ev_strategy/ev_eval 恰好静默，事件拍在最需
// 检测时结构性失明（SilentWatch 看门拍同构先例）。Detect 先于 Respond
// 每拍至多一响（窗口判据 Detect 内部已带）；败=log+升件自报行，不拖
// 事件主拍（T-0369 容错必报——非仅 stderr）。拍壳语义对形 tools/hungertick。
package lifecycle

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/portfolio/hunger"
)

// hungerSensor 饥饿感知缝（T-0378——签名对形 hunger.Sensor；桩测覆写锚）。
type hungerSensor interface {
	Detect(ctx context.Context) (*hunger.Report, error)
	Respond(ctx context.Context, rep *hunger.Report) ([]hunger.StepResult, error)
}

// hunger 拍行席别/环票号（internal/portfolio/hunger 同款口径——DM 执行环=
// 决策席，例行行携常设环票号 T-0360 同例）。
const (
	hungerSeat   = "决策席"
	hungerTicket = "T-0376"
)

// hungerBeatEvery 饥饿拍周期（var=测试面覆写锚——生产值钦定不动）。
var hungerBeatEvery = 6 * time.Hour

// hungerLoop 饥饿感知拍：BeatLoop 同款 ctx 界内协程（Run 拉起——非野径）。
func (r *Runner) hungerLoop(ctx context.Context) {
	t := time.NewTicker(hungerBeatEvery) // sched-exempt-ok 看门判定节拍（饥饿态即事件静默态——事件径结构性失明，检测器兜底非轮询拉取）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.hungerTick(ctx)
		}
	}
}

// hungerTick 单拍：Detect→触发即落 `饥饿` 行（拍壳同构——触发形+证据
// 落账）→Respond 一步响应。败情不上抛——本拍自身终结。
func (r *Runner) hungerTick(ctx context.Context) {
	rep, err := r.sensor.Detect(ctx)
	if err != nil {
		log.Printf("[lifecycle] hungertick detect 败: %v", err)
		r.hungerEscalate(ctx, "detect", err)
		return
	}
	if !rep.Triggered {
		log.Printf("[lifecycle] hungertick 饱食（窗=%d天 零触发）", rep.WindowDays)
		return
	}
	if _, err := r.pool.Exec(ctx,
		`INSERT INTO ops_event(seat,event,ticket,note) VALUES($1,'饥饿',$2,$3)`,
		hungerSeat, hungerTicket, strings.Join(rep.Reasons, " | ")); err != nil {
		log.Printf("[lifecycle] hungertick 饥饿行落败（响应照跑——步行自落账）: %v", err)
	}
	if _, err := r.sensor.Respond(ctx, rep); err != nil {
		// 步级败情已随 hunger.Respond 逐步落 `饥饿响应` 行（logStep 在包内）；
		// 此处环级自报补 logStep 自身败情与可见性（同窗去重不暴）。
		log.Printf("[lifecycle] hungertick respond 败: %v", err)
		r.hungerEscalate(ctx, "respond", err)
	}
}

// hungerEscalate 败情自报升件行（T-0369 容错必报）：写败/LIFECYCLE-HUNGER
// 判例键，note 前缀同窗 24h 至多一行（ops_esc.sh 同构去重——同根因不暴）；
// 自报行自身写败=静默退（升件不能再升——T-0369 结构性限度在案）。
func (r *Runner) hungerEscalate(ctx context.Context, phase string, cause error) {
	prefix := "hungertick " + phase + " 败"
	if _, err := r.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		SELECT $1,'升件',$2,$3,'写败','LIFECYCLE-HUNGER'
		WHERE NOT EXISTS (SELECT 1 FROM ops_event
			WHERE event='升件' AND note LIKE $4 AND ts>now()-interval '24 hours')`,
		hungerSeat, hungerTicket, fmt.Sprintf("%s: %v", prefix, cause), prefix+"%"); err != nil {
		log.Printf("[lifecycle] hungertick 自报行落败（结构性限度静默退）: %v（原败: %v）", err, cause)
	}
}
