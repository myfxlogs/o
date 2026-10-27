// projector —— 投影臂入口（T-0168/D-0030 §四）：PG-md→SG 八桥镜像搬运+
// fin_proposal 双向回写。双 DSN env 引导（MD_PG_DSN=本机 PG-md；SG_PG_DSN=
// 隧道径同 collector 接法）；md 链随 boot 幂等迁移（帧源触发器自保障）；
// 败=岗终 fail-closed（systemd 重启→reconciler 自愈）。
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"o/internal/platform/config"
	"o/internal/platform/notify"
	"o/internal/projector"
	"o/internal/store/pgstore"
)

func main() {
	ctx := context.Background()
	mode := flag.String("mode", "serve", "serve=正形投影（活跃集谓词）|dualwrite=对拍旁写泵（全量搬运）")
	flag.Parse()
	mdDSN := os.Getenv("MD_PG_DSN")
	if mdDSN == "" {
		log.Fatal("MD_PG_DSN 缺——投影无正本源 fail-closed")
	}
	sgDSN := os.Getenv("SG_PG_DSN")
	if sgDSN == "" {
		log.Fatal("SG_PG_DSN 缺——投影无镜像地 fail-closed")
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
	// md 链幂等迁移（md_1067 投影帧触发器随链在位——boot 自保障，不赖外部手排）。
	if err := pgstore.Migrate(ctx, mdPool, "md"); err != nil {
		log.Fatalf("md 链迁移: %v", err)
	}
	// 告警开关面=app_config（SG 本地读——config_changed 热载由 config 包承担）。
	cfg := config.New(sgPool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config load: %v", err)
	}
	alerter := notify.NewAlerter(nil, func() string {
		v, _ := cfg.GetString("feishu.chat_id")
		return v
	}, func() bool { v, _ := cfg.GetBool("notify.feishu.enabled"); return v })
	r, err := projector.New(mdPool, sgPool, alerter, *mode)
	if err != nil {
		log.Fatalf("装配: %v", err)
	}
	errc := make(chan error, 8)
	go r.Run(ctx, errc)
	log.Printf("[projector] 投影臂起（mode=%s）——八桥镜像+proposal 双向+config 第九桥副本", *mode)
	for err := range errc {
		log.Fatalf("[projector] 岗终: %v", err) // 单败即岗终——restart+对账自愈径
	}
}
