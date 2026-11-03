// Package scout —— venue scout 骨架（T-0300/D-0043）：公开面零凭据采集全球
// venue funding 事实行（md_venue_funding）→ 三族净 APR 度量（T-0297 同源
// 口径）→ 达阈自动向业主发「请注册」条件请求。度量域只读不写钱径——
// fin_opp_template/fin_proposal 零触。
package scout

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
)

// Runner scout 臂（采集+度量+推荐——日拍）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	al    *notify.Alerter
	fetch Fetcher // HTTP 取件（测试可桩——生产=httpGet）
	now   func() time.Time
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("scout", "missing_param", "fetch_err", "write_err",
			"parse_err", "venue_down", "earn_rate_degraded"),
		fetch: httpGet, now: time.Now}
}

// AttachAlerter 挂业主通知径（条件请求出口——现有 Alerter 复用）。
func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.al = a; return r }

// Run 日拍节拍（measure_interval_h 热启，缺省 24h——采集每结算一拍足够）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	for {
		if err := r.tick(ctx); err != nil {
			log.Printf("[scout] 拍败（下拍续试）: %v", err)
		}
		iv := 24 * time.Hour
		if v, err := r.cfg.GetFloat("scout.measure_interval_h"); err == nil && v > 0 {
			iv = time.Duration(v * float64(time.Hour))
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(iv): // sched-exempt-ok 日拍度量节拍（公开面轮询钦定节拍非事件域）
		}
	}
}

// TickOnce 单拍壳（测试/实证件消费——tick 本体私有语义不变）。
func (r *Runner) TickOnce(ctx context.Context) error { return r.tick(ctx) }

func (r *Runner) tick(ctx context.Context) error {
	if err := r.collectAll(ctx); err != nil {
		return err
	}
	return r.measure(ctx)
}
