// reads.go —— mt5verify 拆件：gateway-admin/account/orders 读面声明对拍段
// （行数棘轮自主拆分，行为与单文件版逐行同源）。
package main

import (
	"context"
	"fmt"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func verifyAdminAccountOrders(ctx context.Context, host string) {
	// ══ gateway-admin ══
	U("gateway", "Service.Ping", "ok", func(c context.Context) (string, string) {
		r, err := svc.Ping(c, &mt5pb.PingRequest{})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("gateway", "Service.Version", "ok", func(c context.Context) (string, string) {
		r, err := svc.Version(c, &mt5pb.VersionRequest{})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})
	U("gateway", "Service.Health", "unimplemented", func(c context.Context) (string, string) {
		r, err := svc.Health(c, &mt5pb.HealthRequest{})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
	U("gateway", "Service.Search(Exness)", "ok", func(c context.Context) (string, string) {
		r, err := svc.Search(c, &mt5pb.SearchRequest{Company: "Exness"})
		st, _ := status(err, r.GetError(), len(r.GetResult()) == 0)
		n := 0
		for _, co := range r.GetResult() {
			for _, rs := range co.GetResults() {
				n += len(rs.Access)
			}
		}
		return st, fmt.Sprintf("companies=%d access=%d", len(r.GetResult()), n)
	})
	U("gateway", "GetClients", "apierr", func(c context.Context) (string, string) {
		r, err := svc.GetClients(c, &mt5pb.GetClientsRequest{})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetError())
	})
	U("gateway", "MemorySnapshot", "apierr", func(c context.Context) (string, string) {
		r, err := svc.MemorySnapshot(c, &mt5pb.MemorySnapshotRequest{})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetError())
	})
	U("gateway", "PingHost", "ok", func(c context.Context) (string, string) {
		r, err := svc.PingHost(c, &mt5pb.PingHostRequest{Host: host, Port: iptr(443)})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("ms=%d", r.GetResult())
	})
	U("gateway", "PingHostMany(不配对称)", "err", func(c context.Context) (string, string) {
		r, err := svc.PingHostMany(c, &mt5pb.PingHostManyRequest{Host: []string{host, host}, Port: []int32{443}})
		st, det := status(err, r.GetError(), false)
		return st, det
	})
	U("gateway", "CheckConnect", "ok", func(c context.Context) (string, string) {
		r, err := conn.CheckConnect(c, &mt5pb.CheckConnectRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("%v", r.GetResult())
	})

	// ══ account ══
	U("account", "Account", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.Account(c, &mt5pb.AccountRequest{Id: token})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			return st, fmt.Sprintf("bal=%.2f lev=%d type=%s", r.Result.Balance, r.Result.Leverage, r.Result.Type)
		}
		return st, ""
	})
	U("account", "AccountSummary", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.AccountSummary(c, &mt5pb.AccountSummaryRequest{Id: token})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		if r.GetResult() != nil {
			return st, fmt.Sprintf("eq=%.2f fm=%.2f ccy=%s", r.Result.Equity, r.Result.FreeMargin, r.Result.Currency)
		}
		return st, ""
	})
	U("account", "ServerTimezone", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.ServerTimezone(c, &mt5pb.ServerTimezoneRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("offset=%.1f", r.GetResult())
	})
	U("account", "RequiredMargin(BTCUSDm)", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.RequiredMargin(c, &mt5pb.RequiredMarginRequest{Id: token, Symbol: sym, Lots: 0.01})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("margin=%.2f", r.GetResult())
	})
	U("account", "RequiredMargin(XAUUSD裸名)", "apierr:Symbol not exist", func(c context.Context) (string, string) {
		r, err := mt5c.RequiredMargin(c, &mt5pb.RequiredMarginRequest{Id: token, Symbol: "XAUUSD", Lots: 0.01})
		st, det := status(err, r.GetError(), false)
		if det == "" && r.GetError() != nil {
			det = r.GetError().Message
		}
		return st, r.GetError().GetMessage()
	})
	U("account", "Mails", "unimplemented", func(c context.Context) (string, string) {
		r, err := mt5c.Mails(c, &mt5pb.MailsRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})

	// ══ orders 读面 ══
	U("orders", "OpenedOrders", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.OpenedOrders(c, &mt5pb.OpenedOrdersRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("n=%d", len(r.GetResult()))
	})
	U("orders", "OpenedOrdersTickets", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.OpenedOrdersTickets(c, &mt5pb.OpenedOrdersTicketsRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, fmt.Sprintf("tickets=%v", r.GetResult())
	})
	U("orders", "OpenedOrder(未知ticket)", "expect-empty-or-err", func(c context.Context) (string, string) {
		r, err := mt5c.OpenedOrder(c, &mt5pb.OpenedOrderRequest{Id: token, Ticket: iptr64(999999999)})
		st, _ := status(err, r.GetError(), r.GetResult() == nil)
		return st, fmt.Sprintf("err=%v res=%v", r.GetError(), r.GetResult())
	})
	U("orders", "OrderHistory", "ok", func(c context.Context) (string, string) {
		r, err := mt5c.OrderHistory(c, &mt5pb.OrderHistoryRequest{Id: token, From: "2026-09-01T00:00:00", To: "2027-01-01T00:00:00"})
		st, _ := status(err, r.GetError(), false)
		bal := 0
		for _, o := range r.GetResult() {
			if o.GetOrderType() == mt5pb.OrderType_OrderType_Balance {
				bal++
			}
		}
		return st, fmt.Sprintf("orders=%d balanceRows=%d", len(r.GetResult()), bal)
	})
	U("orders", "OrderHistoryPagination", "unimplemented", func(c context.Context) (string, string) {
		r, err := mt5c.OrderHistoryPagination(c, &mt5pb.OrderHistoryPaginationRequest{Id: token, From: "2026-09-01T00:00:00", To: "2027-01-01T00:00:00", OrdersPerPage: 10, PageNumber: 0})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
	U("orders", "PendingOrderHistory", "unimplemented", func(c context.Context) (string, string) {
		r, err := mt5c.PendingOrderHistory(c, &mt5pb.PendingOrderHistoryRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
	U("orders", "MarketWatchMany", "unimplemented", func(c context.Context) (string, string) {
		r, err := mt5c.MarketWatchMany(c, &mt5pb.MarketWatchManyRequest{Id: token})
		st, _ := status(err, r.GetError(), false)
		return st, ""
	})
}
