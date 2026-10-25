package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/netmeter"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

// 端点/节流钉（skill okx-market-ws：WS=h1 Upgrade 非 h2；ping 帧=文本 "ping"）。
const (
	PublicURL  = "wss://ws.okx.com:8443/ws/v5/public"
	pingEvery  = 20 * time.Second
	dataIdle   = 75 * time.Second // 业务帧看门狗（ack≠存活——帧时戳前移才证活）
	backoffMin = 2 * time.Second
	backoffMax = 60 * time.Second
	symbolsKey = "okx.symbols"

	employeeName  = "行情巡"
	employeeDept  = "data"
	employeeTitle = "OKX 公开行情采集岗"
)

// Collector OKX 公开 WS 采集器（连接+订阅+读循环+断线重连）。
type Collector struct {
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	url   string // 测试覆写指向 fake server
	queue chan rowItem

	frames    atomic.Int64 // 业务帧计数（观测面）
	lastFrame atomic.Int64 // 最近业务帧 UnixNano（status 端点/看门狗同读）
	lastSeq   map[string]int64
	Stats     *Door // 别名 rej（对外观测名）
	meter     *netmeter.Meter
}

// AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) AttachMeter(m *netmeter.Meter) { c.meter = m }

type Door = rejects.Door

// NewCollector 默认参数版（生产径）。
func NewCollector(pool *pgxpool.Pool, cfg *config.Client) *Collector {
	return &Collector{
		pool: pool, cfg: cfg, url: PublicURL,
		rej:     rejects.New("okxws", "non_usdt_instid", "seq_gap", "parse_fail", "write_err", "symbols_missing"),
		queue:   make(chan rowItem, 1024),
		lastSeq: map[string]int64{},
		Stats:   nil,
	}
}

// Run 采集主循环：Register「行情巡」→Beat→写协程→拨号循环（重连=全量重订阅；
// symbols 代次变化→断流重连换新订阅面=热启生效）。阻塞至 ctx 取消。
func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	c.Stats = c.rej
	if err := roster.Register(ctx, c.pool, employeeName, employeeDept, employeeTitle, nil); err != nil {
		if ctx.Err() == nil {
			errc <- fmt.Errorf("行情巡 register: %w", err)
		}
		return
	}
	go roster.BeatLoop(ctx, c.pool, employeeName, 30*time.Second, errc)
	go c.writer(ctx)

	backoff := backoffMin
	for ctx.Err() == nil {
		syms, err := c.symbols(ctx)
		if err != nil {
			c.rej.Inc("symbols_missing")
			log.Printf("[okxws] %s 读败/缺席：%v——%ss 后重试（fail-closed 不虚构标的集）", symbolsKey, err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, backoffMax)
			continue
		}
		gen := c.cfg.Generation()
		err = c.dialLoop(ctx, syms, gen)
		if ctx.Err() != nil {
			return
		}
		log.Printf("[okxws] 连接终：%v——%s 后重连", err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, backoffMax)
	}
}

