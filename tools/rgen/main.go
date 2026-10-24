// rgen —— T-0026 dev 实证件：对 dev 库跑一拍三粒度报表+打印行。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/ledger/report"
)

func main() {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	now := time.Now()
	for _, p := range []string{"daily", "weekly", "monthly"} {
		b := report.BoundsOf(p, now)
		if err := report.Gen(ctx, pool, p, b); err != nil {
			fmt.Fprintln(os.Stderr, p, err)
			os.Exit(1)
		}
	}
}
