// mtonboard —— T-0379 broker 接入拍干跑件：单 venue 截拍推进（锚②——
// EXR 3001157「首个 broker」干跑①②探测+草案实证协议面，不激活）。
// 用法：MD_PG_DSN/FIN_PG_DSN 双池+O_MASTER_KEY_FILE 环境，-venue/-beats 参。
// 双池同库可（dev 隧道 5544→SG PG-fin 实落 app_secrets 读+candidate 镜像写
// 留意：真落账以部署后 o-research 臂为准——干跑写面=ops_event+evidence 锚）。
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/venue/onboard"
)

func main() {
	venue := flag.String("venue", "mt5", "venue（<broker>.mt5.* 凭据族前缀；mt5=单例 EXR 首案）")
	beats := flag.Int("beats", 2, "截拍数（0=全链；2=探测+草案不激活——EXR 干跑钦定）")
	flag.Parse()
	ctx := context.Background()
	dsn := func(k string) string {
		v := os.Getenv(k)
		if v == "" {
			log.Fatalf("%s 缺 fail-closed", k)
		}
		return v
	}
	md, err := pgxpool.New(ctx, dsn("MD_PG_DSN"))
	if err != nil {
		log.Fatalf("md 池: %v", err)
	}
	defer md.Close()
	fin, err := pgxpool.New(ctx, dsn("FIN_PG_DSN"))
	if err != nil {
		log.Fatalf("fin 池: %v", err)
	}
	defer fin.Close()
	cfg := config.New(md)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("cfg load: %v", err)
	}
	if err := onboard.New(md, fin, cfg).Advance(ctx, *venue, *beats); err != nil {
		log.Fatalf("advance: %v", err)
	}
	var st string
	if err := md.QueryRow(ctx,
		`SELECT state FROM fin_venue_candidate WHERE venue=$1`, *venue).Scan(&st); err != nil {
		log.Printf("venue=%s 台账行缺（凭据族不在册？）: %v", *venue, err)
		return
	}
	log.Printf("[mtonboard] venue=%s beats=%d 终态=%s——ops_event「venue接入」行+evidence 锚自查", *venue, *beats, st)
}
