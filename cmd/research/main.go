// research —— 研究域外置节点（T-0165/D-0029：开发机算力面——researchArms
// 全族+review/report 迁此；roster 席位名不变 beat 换机；
// SG o-core 撤臂钱径独占）。DSN 经 MD_PG_DSN 本机 PG-md（同 collector 接法——
// T-0232 正名：SG_PG_DSN 名残指 md 归位）。
// T-0170 认证读面：READCRED_URL 设=只读钥 boot 内存取+401 自重取（径 A）。
package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
	"o/internal/review"
	"o/internal/store/pgstore"
	"o/internal/trading/okxexec"
)

func main() {
	ctx := context.Background()
	dsn := os.Getenv("MD_PG_DSN")
	if dsn == "" {
		log.Fatal("MD_PG_DSN 缺——研究节点写径无目的地 fail-closed")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("PG-md 连接败: %v", err)
	}
	defer pool.Close()
	// T-0181：boot 背 md 迁移闸——md 链执行者缺位根治（与 SG o-core
	// Migrate(fin) 对称；dev 部署必重启 research=md schema 自追，
	// 「已合并≠已执行」灰色态物理消失）。
	if err := pgstore.Migrate(ctx, pool, "md"); err != nil {
		log.Fatalf("md migrate 败: %v", err)
	}
	// T-0169 分域：席位台账归 SG 册（ROSTER_PG_DSN 隧道径——D-0030 §五直 beat）。
	if rp := os.Getenv("ROSTER_PG_DSN"); rp != "" {
		rpool, err := pgxpool.New(ctx, rp)
		if err != nil {
			log.Fatalf("roster 池连接败: %v", err)
		}
		defer rpool.Close()
		roster.SetRegistryPool(rpool)
	}
	// T-0172 分域：review 钱径读写面归 SG 小池（FIN_PG_DSN 隧道径——fin 域表
	// 正本在 SG，md 池无 fin 读面=评史官全盲；与 ROSTER_PG_DSN 同形）。
	finDSN := os.Getenv("FIN_PG_DSN")
	if finDSN == "" {
		log.Fatal("FIN_PG_DSN 缺——review 钱径读写面无目的地 fail-closed")
	}
	finPool, err := pgxpool.New(ctx, finDSN)
	if err != nil {
		log.Fatalf("fin 池连接败: %v", err)
	}
	defer finPool.Close()
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config load: %v", err)
	}
	// T-0170 认证读面（径 A 终裁）：READCRED_URL 设→boot 经隧道（5545）取
	// venue-map 仅进内存+401 自重取轮换自愈；缺→无凭据读面（dcd/gapscan 私有
	// 读面 401 fail-closed 让位——桩锚不回归）。T-0174：okx 臂按 venue 提件。
	var okcli authGetter = okxexec.NewClient(okxexec.Credentials{}, false)
	rcMode := "无凭据公开读"
	if url := os.Getenv("READCRED_URL"); url != "" {
		okcli = newAuthClient(url, "okx")
		rcMode = "readcred 认证读（只读钥内存取）"
	}
	alerter := notify.NewAlerter(nil, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v })
	errc := make(chan error, 8)
	rej := rejects.New("research", "dcd_pair_429") // T-0180 对级容错拒因键
	go researchArms(ctx, errc, pool, cfg, okcli, alerter, rej)
	go review.NewRunner(pool, finPool).Run(ctx, errc) // 复盘域随迁（钱径臂 SG 小池——T-0172）
	// T-0165 返修③：report 回 SG（钱径读写面 app_secrets/master key/
	// fin_cashflow——票面第5条明令只迁纯研究面；review 留此节点）。
	log.Printf("[o-research] 研究域节点起——researchArms+review+report 全族（读面=%s）", rcMode)
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
