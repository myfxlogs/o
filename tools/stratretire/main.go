// stratretire —— 策略退役窄门（业主令 2026-11-10：策略上/退役=系统自裁,
// 业主只管出钱/出凭证/收收益——条件请求退役径废止;sessions 判词直裁）。
// 走 lifecycle.Transition 合法迁（live→retired 闭集内,免委员会开关——
// 开关只闸「进 live」）+fin_strategy_transition 留痕+ops_decision 墓志铭,
// 同 retire.go 自动退役完全同径——人裁/机裁/自动裁留痕面无别。
//
//	DB_DSN=postgres://... go run ./tools/stratretire -id 3544 -reason "判词"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/portfolio/lifecycle"
)

func main() {
	id := flag.Int64("id", 0, "fin_strategy.id")
	reason := flag.String("reason", "", "退役判词（必填——进 transition 留痕+决策记忆墓志铭）")
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
	if err := lifecycle.Transition(ctx, pool, *id, "retired", *reason, false); err != nil {
		fmt.Fprintln(os.Stderr, "retire: ", err)
		os.Exit(1)
	}
	fmt.Printf("策略#%d 已退役（fin_strategy_transition+ops_decision 墓志铭在案）\n", *id)
}
