// guardlift —— 守卫熔断开仓权解冻 CLI（T-0423）：fin_guard_suspend 在挂行
// lifted_* 三列落位+ops_event「开仓解冻」行——解冻唯一径=裁定席判词或
// 业主令（-by 闭集机械守；直写 SQL 绕径=违例）。
//
// 用法：
//
//	FIN_PG_DSN=… guardlift -strategy 42 -by 裁定席 \
//	  -evidence "裁词锚:guard_close_loop 案 verdict=误判平仓环" -ticket T-0423
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/guard"
)

func main() {
	strategy := flag.Int64("strategy", 0, "解冻策略 id（fin_guard_suspend.strategy_ref）")
	by := flag.String("by", "", "解冻主体（闭集：裁定席|业主——票面钦定唯一径）")
	evidence := flag.String("evidence", "", "判词证据锚（禁空——无判词不解冻）")
	ticket := flag.String("ticket", "", "归属票号（审计行出处必携，无归属传 ''）")
	flag.Parse()
	if *strategy <= 0 || *by == "" || *evidence == "" {
		fmt.Fprintln(os.Stderr, "用法: guardlift -strategy N -by 裁定席|业主 -evidence E [-ticket T]（前三参必携）")
		os.Exit(2)
	}
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺——钱径池不可达 fail-closed")
		os.Exit(2)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "guardlift: 池建败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	reason := *evidence
	if *ticket != "" {
		reason = *ticket + " " + *evidence
	}
	if err := guard.LiftSuspend(context.Background(), pool, *strategy, *by, reason); err != nil {
		fmt.Fprintf(os.Stderr, "guardlift: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("策略 %d 开仓权解冻（by=%s——ops_event 开仓解冻行已落）\n", *strategy, *by)
}
