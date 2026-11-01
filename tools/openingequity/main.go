// openingequity —— T-0034 期初权益入账机制（venue 接入首账盘点必录——通用径）。
// amount 零假设：必取 venue 自报实时读（mt5=AccountSummary，与对账环
// mt5SummaryOnce 同链同基准），禁字面量/禁手填；eq<=0=拒录（无期初可入，
// 对账自平）。幂等：fin_cashflow source_table='opening_owner_declared' 行在册
// 即跳（重跑不重复入——10-24 手工首录行同标记，天然兼容）。
// 注记来源语义=标记行本身（source_table+'occurred_at' 截时点；禁手改
// fin_cashflow 行——本件是唯一合法写入径）。
// okx/其余 venue 读件未接：fail-loud——补接须自 cmd/core venue_readers 抽
// 共享件（单口径律禁平行实现第二聚合），另案。
// 用法：DB_DSN=… openingequity -venue mt5 -ccy USD [-record]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/mt5"
	"o/internal/platform/config"
)

func main() {
	venue := flag.String("venue", "", "venue 域（如 mt5）")
	ccy := flag.String("ccy", "", "本位币（mt5=USD/okx=USDT——禁混算）")
	record := flag.Bool("record", false, "落账（缺省=干跑——只查报不写）")
	flag.Parse()
	if *venue == "" || *ccy == "" {
		fmt.Fprintln(os.Stderr, "-venue/-ccy 必填")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer pool.Close()

	if id, amt, ok, err := openingExists(ctx, pool, *venue, *ccy); err != nil {
		fmt.Fprintln(os.Stderr, "查重败:", err)
		os.Exit(1)
	} else if ok {
		fmt.Printf("opening 在册 id=%d amount=%.2f %s/%s——幂等跳过（重跑不重复入）\n", id, amt, *venue, *ccy)
		return
	}

	eq, err := liveEquity(ctx, pool, *venue)
	if err != nil {
		fmt.Fprintln(os.Stderr, "venue 自报读败（fail-closed 禁入账）:", err)
		os.Exit(1)
	}
	if eq <= 0 {
		fmt.Fprintf(os.Stderr, "equity=%.2f<=0——无期初可入（对账自平，不落零行）\n", eq)
		os.Exit(1)
	}
	fmt.Printf("%s/%s venue 自报 equity=%.2f（%s）\n", *venue, *ccy, eq, time.Now().UTC().Format("2006-01-02 15:04Z"))
	if !*record {
		fmt.Println("干跑——加 -record 落账")
		return
	}
	id, err := recordOpening(ctx, pool, *venue, *ccy, eq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "落账败:", err)
		os.Exit(1)
	}
	fmt.Printf("已入账 fin_cashflow id=%d kind=deposit amount=%.2f（source_table=opening_owner_declared）\n", id, eq)
}

// liveEquity venue 自报实时读（零假设数源）——mt5=AccountSummary.Equity，
// 与对账环 mt5SummaryOnce 同链（cmd/core/nav_live.go 同调用面）。
func liveEquity(ctx context.Context, pool *pgxpool.Pool, venue string) (float64, error) {
	switch venue {
	case "mt5":
		cfg := config.New(pool)
		if err := cfg.Load(ctx); err != nil {
			return 0, fmt.Errorf("cfg load: %w", err)
		}
		cl, err := mt5.OpenSession(ctx, pool, cfg)
		if err != nil {
			return 0, err
		}
		defer cl.Close()
		eq, _, _, err := cl.AccountSummary(ctx)
		return eq, err
	default:
		return 0, fmt.Errorf("venue %q 读件未接——补接须自 cmd/core venue_readers 抽共享件（单口径律），另案", venue)
	}
}

// openingExists 期初行在册判（幂等锚）——source_table=opening_owner_declared
// 标记行（手工首录与机制录同标记，一册无二径）。
func openingExists(ctx context.Context, pool *pgxpool.Pool, venue, ccy string) (int64, float64, bool, error) {
	var id int64
	var amt float64
	err := pool.QueryRow(ctx, `SELECT id,amount::float8 FROM fin_cashflow
		WHERE venue=$1 AND ccy=$2 AND source_table='opening_owner_declared'
		ORDER BY id LIMIT 1`, venue, ccy).Scan(&id, &amt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, false, nil // 无期初=判缺非错（首录正路）
		}
		return 0, 0, false, err
	}
	return id, amt, true, nil
}

// recordOpening 期初入账（唯一合法写入径——禁手改 fin_cashflow 行的机制面）。
// class 缺省由 fin_0075 trigger 盖册（deposit→transfer）。
func recordOpening(ctx context.Context, pool *pgxpool.Pool, venue, ccy string, eq float64) (int64, error) {
	var id int64
	err := pool.QueryRow(ctx, `INSERT INTO fin_cashflow(kind,venue,ccy,amount,source_ref,source_table,occurred_at)
		VALUES('deposit',$1,$2,$3,0,'opening_owner_declared',now()) RETURNING id`,
		venue, ccy, eq).Scan(&id)
	return id, err
}
