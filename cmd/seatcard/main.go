// seatcard —— 飞书席位活卡消费件（T-0458 业主令件）：与 metabus 同径
// Watch eventbus（localhost:15436 隧道）订 dm 帧 + 15s 兜底拍（领单/裁决/
// 完成类行不在 dm 帧白名单——≤30s 实证锚由兜底拍保）；每拍全席重渲→
// 内容哈希闸→live 时懒建/PATCH（exec.seatcard_live 按次读，缺省 false=
// dry-run 只渲不发——CTO 裁定⑥灰度序第一步）。独立进程：卡败落
// ops_event seatcard_err 行不吞账、不拖累 metabus 唤醒径。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/eventv1"
	"o/internal/gen/eventv1/eventv1connect"
	"o/internal/ops/seatcard"
	"o/internal/platform/config"
	"o/internal/platform/notify"
)

const (
	busAddr     = "http://localhost:15436" // metabus 同径（o-oci-bus-tunnel 隧道口）
	liveKey     = "exec.seatcard_live"     // 暗发闸（dmconfig 写径——缺省缺席=false）
	refreshTick = 15 * time.Second         // 兜底拍（≤30s 实证锚双倍余量）
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("FIN_PG_DSN")
	if dsn == "" {
		log.Fatal("FIN_PG_DSN 缺——席位活卡无账可读 fail-closed")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("fin 池连接败: %v", err)
	}
	defer pool.Close()

	// 卡册缺席（迁移未发窗）→live 闸强制 off（dry-run 照跑）——
	// seatcard 先于迁移上线的灰度序容错。
	var reg string
	if err := pool.QueryRow(ctx, "SELECT COALESCE(to_regclass('ops_seat_card')::text,'')").Scan(&reg); err != nil {
		log.Fatalf("卡册探测败: %v", err)
	}
	forceDry := reg == ""
	if forceDry {
		log.Printf("[seatcard] ops_seat_card 缺席（fin_0145 未发窗）——live 闸强制 off，dry-run 照跑")
	}

	cfg := config.New(pool)
	if err := cfg.Load(ctx); err != nil {
		log.Fatalf("config 快照 boot 载入败: %v", err)
	}
	gate := func() bool {
		if forceDry {
			return false
		}
		cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cfg.Load(cctx); err != nil {
			return false // 读败=暗（fail-closed 往安全侧——暗发闸默认 false 律）
		}
		b, err := cfg.GetBool(liveKey)
		if err != nil {
			return false // 键缺席=暗发默认 false（灰度律——dmconfig 翻 true 开真 PATCH）
		}
		return b
	}

	appID := os.Getenv("FEISHU_MANAGE_APP_ID")
	appSecret := os.Getenv("FEISHU_MANAGE_APP_SECRET")
	chatID := os.Getenv("FEISHU_MANAGE_CHAT_ID")
	fc := notify.NewFeishu(appID, appSecret)

	cards := seatcard.NewCards(seatcard.Config{
		Seats:  seatcard.Phase1Seats,
		Live:   gate,
		Create: fc.CreateInteractive,
		Patch:  fc.PatchCard,
		ChatID: chatID,
		Logf:   func(f string, a ...any) { log.Printf("[seatcard] "+f, a...) },
	}, &seatcard.PGStore{Pool: pool})
	cards.SeedHash(ctx)

	// 单刷新环——dm 帧与兜底拍同径串行（触发合流：pending 即并）。
	trig := make(chan struct{}, 1)
	go func() {
		// 兜底拍——领单/裁决/完成类行不在 eventbus dm 帧白名单（帧域=决策席
		// 唤醒面，扩帧=他席成本面归 eventbus 票域）；≤30s 实证锚由拍保。
		// 哈希闸+PATCH 下限闸下，拍成本=只读推导，无变更零出站。
		tk := time.NewTicker(refreshTick) // sched-exempt-ok 兜底拍（帧白名单外的落账面刷新——非轮询业务）
		defer tk.Stop()
		for {
			select {
			case <-tk.C:
				select {
				case trig <- struct{}{}:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	refreshDone := make(chan struct{})
	go func() {
		defer close(refreshDone)
		for {
			select {
			case <-trig:
				cards.Refresh(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	cards.Reconcile(ctx) // 启动对账（一次性幂等——dry-run 形态=日志示渲染）
	log.Printf("[seatcard] 起——bus=%s tick=%v live闸=%v 席=%d",
		busAddr, refreshTick, gate(), len(seatcard.Phase1Seats))
	watchBus(ctx, func() {
		select {
		case trig <- struct{}{}:
		default:
		}
	})
	<-refreshDone
}

// watchBus eventbus 订阅环（metabus 同构：resume 内存水位+退避 1s→30s 帽
// +45s 无帧无心跳=流死重连）。dm 帧→触发重渲（帧=唤醒信号非载荷——
// 状态真源=ops_event，重渲自读账）。
func watchBus(ctx context.Context, onDM func()) {
	cli := eventv1connect.NewEventBusClient(connect.NewClient(
		connecthttp.NewTransport(http.DefaultClient, busAddr)))
	resume := map[string]int64{}
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := watchOnce(ctx, cli, resume, onDM)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[seatcard] watch 断 %v——%v 后重连", err, backoff)
		time.Sleep(backoff) // sched-exempt-ok 重连退避（metabus 钦定同款）
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func watchOnce(ctx context.Context, cli eventv1connect.EventBusClient,
	resume map[string]int64, onDM func()) error {
	wctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wm []*eventv1.SourceWatermark
	for s, m := range resume {
		wm = append(wm, &eventv1.SourceWatermark{Source: s, MaxId: m})
	}
	st, err := cli.Watch(wctx, &eventv1.WatchRequest{Resume: wm})
	if err != nil {
		return err
	}
	defer st.Close()
	type res struct {
		fr  *eventv1.DataAdvanced
		err error
	}
	rch := make(chan res, 1)
	go func() { // Receive 串行喂通道——看门狗超时 cancel 解阻塞
		for {
			fr, err := st.Receive()
			rch <- res{fr, err}
			if err != nil {
				return
			}
		}
	}()
	idle := time.NewTimer(45 * time.Second) // metabus 同款流死自愈闸
	defer idle.Stop()
	for {
		select {
		case r := <-rch:
			if r.err != nil {
				return r.err
			}
			if !idle.Stop() {
				<-idle.C
			}
			idle.Reset(45 * time.Second)
			if r.fr.Source == "__hb__" { // 心跳哨兵——刷活不触发
				continue
			}
			resume[r.fr.Source] = r.fr.MaxId
			if r.fr.Kind == "dm" {
				onDM() // dm 帧→全席重渲（帧到即合流触发——刷新环单线程串行）
			}
		case <-idle.C:
			cancel()
			return context.DeadlineExceeded
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
