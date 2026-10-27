// sweeper —— PG-md 侧清道夫（T-0169 承接段：清道夫双实例配装——SG 钱径账
// core 自带；PG-md 账=md_1069 在册，本件日拍照账）。席位 beat 走 SG registry
// （fin_employee_registry 归 fin——PG-md 不另立册）。
// 用法：MD_PG_DSN=... SG_PG_DSN=... sweeper
package main

import (
	"context"
	"log"
	"os"
	"time"

	"o/internal/ledger/retention"
	"o/internal/platform/roster"
	"o/internal/store/pgstore"
)

const seatName = "清道夫-md"

func main() {
	ctx := context.Background()
	mdDSN := os.Getenv("MD_PG_DSN")
	if mdDSN == "" {
		log.Fatal("MD_PG_DSN 缺——清道夫无账面 fail-closed")
	}
	sgDSN := os.Getenv("SG_PG_DSN")
	if sgDSN == "" {
		log.Fatal("SG_PG_DSN 缺——席位 beat 无册 fail-closed")
	}
	mdPool, err := pgstore.Connect(ctx, mdDSN)
	if err != nil {
		log.Fatalf("PG-md 连接败: %v", err)
	}
	defer mdPool.Close()
	sgPool, err := pgstore.Connect(ctx, sgDSN)
	if err != nil {
		log.Fatalf("SG 连接败: %v", err)
	}
	defer sgPool.Close()
	// md 链幂等迁移（保留账 md_1069 随链在位——boot 自保障）。
	if err := pgstore.Migrate(ctx, mdPool, "md"); err != nil {
		log.Fatalf("md 链迁移: %v", err)
	}
	if err := roster.Register(ctx, sgPool, seatName, "ledger", "PG-md 数据保留清道（分库账 md_1069）", nil); err != nil {
		log.Fatalf("roster register: %v", err)
	}
	sw := retention.New(mdPool, func(ctx context.Context, msg string) {
		log.Printf("ALERT [retention-md] %s", msg) // dev 侧 log 面告警（feishu 凭据归 SG）
	})
	tick := time.NewTicker(24 * time.Hour) // sched-exempt-ok：日拍清道（保留账巡面非业务轮询——同 cmd/core/retention.go 豁免形）
	defer tick.Stop()
	errc := make(chan error, 4)
	go roster.BeatLoop(ctx, sgPool, seatName, 30*time.Second, errc)
	log.Printf("[sweeper] PG-md 清道夫起——分库账日拍（首拍即时）")
	if err := sw.Sweep(ctx); err != nil {
		log.Fatalf("[sweeper] 首拍: %v", err)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case err := <-errc:
			log.Fatalf("[sweeper] beat: %v", err) // 单败岗终——restart 径
		case <-tick.C:
			if err := sw.Sweep(ctx); err != nil {
				log.Fatalf("[sweeper] 岗终: %v", err)
			}
		}
	}
}
