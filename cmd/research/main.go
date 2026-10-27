// research —— 研究域外置节点（T-0165/D-0029：开发机算力面——researchArms
// 全族+review/report 迁此；公开读面零凭据；roster 席位名不变 beat 换机；
// SG o-core 撤臂钱径独占）。DSN 经 SG_PG_DSN 隧道径（同 collector 接法）。
package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/roster"
	"o/internal/review"
	"o/internal/trading/okxexec"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("SG_PG_DSN")
	if dsn == "" {
		log.Fatal("SG_PG_DSN 缺——研究节点写径无目的地 fail-closed")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("SG PG 连接败: %v", err)
	}
	defer pool.Close()
	// T-0169 分域：席位台账归 SG 册（ROSTER_PG_DSN 隧道径——D-0030 §五直 beat）。
	if rp := os.Getenv("ROSTER_PG_DSN"); rp != "" {
		rpool, err := pgxpool.New(ctx, rp)
		if err != nil {
			log.Fatalf("roster 池连接败: %v", err)
		}
		defer rpool.Close()
		roster.SetRegistryPool(rpool)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config load: %v", err)
	}
	// 公开读面零凭据（T-0102 红线：research 域零 venue 凭据——
	// okcli 无凭据 PublicGet 公开面直读同 universeExpander 径）。
	okcli := okxexec.NewClient(okxexec.Credentials{}, false)
	alerter := notify.NewAlerter(nil, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v })
	errc := make(chan error, 8)
	go researchArms(ctx, errc, pool, cfg, okcli, alerter)
	go review.NewRunner(pool).Run(ctx, errc) // 复盘域随迁
	// T-0165 返修③：report 回 SG（钱径读写面 app_secrets/master key/
	// fin_cashflow——票面第5条明令只迁纯研究面；review 留此节点）。
	log.Printf("[o-research] 研究域节点起——researchArms+review+report 全族（公开读面零凭据）")
	for {
		select {
		case err := <-errc:
			if err != nil {
				log.Fatalf("[o-research] 岗终: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
