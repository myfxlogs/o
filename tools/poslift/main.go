// poslift —— 平仓死信复位 CLI（工单#22109）：fin_guard_close.attempts≥
// guard.retry_max_attempts=设计死信位（scanOnce 批 attempts<maxAtt 谓词
// 永排——VVV仓#86 实证解锁只能手 SQL 绕径）。复位唯一径=裁定席判词或
// 业主令（-by 闭集机械守；直写 SQL 绕径=违例）。
//
// 语义：attempts→0 重获重试预算——state/last_try_at/last_error/loop_*
// 不动（事实账不抹；下拍 scanOnce 重捡后按节拍正常走 tend→markClosing
// attempts+1 续账）。复位+ops_event「平仓解锁」行同 tx——复位非静默。
//
// 用法：
//
//	FIN_PG_DSN=… poslift -position 86 -by 裁定席 \
//	  -evidence "裁词锚:guard_close_loop 案 verdict=误判平仓环" -ticket T-xxxx
//
// 注：FIN_PG_DSN 须载 fin_guard_close/fin_position 写权（roster_rw 隧道
// 径对钱径表拒读拒写——fail-closed；运维径=特权 DSN 同 guardlift 惯例）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	position := flag.Int64("position", 0, "复位仓 id（fin_guard_close.position_id=fin_position.id）")
	by := flag.String("by", "", "复位主体（闭集：裁定席|业主——票面钦定唯一径）")
	evidence := flag.String("evidence", "", "判词证据锚（禁空——无判词不复位）")
	ticket := flag.String("ticket", "", "归属票号（四参强制——无归属不复位）")
	flag.Parse()
	if *position <= 0 || *by == "" || *evidence == "" || *ticket == "" {
		fmt.Fprintln(os.Stderr, "用法: poslift -position N -by 裁定席|业主 -evidence E -ticket T（四参必携）")
		os.Exit(2)
	}
	if *by != "裁定席" && *by != "业主" {
		fmt.Fprintf(os.Stderr, "复位主体 %q 越权——唯一径=裁定席判词/业主令\n", *by)
		os.Exit(2)
	}
	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "FIN_PG_DSN 缺——钱径池不可达 fail-closed")
		os.Exit(2)
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "poslift: 池建败: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := lift(context.Background(), pool, *position, *by, *evidence, *ticket); err != nil {
		fmt.Fprintf(os.Stderr, "poslift: %v\n", err)
		os.Exit(1)
	}
}

// lift 复位一仓：前置核（守卫行在+仓未平）→attempts=0→ops_event 裁定行
// 同 tx 原子落。state='closed' 行复位无意义（scanOnce 按 state<>'closed'
// 另滤——attempts 归零也不重捡）显式拒，防假复位假账。
func lift(ctx context.Context, pool *pgxpool.Pool, posID int64, by, evidence, ticket string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var state string
	var attempts int64
	var closed bool
	err = tx.QueryRow(ctx, `
		SELECT g.state, g.attempts, (p.state='closed')
		FROM fin_guard_close g JOIN fin_position p ON p.id = g.position_id
		WHERE g.position_id = $1`, posID).Scan(&state, &attempts, &closed)
	if err != nil {
		return fmt.Errorf("仓 #%d 守卫行读败（无行/连接败）: %w", posID, err)
	}
	if closed {
		return fmt.Errorf("仓 #%d 已平仓（closed_at 落戳）——死信复位无意义", posID)
	}
	if state == "closed" {
		return fmt.Errorf("仓 #%d 守卫行 state=closed 而仓未平——异态非死信位（attempts 复位不重捡，须另案）", posID)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE fin_guard_close SET attempts=0, updated_at=now() WHERE position_id=$1`, posID); err != nil {
		return fmt.Errorf("仓 #%d attempts 复位写败: %w", posID, err)
	}
	note := fmt.Sprintf("仓#%d 死信复位：attempts %d→0（state=%s 保留）——%s %s", posID, attempts, state, ticket, evidence)
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES($1,'平仓解锁',$2)`, by, note); err != nil {
		return fmt.Errorf("仓 #%d 裁定行写败（复位回滚不留无账态）: %w", posID, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Printf("仓 %d attempts %d→0（by=%s state=%s——ops_event 平仓解锁行已落，下拍 scanOnce 重捡）\n",
		posID, attempts, by, state)
	return nil
}
