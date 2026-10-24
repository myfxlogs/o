// mt5verify —— mtapi 五 skill 全声明实测对拍 harness。
// 逐条跑 skill 所载能力声明（ok/err/unimplemented），输出 PASS/DIFF/SKIP 表。
// 凭据=flag 传入不落盘；mutating 面真户永禁（D-0009/canon），-demo 才跑。
package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	mt5pb "phase/internal/gen/mtapi/mt5"
)

const sym = "BTCUSDm" // EXR 户方言实认符号

var (
	token string
	mt5c  mt5pb.MT5Client
	svc   mt5pb.ServiceClient
	subs  mt5pb.SubscriptionsClient
	strm  mt5pb.StreamsClient
	qh    mt5pb.QuoteHistoryClient
	tkh   mt5pb.TickHistoryClient
	conn  mt5pb.ConnectionClient
	trd   mt5pb.TradingClient
)

type claim struct {
	skill, name, expect string
}

type recver func() (string, error)

type resRow struct {
	claim
	actual, detail string
}

var results []resRow
var mu sync.Mutex

func rec(c claim, actual, detail string) {
	mu.Lock()
	results = append(results, resRow{c, actual, detail})
	mu.Unlock()
}

// status 归一判定：rpc err / api error / 空体 / ok
func status(err error, e *mt5pb.Error, empty bool) (string, string) {
	if err != nil {
		if strings.Contains(err.Error(), "Unimplemented") {
			return "unimplemented", ""
		}
		return "rpcerr:" + short(err.Error()), ""
	}
	if e != nil {
		return "apierr:" + short(e.Message), ""
	}
	if empty {
		return "empty", ""
	}
	return "ok", ""
}

func short(s string) string {
	if len(s) > 90 {
		return s[:90]
	}
	return s
}

func call(name string, f func(ctx context.Context) (string, string)) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	st, det := f(ctx)
	mu.Lock()
	results = append(results, resRow{claim{name: name}, st, det})
	mu.Unlock()
}

