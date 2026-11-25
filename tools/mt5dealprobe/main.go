// mt5dealprobe —— OrderHistory deal 行全字段 dump 探针（O-33207 S0 取证件：
// 手动资金流/手动仓簿记前先实证 deal 行形+sign 约定，禁猜数）。
// 读面零写：ConnectEx 会话+OrderHistory 拉窗逐行印字段——Balance/Credit/
// Charge/Commission 族金额载体字段+Buy/Sell 交易行 Profit/Swap/Commission/
// Fee 分量与 DealInternalIn/Out 逐 deal 载荷全出。
// 用法：DB_DSN+O_MASTER_KEY_FILE 环境（容器内径），-venue/-days/-gw 可参。
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/mt5"
	"o/internal/platform/config"
)

func dealLine(tag string, d *mt5pb.DealInternal) {
	if d == nil {
		return
	}
	fmt.Printf("    %s deal{ticket=%d orderTicket=%d posTicket=%d type=%s dir=%s sym=%s"+
		" px=%.5f vol=%d lots=%.2f profit=%.4f comm=%.4f fee=%.4f swap=%.4f expert=%d comment=%q}\n",
		tag, d.GetTicketNumber(), d.GetOrderTicket(), d.GetPositionTicket(),
		d.GetType(), d.GetDirection(), d.GetSymbol(), d.GetPrice(), d.GetVolume(),
		d.GetLots(), d.GetProfit(), d.GetCommission(), d.GetFee(), d.GetSwap(),
		d.GetExpertId(), d.GetComment())
}

func main() {
	venue := flag.String("venue", "exness", "roster venue token")
	days := flag.Int("days", 60, "回看天数")
	maxRows := flag.Int("max", 200, "行数上限")
	open := flag.Bool("open", false, "只拉 OpenedOrders 在仓 dump（不拉 OrderHistory）")
	flag.Parse()
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "DB_DSN 缺")
		os.Exit(2)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pg:", err)
		os.Exit(1)
	}
	defer pool.Close()
	cl, err := mt5.OpenSession(ctx, pool, config.New(pool), *venue)
	if err != nil {
		fmt.Fprintln(os.Stderr, "session:", err)
		os.Exit(1)
	}
	defer cl.Close()
	if *open {
		rep, err := cl.MT5().OpenedOrders(cl.Authed(ctx), &mt5pb.OpenedOrdersRequest{Id: cl.Token()})
		if err != nil {
			fmt.Fprintln(os.Stderr, "OpenedOrders:", err)
			os.Exit(1)
		}
		if rep.GetError() != nil {
			fmt.Fprintln(os.Stderr, "venue err:", rep.GetError())
			os.Exit(1)
		}
		fmt.Printf("venue=%s openOrders=%d\n", *venue, len(rep.GetResult()))
		for _, o := range rep.GetResult() {
			fmt.Printf("open ticket=%d sym=%s type=%s dealType=%s state=%s"+
				" openPx=%.5f vol=%d lots=%.2f profit=%.4f swap=%.4f comm=%.4f fee=%.4f"+
				" openTs=%d comment=%q expert=%d contractSize=%.1f\n",
				o.GetTicket(), o.GetSymbol(), o.GetOrderType(), o.GetDealType(),
				o.GetState().String(), o.GetOpenPrice(), o.GetVolume(), o.GetLots(),
				o.GetProfit(), o.GetSwap(), o.GetCommission(), o.GetFee(),
				o.GetOpenTimestampUTC(), o.GetComment(), o.GetExpertId(), o.GetContractSize())
			dealLine("in ", o.GetDealInternalIn())
		}
		return
	}
	to := time.Now()
	from := to.Add(-time.Duration(*days) * 24 * time.Hour)
	rep, err := cl.MT5().OrderHistory(cl.Authed(ctx), &mt5pb.OrderHistoryRequest{
		Id:        cl.Token(),
		From:      from.UTC().Format("2006-01-02T15:04:05"),
		To:        to.UTC().Format("2006-01-02T15:04:05"),
		Ascending: proto.Bool(true),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "OrderHistory:", err)
		os.Exit(1)
	}
	if rep.GetError() != nil {
		fmt.Fprintln(os.Stderr, "venue err:", rep.GetError())
		os.Exit(1)
	}
	rows := rep.GetResult()
	fmt.Printf("venue=%s rows=%d window=%s→%s\n", *venue, len(rows),
		from.Format("01-02"), to.Format("01-02"))
	for i, o := range rows {
		if i >= *maxRows {
			fmt.Printf("…截断（余 %d 行）\n", len(rows)-*maxRows)
			break
		}
		ot, ct := int64(0), int64(0)
		if o.OpenTimestampUTC > 0 {
			ot = o.OpenTimestampUTC
		}
		if o.CloseTimestampUTC > 0 {
			ct = o.CloseTimestampUTC
		}
		fmt.Printf("row%d ticket=%d orderType=%s dealType=%s sym=%s dir? state=%s"+
			" openPx=%.5f closePx=%.5f vol=%d closeVol=%d lots=%.2f closeLots=%.2f"+
			" profit=%.4f swap=%.4f comm=%.4f fee=%.4f profitRate=%.4f"+
			" openTs=%d closeTs=%d comment=%q closeComment=%q expert=%d contractSize=%.1f partials=%d\n",
			i, o.GetTicket(), o.GetOrderType(), o.GetDealType(), o.GetSymbol(),
			o.GetState().String(), o.GetOpenPrice(), o.GetClosePrice(), o.GetVolume(),
			o.GetCloseVolume(), o.GetLots(), o.GetCloseLots(),
			o.GetProfit(), o.GetSwap(), o.GetCommission(), o.GetFee(), o.GetProfitRate(),
			ot, ct, o.GetComment(), o.GetCloseComment(), o.GetExpertId(),
			o.GetContractSize(), len(o.GetPartialCloseDeals()))
		dealLine("in ", o.GetDealInternalIn())
		dealLine("out", o.GetDealInternalOut())
		for j, p := range o.GetPartialCloseDeals() {
			dealLine(fmt.Sprintf("pt%d", j), p)
		}
	}
}
