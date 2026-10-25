package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	mt5pb "o/internal/gen/mtapi/mt5"
)

// readDemoEnv demo 凭据 env 解析（值禁打印）。
func readDemoEnv(path string) (uint64, string, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "demoenv:", err)
		os.Exit(2)
	}
	var u uint64
	var p, s string
	for _, ln := range strings.Split(string(b), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(ln), "=")
		if !ok {
			continue
		}
		switch k {
		case "MT5_DEMO_USER":
			u, _ = strconv.ParseUint(v, 10, 64)
		case "MT5_DEMO_PASS":
			p = v
		case "MT5_DEMO_SERVER":
			s = v
		}
	}
	if u == 0 || p == "" || s == "" {
		fmt.Fprintln(os.Stderr, "demoenv incomplete")
		os.Exit(2)
	}
	return u, p, s
}

func fptr(v float64) *float64 { return &v }
func sptr(v string) *string   { return &v }

// runMutating mutating 面全周期验证（demo 户专属——真户永禁）。
// 谱序照 nox mtprobe：市价开→改 SL/TP→平→BuyLimit 深价挂→改价→撤。
func runMutating(ctx context.Context) {
	fmt.Println("\n=== mutating 谱（demo 户）===")
	market := func(name, expect string, f func(context.Context) (string, string)) {
		cctx, c := context.WithTimeout(ctx, 25*time.Second)
		defer c()
		st, det := f(cctx)
		v := "PASS"
		if st != expect {
			v = "DIFF"
		}
		fmt.Printf("%-42s expect=%-14s actual=%-22s %s  %s\n", name, expect, st, v, det)
	}

	var ticket int64
	market("OrderSend 市价 BUY 0.01 BTCUSDm", "ok", func(c context.Context) (string, string) {
		r, err := trd.OrderSend(c, &mt5pb.OrderSendRequest{
			Id: token, Symbol: sym, Operation: mt5pb.OrderType_OrderType_Buy, Volume: 0.01,
			Stoploss: fptr(0), Takeprofit: fptr(0), Comment: sptr("mt5verify-mut")})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			ticket = r.GetResult().GetTicket()
			return st, fmt.Sprintf("ticket=%d", ticket)
		}
		return st, r.GetError().GetMessage()
	})
	if ticket != 0 {
		market("OrderModify SL/TP(ticket)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderModify(c, &mt5pb.OrderModifyRequest{Id: token, Ticket: ticket, Stoploss: 1.0, Takeprofit: 999999.0})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		market("OrderClose(ticket)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderClose(c, &mt5pb.OrderCloseRequest{Id: token, Ticket: ticket})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
	}
	var pt int64
	market("OrderSend BuyLimit 深价挂", "ok", func(c context.Context) (string, string) {
		r, err := trd.OrderSend(c, &mt5pb.OrderSendRequest{
			Id: token, Symbol: sym, Operation: mt5pb.OrderType_OrderType_BuyLimit, Volume: 0.01,
			Price: fptr(1000.0), Comment: sptr("mt5verify-pend")})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			pt = r.GetResult().GetTicket()
			return st, fmt.Sprintf("ticket=%d", pt)
		}
		return st, r.GetError().GetMessage()
	})
	if pt != 0 {
		market("OrderModify 改价(pending)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderModify(c, &mt5pb.OrderModifyRequest{Id: token, Ticket: pt, Price: fptr(2000.0)})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		market("OrderClose 撤挂(pending)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderClose(c, &mt5pb.OrderCloseRequest{Id: token, Ticket: pt})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
	}
}