// symbols 从 config 快照读标的集（JSON 数组；逐键 fail-closed——非 USDT 键域拒入计数）。
func (c *Collector) symbols(ctx context.Context) ([]string, error) {
	raw, err := c.cfg.Get(symbolsKey)
	if err != nil {
		return nil, err
	}
	var all []string
	if err := json.Unmarshal(raw, &all); err != nil {
		return nil, fmt.Errorf("%s JSON 解析: %w", symbolsKey, err)
	}
	var out []string
	for _, s := range all {
		if !isUSDTInst(s) {
			c.rej.Inc("non_usdt_instid")
			log.Printf("[okxws] 非 USDT 计价标的拒订: %s（业主令杂币不采）", s)
			continue
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, errors.New("标的集过滤后为空")
	}
	return out, nil
}

// subscribeArgs 五频道订阅面（funding-rate/mark-price 仅 SWAP instId 合法；
// index-tickers 挂 spot 形 instId；books5/tickers 两形通用）。
func subscribeArgs(syms []string) []map[string]string {
	args := make([]map[string]string, 0, len(syms)*4)
	for _, s := range syms {
		args = append(args, map[string]string{"channel": "tickers", "instId": s},
			map[string]string{"channel": "books5", "instId": s})
		if strings.HasSuffix(s, "-SWAP") {
			args = append(args,
				map[string]string{"channel": "funding-rate", "instId": s},
				map[string]string{"channel": "mark-price", "instId": s})
		} else {
			args = append(args, map[string]string{"channel": "index-tickers", "instId": s})
		}
	}
	return args
}

// dialLoop 单连接生命周期：dial→订阅→读循环+ping+看门狗+config 代次探针。
// 返回终因（错误/代次变）——外层退避重连。
func (c *Collector) dialLoop(ctx context.Context, syms []string, gen int64) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, http.Header{})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	sub := map[string]any{"op": "subscribe", "args": subscribeArgs(syms)}
	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	log.Printf("[okxws] subscribed syms=%d args=%d url=%s", len(syms), len(sub["args"].([]map[string]string)), c.url)
	// 看门狗起表（T-0050 根修）：lastFrame 初值 0=「未起表」盲区——队列塞死时
	// 业务帧永不上前移，watchdog 恒 skip（SG 实勘静默断流）。订阅成功即起表：
	// 75s 无帧判死重连，盲区消失。
	c.lastFrame.Store(time.Now().UnixNano())

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			if mt == websocket.TextMessage && string(data) == "pong" {
				continue
			}
			c.handle(data)
		}
	}()
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	watch := time.NewTicker(10 * time.Second)
	defer watch.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return fmt.Errorf("read: %w", err)
		case <-ping.C:
			if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				return fmt.Errorf("ping: %w", err)
			}
		case <-watch.C:
			if idle := time.Since(time.Unix(0, c.lastFrame.Load())); c.lastFrame.Load() != 0 && idle > dataIdle {
				return fmt.Errorf("业务帧静默 %s > %s（ack≠存活——看门狗判死重连）", idle.Round(time.Second), dataIdle)
			}
			if c.cfg.Generation() != gen {
				return fmt.Errorf("config 代次 %d→%d（%s 热启——重连换订阅面）", gen, c.cfg.Generation(), symbolsKey)
			}
		}
	}
}

// handle 帧分流：按 arg.channel 解析→行入异步队列（解析失败/非 USDT 键域计 rejects）。
func (c *Collector) handle(data []byte) {
	var msg wsMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		c.rej.Inc("parse_fail")
		return
	}
	if msg.Event != "" {
		if msg.Event == "error" {
			log.Printf("[okxws] 服务端 error 帧: %s", data)
		}
		return // subscribe ack/channel-conn-count/error 留痕非业务帧
	}
	instID := msg.Arg.InstID
	if instID != "" && !isUSDTInst(instID) {
		c.rej.Inc("non_usdt_instid")
		return
	}
	for _, d := range msg.Data {
		var it rowItem
		switch msg.Arg.Channel {
		case "tickers":
			it.tick = parseTickers(instID, d)
		case "books5":
			it.books = parseBooks5(instID, d)
			if it.books != nil {
				c.checkSeq(it.books)
			}
		case "funding-rate":
			it.funding = parseFunding(instID, d)
		case "mark-price", "index-tickers":
			it.mark = parseMark(instID, msg.Arg.Channel, d)
		default:
			continue
		}
		if it.tick == nil && it.books == nil && it.funding == nil && it.mark == nil {
			c.rej.Inc("parse_fail")
			continue
		}
		c.frames.Add(1)
		c.lastFrame.Store(time.Now().UnixNano())
		select {
		case c.queue <- it:
		default:
			c.rej.Inc("write_err")
		}
	}
}

// checkSeq books5 seqId 单调性判据（快照频道 seqId 递增；倒退=序列缺——
// 计数+warn 不丢行，事实行原样入库带 recv_ts）。
func (c *Collector) checkSeq(b *booksRow) {
	if !b.hasSeq {
		return
	}
	if last, ok := c.lastSeq[b.instID]; ok && b.seqID <= last {
		c.rej.Inc("seq_gap")
		log.Printf("[okxws] books5 seq 倒退 %s: %d<=%d（序列完整性违例）", b.instID, b.seqID, last)
	}
	c.lastSeq[b.instID] = b.seqID
}

// FrameCount 业务帧累计（status 观测面）。
func (c *Collector) FrameCount() int64 { return c.frames.Load() }

// LastFrame 最近业务帧时刻（status/看门狗观测面）。
func (c *Collector) LastFrame() time.Time { return time.Unix(0, c.lastFrame.Load()) }
