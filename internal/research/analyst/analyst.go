// Package analyst —— regime 分析席臂（T-0310/D-0043：外部事实摄入→结构化
// 判词；写域收窄仅 md_regime_fact/md_regime_memo INSERT——不触 fin_* 不动
// app_config/提案/仓位；注入双层防：ctx 内容=不可信文本 prompt 隔读+stance
// 枚举域校验）。
package analyst

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
)

// LLM 判词件缝（假想官基建同款注入式——AttachLLM 不接=产出行缺席如实；
// 生产注入口径=装配件挂实现 client）。
type LLM func(ctx context.Context, sys, user string) (string, error)

// Fetcher HTTP 取件缝（公开零凭据端点——凭据面禁线同 scout）。
type Fetcher func(ctx context.Context, url string) ([]byte, error)

// Runner 分析席臂（6h 拍+TickOnce 人工触发）。
type Runner struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	al    *notify.Alerter
	llm   LLM
	fetch Fetcher
	now   func() time.Time
}

func NewRunner(pool *pgxpool.Pool, cfg *config.Client) *Runner {
	return &Runner{pool: pool, cfg: cfg,
		rej: rejects.New("analyst", "fetch_err", "write_err", "parse_err",
			"llm_absent", "stance_off_vocab"),
		fetch: httpGet, now: time.Now}
}

func (r *Runner) AttachAlerter(a *notify.Alerter) *Runner { r.al = a; return r }

// AttachLLM 挂判词件（nil=守位——采集照常、产出行缺席如实）。
func (r *Runner) AttachLLM(fn LLM) *Runner { r.llm = fn; return r }

// Run 6h 节拍（每结算事件密度低——日四拍足）。
func (r *Runner) Run(ctx context.Context, errc chan<- error) {
	for {
		if err := r.TickOnce(ctx); err != nil {
			log.Printf("[analyst] 拍败（下拍续试）: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(6 * time.Hour): // sched-exempt-ok 6h 钦定节拍
		}
	}
}

// TickOnce 单拍壳（人工触发+测试——采集+分析两阶段各 fail-soft）。
func (r *Runner) TickOnce(ctx context.Context) error {
	if err := r.collect(ctx); err != nil {
		return err
	}
	return r.analyze(ctx)
}
