// Package review —— 复盘判决书+远期验证（G10 闭环收口）。
// 铁律：一仓一判决书（settled→自动复盘）；verdict 四值闭集分级不二元；
// 归因初判=机读规则禁 LLM 判钱；否决提案同排远期复测（DSHQ 五池模式——
// 拒对/拒错=提案闸质量考据）；复盘写回=提案态行不自动改判据；
// falsified→经济经验库待归纳行。
package review

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/events"
	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
)

var stagingFile = "docs/经济经验库/待归纳.md"

// Runner 评史官岗。
type Runner struct {
	pool          *pgxpool.Pool
	rej           *rejects.Door
	stagingLogged bool // 待归纳镜像写败一次性标记（容器 FS 易失——best-effort 不刷屏）
}

func NewRunner(pool *pgxpool.Pool) *Runner {
	return &Runner{pool: pool, rej: rejects.New("review", "write_err")}
}

func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, r.pool, "评史官", "review", "复盘判决书+远期验证官", nil); err != nil {
		errc <- fmt.Errorf("评史官 register: %w", err)
		return
	}
	go roster.BeatLoop(ctx, r.pool, "评史官", 30*time.Second, errc)
	// 事件化（T-0048：周期环拆——仓行变 ev_position+判决到期期界等待）。
	events.ListenOrDeadline(ctx, r.pool, "review",
		[]string{"ev_position", "ev_fwdchk"}, r.nextDue, r.scan)
}

// nextDue 下一判决到期距（fin_forward_check 最早未判 target_ts——期界等待语义）。
func (r *Runner) nextDue(ctx context.Context) time.Duration {
	var ts time.Time
	if err := r.pool.QueryRow(ctx,
		"SELECT min(target_ts) FROM fin_forward_check WHERE judged_ok IS NULL").Scan(&ts); err != nil {
		return 0
	}
	d := time.Until(ts)
	if d < 0 {
		return 0 // 已到期=立即一拍
	}
	return d
}

func (r *Runner) scan(ctx context.Context) error {
	if err := r.reviewSettled(ctx); err != nil {
		return err
	}
	if err := r.scheduleRejected(ctx); err != nil {
		return err
	}
	return r.judgeDue(ctx)
}

// Verdict 定级（机读规则）：pred>0 下 realized<0 或偏差>60%=falsified；
// ≤10% strong、≤30% mild、≤60% marginal（pred≈0 时按 |actual| 判）。
func Verdict(pred, actual float64) string {
	if pred <= 0 {
		if actual < 0 {
			return "falsified"
		}
		return "marginal"
	}
	dev := (pred - actual) / pred
	switch {
	case actual < 0 || dev > 0.6:
		return "falsified"
	case dev <= 0.1:
		return "strong"
	case dev <= 0.3:
		return "mild"
	default:
		return "marginal"
	}
}

// Attribut 归因初判（机读规则——fee 占比主导=fee_drift；funding 实现<卡值=funding_flip；
// 偏差正侧=luck；其余 model_err）。
func Attribut(pred, actual, feeShare float64) string {
	if actual >= pred*0.9 {
		return "luck"
	}
	if feeShare > 0.5 && actual < pred {
		return "fee_drift"
	}
	if actual < 0 {
		return "funding_flip"
	}
	return "model_err"
}
