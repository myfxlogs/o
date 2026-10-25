// quotes.go —— mt5verify 拆件：quotes/history+订阅族+Many 族对拍段
// （行数棘轮自主拆分，行为与单文件版逐行同源）。
package main

import (
	"context"
	"fmt"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func verifyQuotesSubs(ctx context.Context) {
	// ══ quotes/history ══
	U("quotes", "Symbols", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.Symbols(c, &mt5pb.SymbolsRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "SymbolList", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolList(c, &mt5pb.SymbolListRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "SymbolParams(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolParams(c, &mt5pb.SymbolParamsRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			return st, fmt.Sprintf("group=%s digits=%d tradeMode=%v", r.Result.GetSymbolGroup().GetGroupName(), r.Result.GetSymbolInfo().GetDigits(), r.Result.GetSymbolGroup().GetTradeMode())
		}
		return st, ""
	})
	U("quotes", "SymbolParams(XAUUSD裸名)", "apierr", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolParams(c, &mt5pb.SymbolParamsRequest{Id: token, Symbol: "XAUUSD"})
		st, _ := status(err, r.GetError(), false)
		return st, r.GetError().GetMessage()
	})
	U("quotes", "SymbolParamsMany(混入XAUUSD=全拒)", "apierr", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolParamsMany(c, &mt5pb.SymbolParamsManyRequest{Id: token, Symbol: []string{sym, "XAUUSD"}})
		st, _ := status(err, r.GetError(), false)
		return st, r.GetError().GetMessage()
	})
	U("quotes", "GetQuote(BTCUSDm)", "any", func(c context.Context) (string, string) {
		r, err := mt5c.GetQuote(c, &mt5pb.GetQuoteRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			return st, fmt.Sprintf("bid=%.2f ask=%.2f", r.Result.Bid, r.Result.Ask)
		}
		return st, ""
	})
	U("quotes", "PriceHistory(M1)", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistory(c, &mt5pb.PriceHistoryRequest{Id: token, Symbol: sym, From: "2026-10-23T00:00:00", To: "2026-10-25T00:00:00", TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryEx(M1)", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryEx(c, &mt5pb.PriceHistoryExRequest{Id: token, Symbol: sym, From: "2026-10-23T00:00:00", NumBars: 20, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryToday", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryToday(c, &mt5pb.PriceHistoryTodayRequest{Id: token, Symbol: sym, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryMonth", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryMonth(c, &mt5pb.PriceHistoryMonthRequest{Id: token, Symbol: sym, Year: 2026, Month: 10, Day: 1, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "IsTradeSession(BTCUSDm)", "any", func(c context.Context) (string, string) {
		r, err := mt5c.IsTradeSession(c, &mt5pb.IsTradeSessionRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("inSession=%v", r.GetResult())
	})
	U("quotes", "IsQuoteSession(BTCUSDm)", "any", func(c context.Context) (string, string) {
		r, err := mt5c.IsQuoteSession(c, &mt5pb.IsQuoteSessionRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("inSession=%v", r.GetResult())
	})
	U("quotes", "SymbolSessionsEx(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolSessionsEx(c, &mt5pb.SymbolSessionsExRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		return st, fmt.Sprintf("%v", r.GetResult() != nil)
	})
	U("quotes", "GetTickValueMany", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.GetTickValueMany(c, &mt5pb.GetTickValueManyRequest{Id: token, Symbol: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("vals=%v", r.GetResult())
	})
	U("quotes", "TickValueWithSize", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.TickValueWithSize(c, &mt5pb.TickValueWithSizeRequest{Id: token, Symbol: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})

	// ══ 订阅族 ══
	U("subs", "Subscribe(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := subs.Subscribe(c, &mt5pb.SubscribeRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "SubscribeOrderProfit", "ok", func(c context.Context) (string, string) {
		r, err := subs.SubscribeOrderProfit(c, &mt5pb.SubscribeOrderProfitRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "SubscribeOrderUpdate", "ok", func(c context.Context) (string, string) {
		r, err := subs.SubscribeOrderUpdate(c, &mt5pb.SubscribeOrderUpdateRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "SubscribeTickValue(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := subs.SubscribeTickValue(c, &mt5pb.SubscribeTickValueRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "SubscribeMarketWatch", "unimplemented", func(c context.Context) (string, string) {
		r, err := subs.SubscribeMarketWatch(c, &mt5pb.SubscribeMarketWatchRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
	U("subs", "SubscribeOpenedOrdersTickets", "unimplemented", func(c context.Context) (string, string) {
		r, err := subs.SubscribeOpenedOrdersTickets(c, &mt5pb.SubscribeOpenedOrdersTicketsRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
	U("subs", "SubscribeMany", "ok", func(c context.Context) (string, string) {
		r, err := subs.SubscribeMany(c, &mt5pb.SubscribeManyRequest{Id: token, Symbols: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "UnSubscribe(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := subs.UnSubscribe(c, &mt5pb.UnSubscribeRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("subs", "UnSubscribeMany", "ok", func(c context.Context) (string, string) {
		r, err := subs.UnSubscribeMany(c, &mt5pb.UnSubscribeManyRequest{Id: token, Symbols: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})

	// ══ quotes/history Many 族补员 ══
	U("quotes", "GetQuoteMany", "any", func(c context.Context) (string, string) {
		r, err := mt5c.GetQuoteMany(c, &mt5pb.GetQuoteManyRequest{Id: token, Symbols: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "SymbolSessionsExMany", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.SymbolSessionsExMany(c, &mt5pb.SymbolSessionsExManyRequest{Id: token, Symbol: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "IsTradeSessionMany", "any", func(c context.Context) (string, string) {
		r, err := mt5c.IsTradeSessionMany(c, &mt5pb.IsTradeSessionManyRequest{Id: token, Symbols: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "IsQuoteSessionMany", "any", func(c context.Context) (string, string) {
		r, err := mt5c.IsQuoteSessionMany(c, &mt5pb.IsQuoteSessionManyRequest{Id: token, Symbols: []string{sym}})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryMany", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryMany(c, &mt5pb.PriceHistoryManyRequest{Id: token, Symbol: []string{sym}, From: "2026-10-23T00:00:00", To: "2026-10-25T00:00:00", TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryTodayMany", "any", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryTodayMany(c, &mt5pb.PriceHistoryTodayManyRequest{Id: token, Symbols: []string{sym}, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryExMany", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryExMany(c, &mt5pb.PriceHistoryExManyRequest{Id: token, Symbol: []string{sym}, From: "2026-10-23T00:00:00", NumBars: 10, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "PriceHistoryHighLow", "ok", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryHighLow(c, &mt5pb.PriceHistoryHighLowRequest{Id: token, Symbol: sym, From: "2026-10-23T00:00:00", To: "2026-10-25T00:00:00", TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		if b := r.GetResult(); b != nil {
			return st, fmt.Sprintf("hi=%.2f lo=%.2f", b.GetHighPrice(), b.GetLowPrice())
		}
		return st, r.GetError().GetMessage()
	})
	U("quotes", "PriceHistoryMonthMany", "any", func(c context.Context) (string, string) {
		r, err := qh.PriceHistoryMonthMany(c, &mt5pb.PriceHistoryMonthManyRequest{Id: token, Symbols: []string{sym}, Year: 2026, Month: 10, Day: 1, TimeFrame: 16408})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("bars=%d", len(r.GetResult()))
	})
	U("quotes", "TickHistoryStop", "any", func(c context.Context) (string, string) {
		r, err := tkh.TickHistoryStop(c, &mt5pb.TickHistoryStopRequest{Id: token, Symbol: sym})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})

	claimsWG.Wait()
}
