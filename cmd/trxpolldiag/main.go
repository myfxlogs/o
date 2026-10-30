// trxpolldiag —— T-0192 流水零行诊断件（一次性运维）：生产同构 TxsCollector
// 单次 Poll 全链取证（fetch/unmarshal/classify/upsert 分段打印）。只读链上
// +写 fin_trx_txs 真账（真实回填非污染）。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"o/internal/data/trx"
	"o/internal/store/pgstore"
)

func main() {
	dsn := flag.String("dsn", "", "O 库 DSN")
	addr := flag.String("addr", "TWsbXR6SPvvXPxUtPMQvEj7AMH55555555", "TRX 地址")
	key := flag.String("key", "", "trongrid key")
	flag.Parse()
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "-dsn 必填")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		die(err)
	}
	defer pool.Close()
	st := pgstore.NewStore(pool)
	c := trx.NewTxsCollector(*addr, st, "https://api.trongrid.io",
		func() string { return *key }, "https://apilist.tronscan.org", nil, nil)
	stageProbe(*key, *addr)
	var lastTs, firstTs interface{}
	_ = pool.QueryRow(ctx, "SELECT max(ts),min(ts) FROM fin_trx_txs").Scan(&lastTs, &firstTs)
	fmt.Println("poll 前表态: max=", lastTs, " min=", firstTs)
	facts, err := c.Poll(ctx)
	fmt.Printf("Poll 返回: err=%v facts=%d\n", err, len(facts))
	if err != nil {
		os.Exit(1)
	}
	var after int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM fin_trx_txs").Scan(&after); err != nil {
		die(err)
	}
	fmt.Println("poll 后行数:", after)
	if err := pool.QueryRow(ctx, "SELECT max(ts),min(ts) FROM fin_trx_txs").Scan(&lastTs, &firstTs); err != nil {
		die(err)
	}
	fmt.Println("poll 后表态: max=", lastTs, " min=", firstTs)
}
func timeSeconds() (d time.Duration) { return 30 * time.Second }

func die(e error) { fmt.Fprintln(os.Stderr, e); os.Exit(1) }
