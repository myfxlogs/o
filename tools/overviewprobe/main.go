// overviewprobe —— T-0086 只读验收件：RPC Overview 实拉（生产三分卡/NAV 口径对拍）。
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"

	dashboardv1 "phase/internal/gen/dashboardv1"
	"phase/internal/gen/dashboardv1/dashboardv1connect"
)

func main() {
	client := dashboardv1connect.NewCoreServiceClient(
		connect.NewClient(connecthttp.NewTransport(http.DefaultClient, "http://127.0.0.1:50052")))
	o, err := client.Overview(context.Background(), &dashboardv1.OverviewRequest{})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("NAV=%.2f ts=%s\n", o.NavTotal, o.Ts)
	for _, m := range o.Money {
		fmt.Printf("  [%s] %s (%s) = %.4f legacy=%v\n", m.Section, m.Label, m.Ccy, m.Amount, m.Legacy)
	}
}