// runMutatingReal 真户 mutating 谱——业主令授权 2026-10-24。
// 安全断：①禁 BTCUSDm（业主在册仓所在符号——netting 防交叉）②只动自开 ticket
// ③跑前后 OpenedOrders 票集对拍（在册仓一票不许动）。
func runMutatingReal(ctx context.Context, mutSym string) {
	fmt.Println("\n=== mutating 谱（真户·授权）===")
	if mutSym == "" {
		fmt.Println("REJECT: mutsym 必须显式给")
		return
	}
	market := func(name, expect string, f func(context.Context) (string, string)) {
		cctx, c := context.WithTimeout(ctx, 25*time.Second)
		defer c()
		st, det := f(cctx)
		v := "PASS"
		if st != expect {
			v = "DIFF"
		}
		fmt.Printf("%-46s expect=%-14s actual=%-22s %s  %s\n", name, expect, st, v, det)
	}

	// 前置：在册票集 + 隔离符号存在性+FullAccess
	preTickets := map[int64]bool{}
	{
		cctx, c := context.WithTimeout(ctx, 15*time.Second)
		o, err := mt5c.OpenedOrders(cctx, &mt5pb.OpenedOrdersRequest{Id: token})
		c()
		if err == nil {
			for _, od := range o.GetResult() {
				preTickets[od.Ticket] = true
			}
		}
		fmt.Printf("前置在册票集=%v\n", keys(preTickets))
	}
	// 在册仓处置拍（业主令：在册仓即测试料——Modify/Close 全周期验在真仓上）
	for t := range preTickets {
		market(fmt.Sprintf("OrderModify 在册仓(ticket=%d) SL/TP", t), "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderModify(c, &mt5pb.OrderModifyRequest{Id: token, Ticket: t, Stoploss: 1.0, Takeprofit: 999999.0})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		market(fmt.Sprintf("OrderClose 在册仓(ticket=%d)", t), "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderClose(c, &mt5pb.OrderCloseRequest{Id: token, Ticket: t})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		break // 只处置首仓——净库防误清一片
	}

	var tradeOK bool
	vol := 0.01
	market(fmt.Sprintf("SymbolParams(%s) 可交易", mutSym), "ok", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolParams(c, &mt5pb.SymbolParamsRequest{Id: token, Symbol: mutSym})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			g := r.Result.GetSymbolGroup()
			tradeOK = g.GetTradeMode() == mt5pb.TradeMode_TradeMode_FullAccess
			if g.GetMinLots() > vol {
				vol = g.GetMinLots()
			}
			return st, fmt.Sprintf("tradeMode=%v minLots=%.3f group=%s", g.GetTradeMode(), g.GetMinLots(), g.GetGroupName())
		}
		return st, r.GetError().GetMessage()
	})
	if !tradeOK {
		fmt.Println("REJECT: 隔离符号非 FullAccess——mutating 谱止")
		return
	}

	var ticket int64
	market(fmt.Sprintf("OrderSend 市价 BUY 0.01 %s", mutSym), "ok", func(c context.Context) (string, string) {
		r, err := trd.OrderSend(c, &mt5pb.OrderSendRequest{
			Id: token, Symbol: mutSym, Operation: mt5pb.OrderType_OrderType_Buy, Volume: vol,
			Stoploss: fptr(0), Takeprofit: fptr(0), Comment: sptr("mt5verify-realmut")})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			ticket = r.GetResult().GetTicket()
			return st, fmt.Sprintf("ticket=%d", ticket)
		}
		return st, r.GetError().GetMessage()
	})
	if ticket != 0 && preTickets[ticket] {
		fmt.Println("⚠ 开出 ticket 撞在册票号——异常")
	}
	if ticket != 0 {
		market("OrderModify SL/TP", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderModify(c, &mt5pb.OrderModifyRequest{Id: token, Ticket: ticket, Stoploss: 1.0, Takeprofit: 999999.0})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		market("OrderClose 平自开仓", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderClose(c, &mt5pb.OrderCloseRequest{Id: token, Ticket: ticket})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
	}
	var pt int64
	market(fmt.Sprintf("OrderSend BuyLimit 深价挂 %s", mutSym), "ok", func(c context.Context) (string, string) {
		r, err := trd.OrderSend(c, &mt5pb.OrderSendRequest{
			Id: token, Symbol: mutSym, Operation: mt5pb.OrderType_OrderType_BuyLimit, Volume: vol,
			Price: fptr(1.0), Comment: sptr("mt5verify-realmut-pend")})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			pt = r.GetResult().GetTicket()
			return st, fmt.Sprintf("ticket=%d", pt)
		}
		return st, r.GetError().GetMessage()
	})
	if pt != 0 {
		market("OrderModify 改价(pending)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderModify(c, &mt5pb.OrderModifyRequest{Id: token, Ticket: pt, Price: fptr(2.0)})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
		market("OrderClose 撤挂(pending)", "ok", func(c context.Context) (string, string) {
			r, err := trd.OrderClose(c, &mt5pb.OrderCloseRequest{Id: token, Ticket: pt})
			st, _ := status(err, r.GetError(), false)
			return st, r.GetError().GetMessage()
		})
	}

	// 后置：票集对拍——在册仓一票不动
	{
		cctx, c := context.WithTimeout(ctx, 15*time.Second)
		o, err := mt5c.OpenedOrders(cctx, &mt5pb.OpenedOrdersRequest{Id: token})
		c()
		post := map[int64]bool{}
		if err == nil {
			for _, od := range o.GetResult() {
				post[od.Ticket] = true
			}
		}
		verdict := "PASS"
		for t := range preTickets {
			if !post[t] {
				verdict = "DIFF——在册票丢失!"
			}
		}
		for t := range post {
			if !preTickets[t] {
				verdict = "DIFF——残留票未清!"
			}
		}
		fmt.Printf("%-46s %s  后集=%v\n", "后置票集对拍（在册仓一票不动）", verdict, keys(post))
	}
}

func keys(m map[int64]bool) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
