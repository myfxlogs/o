// mt5close —— 真户残留仓清场：实读 OpenedOrders→逐票 OrderClose（带 Lots）。
// venue 权威先读后动；仅清 --tickets 指定或全部（--all）。
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	mt5pb "phase/internal/gen/mtapi/mt5"
)

func main() {
	login := flag.Uint64("login", 0, "")
	pass := flag.String("pass", "", "")
	server := flag.String("server", "", "")
	gw := flag.String("gw", "mt5grpc3.mtapi.io:443", "")
	tick := flag.Int64("ticket", 0, "指定票号（0=全部）")
	flag.Parse()
	if *login == 0 || *pass == "" {
		fmt.Fprintln(os.Stderr, "need -login/-pass")
		os.Exit(2)
	}
	ctx := context.Background()
	cc, err := grpc.NewClient(*gw, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer cc.Close()
	conn := mt5pb.NewConnectionClient(cc)
	mt5c := mt5pb.NewMT5Client(cc)
	trd := mt5pb.NewTradingClient(cc)

	sid := fmt.Sprintf("close-%d", time.Now().UnixNano())
	cctx, cc1 := context.WithTimeout(metadata.NewOutgoingContext(ctx,
		metadata.New(map[string]string{"id": sid})), 30*time.Second)
	r, err := conn.ConnectEx(cctx, &mt5pb.ConnectExRequest{User: *login, Password: *pass, Server: *server})
	cc1()
	if err != nil || r.Error != nil {
		fmt.Fprintln(os.Stderr, "ConnectEx:", err, r.GetError())
		os.Exit(1)
	}
	token := r.Result
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": token}))

	octx, oc := context.WithTimeout(ctx, 15*time.Second)
	o, err := mt5c.OpenedOrders(octx, &mt5pb.OpenedOrdersRequest{Id: token})
	oc()
	if err != nil || o.Error != nil {
		fmt.Fprintln(os.Stderr, "OpenedOrders:", err, o.GetError())
		os.Exit(1)
	}
	fmt.Printf("在册 %d 仓:\n", len(o.Result))
	for _, od := range o.Result {
		fmt.Printf("  ticket=%d %s %.2flots @%.2f profit=%.2f\n", od.Ticket, od.Symbol, od.Lots, od.OpenPrice, od.Profit)
	}
	for _, od := range o.Result {
		if *tick != 0 && od.Ticket != *tick {
			continue
		}
		for att := 1; att <= 4; att++ {
			cctx2, c2 := context.WithTimeout(ctx, 20*time.Second)
			lots := od.Lots
			cl, err := trd.OrderClose(cctx2, &mt5pb.OrderCloseRequest{Id: token, Ticket: od.Ticket, Lots: &lots})
			c2()
			if err == nil && cl.GetError() == nil {
				fmt.Printf("CLOSED ticket=%d (att%d)\n", od.Ticket, att)
				break
			}
			msg := ""
			if err != nil {
				msg = err.Error()
			} else {
				msg = cl.GetError().GetMessage()
			}
			fmt.Printf("  close ticket=%d att%d: %s\n", od.Ticket, att, msg)
			time.Sleep(time.Duration(att) * 1500 * time.Millisecond) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
		}
	}
	// 收尾实读
	o2ctx, o2c := context.WithTimeout(ctx, 15*time.Second)
	o2, _ := mt5c.OpenedOrders(o2ctx, &mt5pb.OpenedOrdersRequest{Id: token})
	o2c()
	fmt.Printf("收尾在册 %d 仓\n", len(o2.GetResult()))
	dctx, dc := context.WithTimeout(ctx, 8*time.Second)
	_, _ = conn.Disconnect(dctx, &mt5pb.DisconnectRequest{Id: token})
	dc()
}
