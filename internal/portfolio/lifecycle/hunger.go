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

	"o/internal/platform/opschain"
	"o/internal/portfolio/hunger"
)

// hungerSensor 饥饿感知缝（T-0378——签名对形 hunger.Sensor；桩测覆写锚）。
// T-0384：ProbeFamilies 族级沉默探针同缝入拍（Detect 同槽省协程）。
// T-0385：MineLessons 教训矿工同缝——独立 minerLoop 12h 慢拍槽。
// T-0392：ProbeLegCoverage 腿-行情覆盖探针同缝——探针族同槽入拍。
type hungerSensor interface {
	Detect(ctx context.Context) (*hunger.Report, error)
	Respond(ctx context.Context, rep *hunger.Report) ([]hunger.StepResult, error)
	ProbeFamilies(ctx context.Context) error
	ProbeLegCoverage(ctx context.Context) error
	MineLessons(ctx context.Context) (int, error)
	DraftTickets(ctx context.Context) (int, error)
}

// opsChainVerifier 验链缝（T-0386——ops_event 防篡改账验链拍挂饥饿同槽；
// 桩测覆写锚，置零=跳验链拍——桩测无池 Runner 合法同例）。
type opsChainVerifier interface {
	Verify(ctx context.Context) (*opschain.Report, error)
	ReportBreak(ctx context.Context, rep *opschain.Report) error
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
// 进 ticker 前先跑一次 hungerTick——部署即首侦（T-0381：生产已饿 ~8 天，
// 首拍再等 +6h 是起表副作用非设计意图）；之后照旧 hungerBeatEvery 复拍。
func (r *Runner) hungerLoop(ctx context.Context) {
	t := time.NewTicker(hungerBeatEvery) // sched-exempt-ok 看门判定节拍（饥饿态即事件静默态——事件径结构性失明，检测器兜底非轮询拉取）
	defer t.Stop()
	r.hungerTick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.hungerTick(ctx)
		}
	}
}

// hungerTick 单拍：验链拍（T-0386——先于一切每拍全链重算，断链/伪戳
// →账链断+升件自报）→族沉默探针（T-0384——先于 Detect 每拍皆测，饥饿与否
// 无关；败情自报不拖主拍）→腿覆盖探针（T-0392——同款隔离位）→Detect→
// 触发即落 `饥饿` 行（拍壳同构——触发形+证据落账）→Respond 一步响应。
// 败情不上抛——本拍自身终结。
func (r *Runner) hungerTick(ctx context.Context) {
	// T-0386 验链拍同槽：全链重算纯读，断链/伪戳→账链断+升件自报；
	// 败情让位不阻主拍（ProbeFamilies 同款隔离位）。
	if r.chainV != nil {
		if rep, err := r.chainV.Verify(ctx); err != nil {
			log.Printf("[lifecycle] hungertick 验链拍败: %v", err)
			r.hungerEscalate(ctx, "opschain", err)
		} else if !rep.OK() {
			log.Printf("[lifecycle] hungertick 账链断 检出=%d 首处=%s@#%d",
				len(rep.Findings), rep.Findings[0].Kind, rep.Findings[0].ID)
			if err := r.chainV.ReportBreak(ctx, rep); err != nil {
				log.Printf("[lifecycle] hungertick 账链断自报落败（结构性限度静默退）: %v", err)
			}
		}
	}
	if err := r.sensor.ProbeFamilies(ctx); err != nil {
		log.Printf("[lifecycle] hungertick 族沉默探针败: %v", err)
		r.hungerEscalate(ctx, "silence_probe", err)
	}
	// T-0392 腿覆盖探针同槽（ProbeFamilies 同款败情隔离位——败情让位
	// 不阻主拍；探针自身败=升件自报非静默）。
	if err := r.sensor.ProbeLegCoverage(ctx); err != nil {
		log.Printf("[lifecycle] hungertick 腿覆盖探针败: %v", err)
		r.hungerEscalate(ctx, "legcover_probe", err)
	}
	// T-0393 起草拍同槽：探针升件行→候选草案 pending（败情让位同款
	// 隔离位——起草拍自身终结不拖主拍）。
	if n, err := r.sensor.DraftTickets(ctx); err != nil {
		log.Printf("[lifecycle] hungertick 起草拍败: %v", err)
		r.hungerEscalate(ctx, "drafter", err)
	} else if n > 0 {
		log.Printf("[lifecycle] hungertick 起草拍候选草案 ×%d", n)
	}
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
