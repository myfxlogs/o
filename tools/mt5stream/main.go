// mt5stream —— MT5 账户信息流探测：登录→AccountSummary/Account 基线→逐流试接
// （Events/OnOrderUpdate/OnOrderProfit/OnOpenedOrdersTickets/OnMarketWatch/OnTickValue）。
// 判定目标：账户摘要（equity/margin）有无推送流。
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

type recver func() (string, error)

func main() {
	login := flag.Uint64("login", 0, "MT5 login")
	pass := flag.String("pass", "", "MT5 password")
	server := flag.String("server", "", "MT5 server 名")
	host := flag.String("host", "", "MT5 host IP（备径）")
	gw := flag.String("gw", "mt5grpc3.mtapi.io:443", "mtapi 网关")
	win := flag.Duration("win", 15*time.Second, "每流监听窗")
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

	connM := mt5pb.NewConnectionClient(cc)
	mt5c := mt5pb.NewMT5Client(cc)
	strm := mt5pb.NewStreamsClient(cc)

	sid := fmt.Sprintf("probe-%d", time.Now().UnixNano())
	cctx, ccancel := context.WithTimeout(
		metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": sid})), 30*time.Second)
	var token string
	if *server != "" {
		r, err := connM.ConnectEx(cctx, &mt5pb.ConnectExRequest{User: *login, Password: *pass, Server: *server})
		if err != nil {
			fmt.Fprintln(os.Stderr, "ConnectEx:", err)
		} else if r.Error != nil {
			fmt.Fprintln(os.Stderr, "ConnectEx err:", r.Error.Message)
		} else {
			token = r.Result
			fmt.Println("ConnectEx ok (server 名径)")
		}
	}
	if token == "" && *host != "" {
		r, err := connM.Connect(cctx, &mt5pb.ConnectRequest{User: *login, Password: *pass, Host: *host, Port: 443})
		if err != nil || r.Error != nil {
			fmt.Fprintln(os.Stderr, "Connect 败:", err, r.GetError())
			os.Exit(1)
		}
		token = r.Result
		fmt.Println("Connect ok (host 径)")
	}
	ccancel()
	if token == "" {
		fmt.Fprintln(os.Stderr, "无 token")
		os.Exit(1)
	}
	fmt.Printf("token 取得（len=%d）\n\n", len(token))
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": token}))

	for i := 0; i < 3; i++ {
		sctx, scancel := context.WithTimeout(ctx, 15*time.Second)
		s, err := mt5c.AccountSummary(sctx, &mt5pb.AccountSummaryRequest{Id: token})
		scancel()
		if err != nil {
			fmt.Printf("AccountSummary 拍%d RPC err: %v\n", i, err)
		} else if s.Error != nil {
			fmt.Printf("AccountSummary 拍%d error: %s\n", i, s.Error.Message)
		} else if s.Result != nil {
			fmt.Printf("AccountSummary 拍%d: equity=%.2f free_margin=%.2f balance=%.2f margin=%.2f leverage=%.0f profit=%.2f\n",
				i, s.Result.Equity, s.Result.FreeMargin, s.Result.Balance, s.Result.Margin, s.Result.Leverage, s.Result.Profit)
			break
		} else {
			fmt.Printf("AccountSummary 拍%d: 空体（result=nil error=nil）\n", i)
		}
		time.Sleep(2 * time.Second) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
	}
	actx, acancel := context.WithTimeout(ctx, 15*time.Second)
	a, err := mt5c.Account(actx, &mt5pb.AccountRequest{Id: token})
	acancel()
	if err != nil {
		fmt.Println("Account RPC err:", err)
	} else if a.Error != nil {
		fmt.Println("Account error:", a.Error.Message)
	} else if a.Result != nil {
		fmt.Printf("Account: login=%d balance=%.2f credit=%.2f leverage=%d type=%s user=%s\n",
			a.Result.Login, a.Result.Balance, a.Result.Credit, a.Result.Leverage, a.Result.Type, a.Result.UserName)
	}

	// 持仓基线
	octx, ocancel := context.WithTimeout(ctx, 15*time.Second)
	o, err := mt5c.OpenedOrders(octx, &mt5pb.OpenedOrdersRequest{Id: token})
	ocancel()
	if err != nil {
		fmt.Println("OpenedOrders RPC err:", err)
	} else if o.Error != nil {
		fmt.Println("OpenedOrders error:", o.Error.Message)
	} else {
		fmt.Printf("OpenedOrders: %d 持仓\n", len(o.Result))
		for _, od := range o.Result {
			fmt.Printf("  ticket=%d sym=%s lots=%.2f open_px=%.2f profit=%.2f swap=%.2f\n",
				od.Ticket, od.Symbol, od.Lots, od.OpenPrice, od.Profit, od.Swap)
		}
	}

	// 窗内 AccountSummary 3 拍对拍权益漂移（诊断件——一次性测试脚本豁免轮询禁）
	go func() {
		for i := 1; i <= 3; i++ {
			time.Sleep(*win / 3) // sched-exempt-ok 探针 settle/退避等待——一次性证据采集件非工作环
			sctx, scancel := context.WithTimeout(context.Background(), 10*time.Second)
			s, err := mt5c.AccountSummary(sctx, &mt5pb.AccountSummaryRequest{Id: token})
			scancel()
			if err == nil && s.Result != nil {
				fmt.Printf("[pull t+%v] equity=%.2f free_margin=%.2f profit=%.2f\n",
					time.Duration(i)*(*win/3), s.Result.Equity, s.Result.FreeMargin, s.Result.Profit)
			}
		}
	}()

	fmt.Printf("\n=== 逐流试接（窗 %v）===\n", *win)
	var wg sync.WaitGroup
	report := func(name string, frames int, first string, end error) {
		st := "窗毕无帧"
		if end != nil {
			st = end.Error()
		}
		fmt.Printf("%-24s frames=%d  first=%.120s  end=%s\n", name, frames, first, st)
	}
	launch := func(name string, open func(context.Context) recver) {
		defer wg.Done()
		sctx, cancel := context.WithTimeout(ctx, *win)
		defer cancel()
		rcv := open(sctx)
		n := 0
		first := ""
		for {
			msg, err := rcv()
			if err != nil {
				report(name, n, first, err)
				return
			}
			if n == 0 {
				first = msg
			}
			n++
		}
	}
	wg.Add(6)
	go launch("Events", func(c context.Context) recver {
		s, err := strm.Events(c, &mt5pb.EventsRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return ev.Result, nil
		}
	})
	go launch("OnOrderUpdate", func(c context.Context) recver {
		s, err := strm.OnOrderUpdate(c, &mt5pb.OnOrderUpdateRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if ev.Result != nil {
				return fmt.Sprintf("%+v", ev.Result), nil
			}
			return "<nil result>", nil
		}
	})
	go launch("OnOrderProfit", func(c context.Context) recver {
		s, err := strm.OnOrderProfit(c, &mt5pb.OnOrderProfitRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if ev.Result != nil {
				return fmt.Sprintf("%+v", ev.Result), nil
			}
			return "<nil result>", nil
		}
	})
	go launch("OnOpenedOrdersTickets", func(c context.Context) recver {
		s, err := strm.OnOpenedOrdersTickets(c, &mt5pb.OnOpenedOrdersTicketsRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if ev.Result != nil {
				return fmt.Sprintf("%+v", ev.Result), nil
			}
			return "<nil result>", nil
		}
	})
	go launch("OnMarketWatch", func(c context.Context) recver {
		s, err := strm.OnMarketWatch(c, &mt5pb.OnMarketWatchRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if ev.Result != nil {
				return fmt.Sprintf("sym=%s bid=%.5f ask=%.5f", ev.Result.Symbol, ev.Result.Bid, ev.Result.Ask), nil
			}
			return "<nil result>", nil
		}
	})
	go launch("OnTickValue", func(c context.Context) recver {
		s, err := strm.OnTickValue(c, &mt5pb.OnTickValueRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if ev.Result != nil {
				return fmt.Sprintf("%+v", ev.Result), nil
			}
			return "<nil result>", nil
		}
	})
	wg.Wait()
}
