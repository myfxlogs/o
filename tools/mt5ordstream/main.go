// mt5ordstream —— OnOrderUpdate 内容实证：流开着的同时真户下单一笔再平，
// 观测订单事件帧（「实现但内容未证」格子的终结实验）。
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func fptr(v float64) *float64 { return &v }
func sptr(v string) *string   { return &v }
func errStr(e *mt5pb.Error) string {
	if e == nil {
		return ""
	}
	return e.Message
}

func main() {
	login := flag.Uint64("login", 0, "")
	pass := flag.String("pass", "", "")
	server := flag.String("server", "", "")
	gw := flag.String("gw", "mt5grpc3.mtapi.io:443", "")
	sym := flag.String("sym", "BTCUSDm", "")
	flag.Parse()
	if *login == 0 || *pass == "" {
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
	strm := mt5pb.NewStreamsClient(cc)
	subs := mt5pb.NewSubscriptionsClient(cc)

	sid := fmt.Sprintf("ordstream-%d", time.Now().UnixNano())
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
	fmt.Println("connected")

	sctx3, sc3 := context.WithTimeout(ctx, 10*time.Second)
	sr, _ := subs.SubscribeOrderUpdate(sctx3, &mt5pb.SubscribeOrderUpdateRequest{Id: token})
	sc3()
	fmt.Println("SubscribeOrderUpdate:", errStr(sr.GetError()))

	sctx, scancel := context.WithTimeout(ctx, 45*time.Second)
	defer scancel()
	var wg sync.WaitGroup
	frames := make(chan string, 64)
	wg.Add(1)
	go func() {
		defer wg.Done()
		s, err := strm.OnOrderUpdate(sctx, &mt5pb.OnOrderUpdateRequest{Id: token})
		if err != nil {
			frames <- "OPEN-ERR: " + err.Error()
			return
		}
		for {
			ev, err := s.Recv()
			if err != nil {
				frames <- "END: " + err.Error()
				return
			}
			if r := ev.GetResult(); r != nil {
				upd := r.GetUpdate()
				frames <- fmt.Sprintf("FRAME upd=%+v eq=%.2f fm=%.2f bal=%.2f profit=%.2f opened=%d",
					upd, r.GetEquity(), r.GetFreeMargin(), r.GetBalance(), r.GetProfit(), len(r.GetOpenedOrders()))
			} else if ev.Error != nil {
				frames <- "ERRF: " + ev.Error.Message
			} else {
				frames <- "<nil frame>"
			}
		}
	}()

	time.Sleep(3 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环

	octx3, oc3 := context.WithTimeout(ctx, 20*time.Second)
	od, err := trd.OrderSend(octx3, &mt5pb.OrderSendRequest{Id: token, Symbol: *sym,
		Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.01, Stoploss: fptr(0), Takeprofit: fptr(0),
		Comment: sptr("ordstream")})
	oc3()
	var tk int64
	if err == nil && od.GetError() == nil {
		tk = od.GetResult().GetTicket()
		fmt.Printf("SEND ok ticket=%d\n", tk)
	} else {
		fmt.Println("SEND err:", err, od.GetError())
	}
	if tk != 0 {
		time.Sleep(4 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
		cctx2, c2 := context.WithTimeout(ctx, 20*time.Second)
		lots := 0.01
		cl, err := trd.OrderClose(cctx2, &mt5pb.OrderCloseRequest{Id: token, Ticket: tk, Lots: &lots})
		c2()
		if err == nil && cl.GetError() == nil {
			fmt.Printf("CLOSE ok ticket=%d\n", tk)
		} else {
			fmt.Println("CLOSE err:", err, cl.GetError())
		}
	}

	time.Sleep(8 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	scancel()
	wg.Wait()
	close(frames)
	fmt.Println("\n=== OnOrderUpdate 帧录 ===")
	n := 0
	for f := range frames {
		n++
		fmt.Printf("[%d] %s\n", n, f)
	}
	if n == 0 {
		fmt.Println("零帧——OnOrderUpdate 在真实订单事件中仍静默")
	}

	octx, ocx := context.WithTimeout(context.Background(), 15*time.Second)
	o, oerr := mt5c.OpenedOrders(octx, &mt5pb.OpenedOrdersRequest{Id: token})
	ocx()
	if oerr != nil || o.GetError() != nil {
		fmt.Printf("\n收尾在册 UNKNOWN——OpenedOrders err=%v body=%v（fail-closed 禁猜 0）\n", oerr, o.GetError())
	} else {
		fmt.Printf("\n收尾在册 %d 仓\n", len(o.GetResult()))
	}
	dx, dx2 := context.WithTimeout(context.Background(), 8*time.Second)
	_, _ = conn.Disconnect(dx, &mt5pb.DisconnectRequest{Id: token})
	dx2()
}
