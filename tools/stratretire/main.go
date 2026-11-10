// stratretire —— 策略退役窄门（业主令 2026-11-10：策略上/退役=系统自裁,
// 业主只管出钱/出凭证/收收益——条件请求退役径废止;sessions 判词直裁）。
// 调 fin_retire(id,reason) SECURITY DEFINER 函数（fin_0161——roster_rw 持
// EXECUTE 即最窄写面;函数体内合法迁+state 翻转+transition 留痕同事务,
// 与 lifecycle.Transition 退役径同构）。退役对全态合法=闭集无漂移面。
//
//	DB_DSN=postgres://... go run ./tools/stratretire -id 3544 -reason "判词"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	id := flag.Int64("id", 0, "fin_strategy.id")
	reason := flag.String("reason", "", "退役判词（必填——进 transition 留痕）")
	flag.Parse()
	if *id <= 0 || *reason == "" {
		fmt.Fprintln(os.Stderr, "用法: DB_DSN=... stratretire -id <策略id> -reason <判词>")
		os.Exit(1)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "db: ", err)
		os.Exit(1)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, "SELECT fin_retire($1,$2)", *id, *reason); err != nil {
		fmt.Fprintln(os.Stderr, "retire: ", err)
		os.Exit(1)
	}
	fmt.Printf("策略#%d 已退役（fin_strategy_transition 留痕在案）\n", *id)
}
