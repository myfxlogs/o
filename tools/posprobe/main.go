// posprobe —— T-0511 取证探针（服务端跑，凭据全程不出机）：
// 对 fin_guard_close 滞留两仓逐腿复现 sizer 现读——判定哪条腿读败/对齐零。
// 只印量与乐器名，禁印凭据。
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/data/okxpriv"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
	"o/internal/trading/execwiring"
	"o/internal/trading/okxexec"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, os.Getenv("DB_DSN"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "pool:", err)
		os.Exit(1)
	}
	defer pool.Close()

	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "cfg:", err)
		os.Exit(1)
	}

	priv, err := okxpriv.NewCollector(ctx, pool, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, "okxpriv:", err)
		os.Exit(1)
	}
	cli := okxexec.NewClient(priv.Credentials(), false)
	sz := execwiring.OkxSizer{C: cli}

	fmt.Println("=== okx 腿现读（生产同径 execwiring.OkxSizer）===")
	for _, inst := range []string{"MUU-USDT", "XMUU-USDT", "MUU-USDT-SWAP", "BTC-USDT-SWAP"} {
		fmt.Printf("-- %s\n", inst)
		if v, err := sz.SpotAvailBal(ctx, inst); err != nil {
			fmt.Printf("   SpotAvailBal: ERR %v\n", err)
		} else {
			fmt.Printf("   SpotAvailBal: %.8f\n", v)
		}
		if v, err := sz.ContractPos(ctx, inst); err != nil {
			fmt.Printf("   ContractPos : ERR %v\n", err)
		} else {
			fmt.Printf("   ContractPos : %.8f\n", v)
		}
		if v, err := sz.LotSz(ctx, inst); err != nil {
			fmt.Printf("   LotSz       : ERR %v\n", err)
		} else {
			fmt.Printf("   LotSz       : %.8f\n", v)
		}
	}

	fmt.Println("=== mt5 OpenedOrders 实读 ===")
	cl, err := mt5.OpenSession(ctx, pool, cfg)
	if err != nil {
		fmt.Println("mt5 session:", err)
		os.Exit(1)
	}
	defer cl.Disconnect(ctx)
	or, err := cl.MT5().OpenedOrders(ctx, &mt5pb.OpenedOrdersRequest{})
	if err != nil {
		fmt.Println("OpenedOrders:", err)
		os.Exit(1)
	}
	fmt.Printf("opened rows=%d\n", len(or.GetResult()))
	for _, o := range or.GetResult() {
		fmt.Printf("  ticket=%d sym=%s lots=%.4f comment=%q\n",
			o.GetTicket(), o.GetSymbol(), o.GetLots(), o.GetComment())
	}
}