func main() {
	login := flag.Uint64("login", 0, "")
	pass := flag.String("pass", "", "")
	server := flag.String("server", "", "")
	host := flag.String("host", "", "")
	gw := flag.String("gw", "mt5grpc3.mtapi.io:443", "")
	streamWin := flag.Duration("swin", 25*time.Second, "")
	demoEnv := flag.String("demoenv", "", "demo 户 env 文件（MT5_DEMO_*）——置位=只跑 mutating 谱")
	realMut := flag.Bool("realmut", false, "真户 mutating 谱（业主授权面）——隔离符号跑，绝不碰在册持仓")
	mutSym := flag.String("mutsym", "", "mutating 隔离符号（禁 BTCUSDm）")
	flag.Parse()
	if *demoEnv != "" {
		du, dp, ds := readDemoEnv(*demoEnv)
		*login, *pass, *server = du, dp, ds
	}
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
	conn = mt5pb.NewConnectionClient(cc)
	mt5c = mt5pb.NewMT5Client(cc)
	trd = mt5pb.NewTradingClient(cc)
	svc = mt5pb.NewServiceClient(cc)
	subs = mt5pb.NewSubscriptionsClient(cc)
	strm = mt5pb.NewStreamsClient(cc)
	qh = mt5pb.NewQuoteHistoryClient(cc)
	tkh = mt5pb.NewTickHistoryClient(cc)

	// ── 登录（ConnectEx 优先，Connect 备）+ id header 验证 ──
	sid := fmt.Sprintf("verify-%d", time.Now().UnixNano())
	lctx, lc := context.WithTimeout(metadata.NewOutgoingContext(ctx,
		metadata.New(map[string]string{"id": sid})), 30*time.Second)
	if *server != "" {
		r, err := conn.ConnectEx(lctx, &mt5pb.ConnectExRequest{User: *login, Password: *pass, Server: *server})
		if err == nil && r.Error == nil {
			token = r.Result
			fmt.Println("ConnectEx ok")
		} else {
			fmt.Println("ConnectEx:", err, r.GetError())
		}
	}
	if token == "" && *host != "" {
		r, err := conn.Connect(lctx, &mt5pb.ConnectRequest{User: *login, Password: *pass, Host: *host, Port: 443})
		if err != nil || r.Error != nil {
			fmt.Fprintln(os.Stderr, "Connect:", err, r.GetError())
			os.Exit(1)
		}
		token = r.Result
		fmt.Println("Connect ok")
	}
	lc()
	if token == "" {
		fmt.Fprintln(os.Stderr, "no token")
		os.Exit(1)
	}
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": token}))

	// ── id header 必带验证（对照组：剥 header 调 AccountSummary）──
	{
		naked, nc := context.WithTimeout(context.Background(), 10*time.Second)
		nr, err := mt5c.AccountSummary(naked, &mt5pb.AccountSummaryRequest{Id: token})
		nc()
		st, det := status(err, nr.GetError(), nr.GetResult() == nil)
		rec(claim{"gateway-admin", "id header 必带（对照剥header应拒）", "apierr"}, st, det)
	}

	initClaimRunner(ctx)

	// ── 读面/流面谱（拆件 reads/quotes/streams.go——行数棘轮） ──
	verifyAdminAccountOrders(ctx, *host)
	verifyQuotesSubs(ctx)
	verifyStreams(ctx, *streamWin)
	// ══ 判定输出 ══
	fmt.Printf("\n%-10s %-42s %-28s %-30s %s\n", "SKILL", "CLAIM", "EXPECT", "ACTUAL", "VERDICT")
	fmt.Println(strings.Repeat("-", 128))
	sort.SliceStable(results, func(i, j int) bool {
		return results[i].claim.skill+results[i].claim.name < results[j].claim.skill+results[j].claim.name
	})
	npass, ndiff, nskip := 0, 0, 0
	for _, r := range results {
		v := "PASS"
		exp := r.expect
		act := r.actual
		match := act == exp ||
			(exp == "ok" && act == "ok") ||
			(strings.HasPrefix(exp, "apierr") && strings.HasPrefix(act, "apierr")) ||
			(exp == "err" && (strings.HasPrefix(act, "apierr") || strings.HasPrefix(act, "rpcerr"))) ||
			(exp == "any") ||
			(exp == "expect-empty-or-err" && (act == "empty" || strings.HasPrefix(act, "apierr") || strings.HasPrefix(act, "rpcerr"))) ||
			(exp == "rpcerr" && strings.HasPrefix(act, "rpcerr"))
		if !match {
			v = "DIFF"
			ndiff++
		} else {
			npass++
		}
		fmt.Printf("%-10s %-42s %-28s %-30s %s  %s\n", r.skill, r.name, exp, act, v, r.detail)
	}
	fmt.Println(strings.Repeat("-", 128))
	fmt.Printf("unary/subs: PASS=%d DIFF=%d SKIP=%d\n", npass, ndiff, nskip)

	fmt.Println("\n流面:")
	for name, r := range sresM {
		end := "窗毕"
		if r.end != nil {
			end = short(r.end.Error())
		}
		fmt.Printf("%-24s frames=%-4d first=%.80s end=%s\n", name, r.n, r.first, end)
	}

	if *demoEnv != "" {
		runMutating(ctx)
	}
	if *realMut {
		runMutatingReal(ctx, *mutSym)
	}
	dctx, dc := context.WithTimeout(ctx, 10*time.Second)
	_, _ = conn.Disconnect(dctx, &mt5pb.DisconnectRequest{Id: token})
	dc()
}

func errStr(e error) string {
	if e == nil {
		return ""
	}
	return e.Error()
}

func iptr(i int32) *int32   { return &i }
func iptr64(i int64) *int64 { return &i }
