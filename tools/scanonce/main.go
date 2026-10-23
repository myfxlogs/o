// scanonce —— T-0018 dev 实证件：直连 dev 库跑一拍 research scan，
// 证「数字种子+类型化读器」下零 unmarshal 错+fin_opp_eval 产出 verdict 行。
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/research/funding"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := funding.NewRunner(pool, cfg).ScanOnce(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "scan FAIL:", err)
		os.Exit(1)
	}
	var n int
	if err := pool.QueryRow(ctx,
		"SELECT count(*) FROM fin_opp_eval WHERE ts > now()-interval '2 minutes'").Scan(&n); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("SCAN-OK eval_rows_last_2min=%d\n", n)
}
