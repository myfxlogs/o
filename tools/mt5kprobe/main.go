// mt5kprobe —— O-35162 取证：共享会话消费腿 PriceHistoryExMany 实测——
// From=末锚语义下 venue 回棒窗末端在哪（md_klines venue='mt5' 棒截
// 2026-04-19 而 recv_ts 现役——判 venue 史深/网关锚语义/会话缓存）。
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
	for _, nb := range []int32{500, 200, 60} {
		from := now.AddDate(0, 0, 2).Format("2006-01-02T15:04:05Z")
		r, err := mt5pb.NewQuoteHistoryClient(cl.Conn()).PriceHistoryExMany(cl.Authed(ctx),
			&mt5pb.PriceHistoryExManyRequest{
				Id: cl.Token(), Symbol: []string{"XAUUSDm", "BTCUSDm"}, From: from,
				NumBars: nb, TimeFrame: 16408})
		if err != nil {
			fmt.Printf("nb=%d RPC err: %v\n", nb, err)
			continue
		}
		if r.GetError() != nil {
			fmt.Printf("nb=%d apierr: %v\n", nb, r.GetError())
			continue
		}
		for _, res := range r.GetResult() {
			bars := res.GetBars()
			fmt.Printf("nb=%d sym=%s n=%d exc=%q", nb, res.GetSymbol(), len(bars), res.GetException())
			if len(bars) > 0 {
				fmt.Printf(" first=%s last=%s",
					bars[0].GetTime().AsTime().UTC().Format("2006-01-02"),
					bars[len(bars)-1].GetTime().AsTime().UTC().Format("2006-01-02"))
			}
			fmt.Println()
		}
	}
}
