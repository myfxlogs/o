// tfprobe —— O-38639 取证：PriceHistoryExMany timeFrame 方言实测——
// 候选值 {1,60,240,1440,10080,16408,32769} 各拉 EURUSDm 少量棒，
// 量末三棒 ts 间距定网关 timeFrame 语义（分钟?MQL5枚举?）。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
)

type sess struct {
	Addr  string `json:"addr"`
	Token string `json:"token"`
}

func main() {
	dsn := os.Getenv("FIN_PG_DSN")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pg:", err)
		os.Exit(1)
	}
	defer pool.Close()
	var raw string
	if err := pool.QueryRow(ctx, `SELECT value FROM app_config WHERE key='mt5.session_token'`).Scan(&raw); err != nil {
		fmt.Fprintln(os.Stderr, "session:", err)
		os.Exit(1)
	}
	var ss sess
	json.Unmarshal([]byte(raw), &ss)
	cl, err := mt5.Dial(ss.Addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer cl.Close()
	cl.SetSharedToken(ss.Token)
	if err := cl.CheckAlive(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "alive:", err)
		os.Exit(1)
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, 2).Format("2006-01-02T15:04:05Z")
	for _, tf := range []int32{1, 60, 240, 1440, 10080, 16408, 32769} {
		r, err := mt5pb.NewQuoteHistoryClient(cl.Conn()).PriceHistoryExMany(cl.Authed(ctx),
			&mt5pb.PriceHistoryExManyRequest{
				Id: cl.Token(), Symbol: []string{"EURUSDm"}, From: from,
				NumBars: 8, TimeFrame: tf})
		if err != nil {
			fmt.Printf("tf=%d RPC err: %v\n", tf, err)
			continue
		}
		if r.GetError() != nil {
			fmt.Printf("tf=%d apierr: %v\n", tf, r.GetError())
			continue
		}
		for _, res := range r.GetResult() {
			bars := res.GetBars()
			fmt.Printf("tf=%d sym=%s n=%d exc=%q", tf, res.GetSymbol(), len(bars), res.GetException())
			for i, b := range bars {
				if i < len(bars)-3 {
					continue
				}
				fmt.Printf(" [%s c=%.5f]", b.GetTime().AsTime().UTC().Format("01-02 15:04"), b.GetClosePrice())
			}
			fmt.Println()
		}
	}
}
