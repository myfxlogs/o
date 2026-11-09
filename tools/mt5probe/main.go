// mt5probe —— 免鉴权 gateway 面探针：Search(公司)/Version/Ping。
// 用途：Connect 前段寻址核验（券商服务器名迁移侦测）。
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	mt5pb "o/internal/gen/mtapi/mt5"
)

func ip443() *int32 { p := int32(443); return &p }

func main() {
	gw := flag.String("gw", "mt5grpc3.mtapi.io:443", "")
	company := flag.String("company", "Exness", "")
	login := flag.Uint64("login", 0, "")
	pass := flag.String("pass", "", "")
	server := flag.String("server", "", "")
	host := flag.String("host", "", "")
	servers := flag.String("servers", "", "逗号分隔 ConnectEx 扫描名单")
	symFilter := flag.String("syms", "", "登录后 Symbols 子串过滤印名")
	flag.Parse()

	cc, err := grpc.NewClient(*gw, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	if err != nil {
		fmt.Fprintln(os.Stderr, "dial:", err)
		os.Exit(1)
	}
	defer cc.Close()
	svc := mt5pb.NewServiceClient(cc)
	_ = *host
	conn := mt5pb.NewConnectionClient(cc)

	if *symFilter != "" && *login != 0 {
		m5 := mt5pb.NewMT5Client(cc)
		cx, cl := context.WithTimeout(metadata.NewOutgoingContext(context.Background(),
			metadata.New(map[string]string{"id": "probe-c"})), 15*time.Second)
		r, err := conn.ConnectEx(cx, &mt5pb.ConnectExRequest{
			User: *login, Password: *pass, Server: *server})
		cl()
		if err != nil || r.GetError() != nil {
			fmt.Println("connect:", err, r.GetError())
			os.Exit(1)
		}
		tok := r.GetResult()
		sx, sc := context.WithTimeout(metadata.NewOutgoingContext(context.Background(),
			metadata.New(map[string]string{"id": tok})), 15*time.Second)
		sl, err := m5.SymbolList(sx, &mt5pb.SymbolListRequest{Id: tok})
		sc()
		if err != nil {
			fmt.Println("Symbols:", err)
			os.Exit(1)
		}
		for _, n := range sl.GetResult() {
			if strings.Contains(strings.ToUpper(n), strings.ToUpper(*symFilter)) {
				fmt.Println(n)
			}
		}
		return
	}
	if *servers != "" && *login != 0 {
		for _, sv := range strings.Split(*servers, ",") {
			cx, cl := context.WithTimeout(metadata.NewOutgoingContext(context.Background(),
				metadata.New(map[string]string{"id": "probe-c"})), 15*time.Second)
			r, err := conn.ConnectEx(cx, &mt5pb.ConnectExRequest{
				User: *login, Password: *pass, Server: strings.TrimSpace(sv)})
			cl()
			switch {
			case err != nil:
				fmt.Printf("%-28s rpcerr: %v\n", sv, err)
			case r.GetError() != nil:
				fmt.Printf("%-28s apierr: %s\n", sv, r.GetError().GetMessage())
			default:
				fmt.Printf("%-28s CONNECTED\n", sv)
			}
		}
		return
	}

	ctx, cancel := context.WithTimeout(metadata.NewOutgoingContext(context.Background(),
		metadata.New(map[string]string{"id": "probe-1"})), 20*time.Second)
	defer cancel()

	if v, err := svc.Version(ctx, &mt5pb.VersionRequest{}); err != nil {
		fmt.Println("Version err:", err)
	} else {
		fmt.Println("Version:", v.GetResult(), v.GetError())
	}

	if ph, err := svc.PingHost(ctx, &mt5pb.PingHostRequest{Host: "48.222.251.98", Port: ip443()}); err != nil {
		fmt.Println("PingHost err:", err)
	} else {
		fmt.Println("PingHost(48.222.251.98):", ph.GetResult(), "ms", ph.GetError())
	}
	r, err := svc.Search(ctx, &mt5pb.SearchRequest{Company: *company})
	if err != nil {
		fmt.Println("Search err:", err)
		os.Exit(1)
	}
	if r.GetError() != nil {
		fmt.Println("Search apierr:", r.GetError())
	}
	for _, co := range r.GetResult() {
		for _, rs := range co.GetResults() {
			fmt.Printf("name=%q access=%v\n", rs.GetName(), rs.GetAccess())
		}
	}
}
