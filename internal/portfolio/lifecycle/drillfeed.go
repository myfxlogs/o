// drillfeed.go —— probe 演习凭据产线（工单#45468：fin_exec_drill 全为
// simdrill:cell:* 占位格凭据、真策略 0 行——exec_drill_missing 死径
// 占 armed_ttl；lifecycle 拍对 state=probe∧Drillable 候选经 DrillFunc
// 缝产凭据，凭据产线并入毕业链——径①票面钦定，缝实现=simdrill.
// FeedDrill 装配件由 cmd/core 注入（portfolio→trading 分层禁直引
// D-514——缝缺席=产线让位不炸拍）。产出=单工人串行（sweepStaleEphemeral
// 全局清扫径不耐并发）；scan 事件拍即踢+慢拍兜底存量候选。
package lifecycle

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/excell"
)

// drillRetryEvery 机械败重试节流（凭据未落的候选下踢仍在列——
// 无节流=scan 事件密拍下同策略连环空跑演习全径）。
const drillRetryEvery = 30 * time.Minute

// DrillFunc 演习缝签名（portfolio 层不引 trading/risk——D-514 分层；
// 实现=simdrill.FeedDrill，cmd/core 装配注入）。verdict=pass|fail
// 凭据已落 fin_exec_drill；机械败走 err（凭据未落——节流重试）。
type DrillFunc func(ctx context.Context, ref int64, inst string) (verdict string, err error)

// AttachDrillFeed 演习缝注入（链式同 AttachUniverse/AttachOpener——
// 缺席=产线让位，scan 拍不产不查）。
func (r *Runner) AttachDrillFeed(fn DrillFunc) *Runner {
	r.drillFn = fn
	return r
}

// drillJob 演习任务（inst 随行——VenueForInst 装配在缝实现侧）。
type drillJob struct {
	ref  int64
	inst string
}

// drillFeedCandidates 缺凭据 probe 候选集（execDrillPass 同窗同真源：
// 窗内无任何 drill 行=drilled=false——fail 凭据在案者不重钻，病灶修复
// 后凭据面自解归闸语义不动）。simdrill% ephemeral 行排除（演习产物
// 非真策略）。Drillable 谓词 Go 侧滤（SQL 不可达 LegShapeOf 单源）。
func drillFeedCandidates(ctx context.Context, pool *pgxpool.Pool) ([]drillJob, error) {
	rows, err := pool.Query(ctx, `SELECT s.id, s.inst_id, s.template_ref
		FROM fin_strategy s
		WHERE s.state='probe'
		  AND s.template_ref NOT LIKE 'simdrill%'
		  AND NOT EXISTS (
			SELECT 1 FROM fin_exec_drill d
			WHERE d.strategy_ref = s.id
			  AND d.ts >= s.entered_at - interval '2 days')
		ORDER BY s.id
		LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []drillJob
	for rows.Next() {
		var j drillJob
		var tmpl string
		if err := rows.Scan(&j.ref, &j.inst, &tmpl); err != nil {
			return nil, err
		}
		if !excell.Drillable(tmpl) {
			continue // exec_drill_unshaped 域——Run 拒载族不烧演习全径
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// drillFeedKick 候选扫描+入队（scan 拍/慢拍共用——非阻塞投递，
// 工人满/节流中静默跳过下踢自补；缝缺席/读败只记日志不拖主拍）。
func (r *Runner) drillFeedKick(ctx context.Context) {
	if r.drillFn == nil {
		return // 缝缺席=产线让位（AttachDrillFeed 未注——o-research 等非 cmd/core 宿主天然关）
	}
	cands, err := drillFeedCandidates(ctx, r.pool)
	if err != nil {
		log.Printf("[lifecycle] 演习凭据候选扫败（产线让位下踢自补）: %v", err)
		return
	}
	now := time.Now()
	queued := 0
	for _, j := range cands {
		r.drillMu.Lock()
		if now.Sub(r.drillAt[j.ref]) < drillRetryEvery {
			r.drillMu.Unlock()
			continue
		}
		r.drillAt[j.ref] = now
		select {
		case r.drillCh <- j:
			queued++
		default:
		}
		r.drillMu.Unlock()
	}
	if queued > 0 {
		log.Printf("[lifecycle] 演习凭据产线入队 %d 候选（probe 缺凭据——毕业链供件）", queued)
	}
}

// drillFeedWorker 单工人串行消费（sweepStaleEphemeral 全局清扫径不耐
// 并发——多候选排队逐跑）。
func (r *Runner) drillFeedWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-r.drillCh:
			if r.drillFn == nil {
				continue
			}
			verdict, err := r.drillFn(ctx, j.ref, j.inst)
			if err != nil {
				log.Printf("[lifecycle] 演习 #%d %s 机械败（凭据未落——%s 后重试）: %v",
					j.ref, j.inst, drillRetryEvery, err)
				continue
			}
			log.Printf("[lifecycle] 演习 #%d %s verdict=%s（凭据已落 fin_exec_drill——毕业闸自解径）",
				j.ref, j.inst, verdict)
		}
	}
}

// drillFeedTicker 慢拍兜底（存量 probe 候选/事件漏踢——scan 事件驱动
// 拍只保新入探；间隔键 lifecycle.drill_feed_min 缺省 10min）。
func (r *Runner) drillFeedTicker(ctx context.Context) {
	every := 10 * time.Minute
	if v, err := r.cfg.GetFloat("lifecycle.drill_feed_min"); err == nil && v > 0 {
		every = time.Duration(v * float64(time.Minute))
	}
	t := time.NewTicker(every) // sched-exempt-ok 慢拍兜底面（存量 probe 候选入仓事件已发不重发——事件径结构性失明段，幂等重扫无害同 paramIntakeLoop 律）
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.drillFeedKick(ctx)
		}
	}
}

// drillFeedStart 产线两协程拉起（Run() 同槽——ctx 界内协程非野径）。
func (r *Runner) drillFeedStart(ctx context.Context) {
	go r.drillFeedWorker(ctx)
	go r.drillFeedTicker(ctx)
}
