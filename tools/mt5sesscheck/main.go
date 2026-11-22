// mt5sesscheck —— 会话活性真值探针（OLD_VERSION 案教训件）：
// token 取得≠会话活——僵尸会话返陈账（AccountSummary 陈快照）而数据面已死。
// 判活链：ConnectEx(无体id)→AccountSummary(读)→GetQuote(msNotOlder 活鲜)→
// Subscribe+OnQuote(流活)——每段独立定谳，任一段死=该面尸。
// 退出码：0=全活 1=尸/拒 2=用法错。
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

	mt5pb "o/internal/gen/mtapi/mt5"
)

var dead bool

func stage(name string, ok bool, detail string) {
	mark := "OK  "
	if !ok {
		mark = "DEAD"
		dead = true
	}
	fmt.Printf("[%s] %-14s %s\n", mark, name, detail)
}

func main() {
	gw := flag.String("gw", "mt5grpc2.mtapi.io:443", "")
	login := flag.Uint64("login", 0, "")
	pass := flag.String("pass", "", "")
	server := flag.String("server", "", "")
	sym := flag.String("sym", "BTCUSDm", "活体验证品种（crypto 全年无休 tick）")
	streamWin := flag.Duration("stream", 10*time.Second, "OnQuote 流监听窗")
	flag.Parse()
	if *login == 0 || *pass == "" || *server == "" {
		fmt.Fprintln(os.Stderr, "need -login/-pass/-server")
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
	subs := mt5pb.NewSubscriptionsClient(cc)
	strm := mt5pb.NewStreamsClient(cc)

	// ① ConnectEx——体字段 Id 永禁填（自供 id 路由未更新成员面=OLD_VERSION 实证）
	sid := fmt.Sprintf("sesscheck-%d", time.Now().UnixNano())
	lctx, lcancel := context.WithTimeout(
		metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": sid})), 30*time.Second)
	r, err := connM.ConnectEx(lctx, &mt5pb.ConnectExRequest{User: *login, Password: *pass, Server: *server})
	lcancel()
	if err != nil {
		stage("ConnectEx", false, fmt.Sprintf("RPC err: %v", err))
		os.Exit(1)
	}
	if r.Error != nil {
		stage("ConnectEx", false, fmt.Sprintf("err: %s", r.Error.Message))
		os.Exit(1)
	}
	token := r.Result
	stage("ConnectEx", true, fmt.Sprintf("token=%s", token))
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": token}))

	// ② AccountSummary——读面通（注意：僵尸会话此步也通，勿单独作判据）
	sctx, scancel := context.WithTimeout(ctx, 15*time.Second)
	s, serr := mt5c.AccountSummary(sctx, &mt5pb.AccountSummaryRequest{Id: token})
	scancel()
	if serr != nil {
		stage("Summary", false, fmt.Sprintf("RPC err: %v", serr))
	} else if s.Error != nil {
		stage("Summary", false, fmt.Sprintf("err: %s", s.Error.Message))
	} else {
		stage("Summary", true, fmt.Sprintf("balance=%.2f equity=%.2f type=%s",
			s.Result.GetBalance(), s.Result.GetEquity(), s.Result.GetType()))
	}

	// ②b CheckConnect——情报段（vendor 内部重登面已知毒——活也报 OLD_VERSION；
	// 不计入判死，仅作 vendor 修复进度读数）
	kctx, kcancel := context.WithTimeout(ctx, 15*time.Second)
	k, kerr := connM.CheckConnect(kctx, &mt5pb.CheckConnectRequest{Id: token})
	kcancel()
	switch {
	case kerr != nil:
		fmt.Printf("[info] CheckConnect   RPC err: %v\n", kerr)
	case k.Error != nil:
		fmt.Printf("[info] CheckConnect   err: %s\n", k.Error.Message)
	default:
		fmt.Printf("[info] CheckConnect   result=%s（vendor 已修内部径）\n", k.GetResult())
	}

	// ③ GetQuote msNotOlder=3000——活鲜探（僵尸会话 socket disposed 即死于此）
	gctx, gcancel := context.WithTimeout(ctx, 20*time.Second)
	q, qerr := mt5c.GetQuote(gctx, &mt5pb.GetQuoteRequest{Id: token, Symbol: *sym, MsNotOlder: int32Ptr(3000)})
	gcancel()
	if qerr != nil {
		stage("GetQuote", false, fmt.Sprintf("RPC err: %v", qerr))
	} else if q.Error != nil {
		stage("GetQuote", false, fmt.Sprintf("err: %s", q.Error.Message))
	} else if q.Result != nil {
		stage("GetQuote", true, fmt.Sprintf("%s bid=%.2f ask=%.2f", q.Result.GetSymbol(), q.Result.GetBid(), q.Result.GetAsk()))
	} else {
		stage("GetQuote", false, "空体")
	}

	// ④ Subscribe——订阅径（内部重登面——OLD_VERSION 毒族成员，独立成段）
	subCtx, subCancel := context.WithTimeout(ctx, 10*time.Second)
	sub, subErr := subs.Subscribe(subCtx, &mt5pb.SubscribeRequest{Id: token, Symbol: *sym})
	subCancel()
	subOK := subErr == nil && sub.GetError() == nil
	switch {
	case subErr != nil:
		stage("Subscribe", false, fmt.Sprintf("RPC err: %v", subErr))
	case sub.Error != nil:
		stage("Subscribe", false, fmt.Sprintf("err: %s", sub.Error.Message))
	default:
		stage("Subscribe", true, fmt.Sprintf("result=%s", sub.GetResult()))
	}

	// ⑤ OnQuote——流面活（数帧判活；Subscribe 死则流必无帧）
	if !subOK {
		stage("OnQuote", false, "Subscribe 死——流无源")
	} else {
		stCtx, stCancel := context.WithTimeout(ctx, *streamWin+10*time.Second)
		st, stErr := strm.OnQuote(stCtx, &mt5pb.OnQuoteRequest{Id: token})
		if stErr != nil {
			stCancel()
			stage("OnQuote", false, fmt.Sprintf("stream err: %v", stErr))
		} else {
			deadline := time.Now().Add(*streamWin)
			frames := 0
			for time.Now().Before(deadline) {
				ev, err := st.Recv()
				if err != nil {
					break
				}
				if ev.GetResult() != nil {
					frames++
				}
			}
			stCancel()
			stage("OnQuote", frames > 0, fmt.Sprintf("%v 窗内 %d 帧", *streamWin, frames))
		}
	}

	// ⑤ Disconnect 收尾（探针会话勿留尸占会话位）
	dctx, dcancel := context.WithTimeout(ctx, 10*time.Second)
	dr, _ := connM.Disconnect(dctx, &mt5pb.DisconnectRequest{Id: token})
	dcancel()
	fmt.Printf("[----] Disconnect     %v\n", dr.GetResult())

	if dead {
		fmt.Println("\n判：尸——存在死面（对照：OLD_VERSION=版本闸面 / disposed socket=僵尸会话 / INVALID_SYMBOL=品种方言）")
		os.Exit(1)
	}
	fmt.Println("\n判：全活——会话可用，数据面真实")
}

func int32Ptr(v int32) *int32 { return &v }
