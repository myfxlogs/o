// migrate —— 分域迁移一次性执行器（T-0167 票面④的开发机执行载体）：
// PG-md 建库/补链显式径。SG 侧 fin 链由 o-core boot 自带（cmd/core）；开发机
// md 链在 collector/research 的 DSN 切换落定（T-0169 boot 迁移接管）前由本件
// 建/补链。fail-closed：域必选、PG_DSN 必设、非法域拒。
//
// 用法：PG_DSN=postgres://o:o@127.0.0.1:5435/o?sslmode=disable migrate -domain=md
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/pgguard"
	"o/internal/store/pgstore"
)

func main() {
	domain := flag.String("domain", "", "选链域 md|fin（必选——分域选链 fail-closed）")
	flag.Parse()
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		log.Fatal("PG_DSN 缺——迁移无目的地 fail-closed")
	}
	if *domain == "" {
		log.Fatal("-domain 缺（md|fin）——分域选链 fail-closed")
	}
	ctx := context.Background()
	pcfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("PG_DSN 解析败: %v", err)
	}
	pgguard.GuardPool(pcfg) // T-0154 事务残留守卫（同源普查——自建池同守）
	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		log.Fatalf("PG 连接败: %v", err)
	}
	defer pool.Close()
	if err := pgstore.Migrate(ctx, pool, *domain); err != nil {
		log.Fatalf("migrate %s: %v", *domain, err)
	}
	log.Printf("[migrate] %s 链应用毕（schema_migrations 基线齐）", *domain)
}
