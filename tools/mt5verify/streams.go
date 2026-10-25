// streams.go —— mt5verify 拆件：流面试接段（订阅后并发开窗+TickHistory）
// （行数棘轮自主拆分，行为与单文件版逐行同源）。
package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// sres 流面单流结果；sresM 流面结果（包级——main 判定输出段消费；verifyStreams 填充）。
type sres struct {
	n     int
	first string
	end   error
}

var sresM map[string]*sres

func verifyStreams(ctx context.Context, win time.Duration) {
	// ══ 流面（订阅后并发开窗）══
	fmt.Printf("\n=== 流面试接（窗 %v，已先行订阅）===\n", win)
	sresM = map[string]*sres{"Events": {}, "OnQuote": {}, "OnOrderUpdate": {}, "OnOrderProfit": {}, "OnTickValue": {}, "OnMarketWatch": {}, "OnOpenedOrdersTickets": {}, "OnTickHistory": {}, "OnMail": {}}
	var sresMu sync.Mutex
	var swg sync.WaitGroup
	stream := func(name string, open func(context.Context) recver) {
		defer swg.Done()
		sctx, cancel := context.WithTimeout(ctx, win)
		defer cancel()
		rcv := open(sctx)
		n := 0
		first := ""
		for {
			m, err := rcv()
			if err != nil {
				sresMu.Lock()
				sresM[name].n, sresM[name].first, sresM[name].end = n, first, err
				sresMu.Unlock()
				return
			}
			if n == 0 {
				first = m
			}
			n++
		}
	}
	// TickHistory: 先发请求再收帧
	go func() {
		cctx, cc3 := context.WithTimeout(ctx, 10*time.Second)
		defer cc3()
		_, _ = tkh.TickHistoryRequest(cctx, &mt5pb.TickHistoryRequestRequest{Id: token, Symbol: sym, StartYear: 2026, StartMonth: 10, StartDay: 24})
	}()

	swg.Add(9)
	go stream("Events", func(c context.Context) recver {
		s, err := strm.Events(c, &mt5pb.EventsRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(ev.GetResult()), nil
		}
	})
	go stream("OnQuote", func(c context.Context) recver {
		s, err := strm.OnQuote(c, &mt5pb.OnQuoteRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			if q := ev.GetResult(); q != nil {
				return fmt.Sprintf("bid=%.2f ask=%.2f", q.Bid, q.Ask), nil
			}
			return "<nil>", nil
		}
	})
	go stream("OnOrderUpdate", func(c context.Context) recver {
		s, err := strm.OnOrderUpdate(c, &mt5pb.OnOrderUpdateRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	go stream("OnOrderProfit", func(c context.Context) recver {
		s, err := strm.OnOrderProfit(c, &mt5pb.OnOrderProfitRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			p := ev.GetResult()
			if p == nil {
				return "<nil>", nil
			}
			return fmt.Sprintf("eq=%.2f profit=%.2f orders=%d", p.Equity, p.Profit, len(p.Orders)), nil
		}
	})
	go stream("OnTickValue", func(c context.Context) recver {
		s, err := strm.OnTickValue(c, &mt5pb.OnTickValueRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	go stream("OnMarketWatch", func(c context.Context) recver {
		s, err := strm.OnMarketWatch(c, &mt5pb.OnMarketWatchRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	go stream("OnOpenedOrdersTickets", func(c context.Context) recver {
		s, err := strm.OnOpenedOrdersTickets(c, &mt5pb.OnOpenedOrdersTicketsRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	go stream("OnTickHistory", func(c context.Context) recver {
		s, err := tkh.OnTickHistory(c, &mt5pb.OnTickHistoryRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	go stream("OnMail", func(c context.Context) recver {
		s, err := strm.OnMail(c, &mt5pb.OnMailRequest{Id: token})
		return func() (string, error) {
			if err != nil {
				return "", err
			}
			ev, err := s.Recv()
			if err != nil {
				return "", err
			}
			return short(fmt.Sprintf("%+v", ev.GetResult())), nil
		}
	})
	swg.Wait()
}
