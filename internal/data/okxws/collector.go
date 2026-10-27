package okxws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/netmeter"
	"o/internal/platform/notify"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

// 端点/节流钉（skill okx-market-ws：WS=h1 Upgrade 非 h2；ping 帧=文本 "ping"）。
const (
	PublicURL  = "wss://ws.okx.com:443/ws/v5/public"
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
	Stats     *Door        // 别名 rej（对外观测名）
	meter     *netmeter.Meter
	al        *notify.Alerter
	refuseSub map[string]bool // 60018 拒收集（本连接内生效——T-0133 ③）
	FullSub   bool            // T-0137 桥面全量 WS 订阅（index-tickers/mark-price 同订——dev 蒸馏面）
	AlertKind string          // 面死告警 kind 覆写（桥=bridge_stale；缺省 okxws_face_dead）
	// mark-price REST 轮询读件已随 T-0171 快照价格全面退役全撤——mark/index
	// 行唯一来源=WS（桥全量订阅+SG 活跃集 tickers），REST 只准条款/目录注册面。
	// ExpandSymbols 宇宙扩件（T-0122：okx.symbols 含 "*"=放开全 SWAP+季度
	// 期货——装配层挂 instruments 实读闭包，未挂=键面拒签 fail-closed）。
	ExpandSymbols func(ctx context.Context) ([]string, error)

	// T-0139 桥二期信号面：books5 复活（D-0022——特征值非原始盘口；标的集
	// 可配置域 bridge.depth5.symbols，非全宇宙防桥自造洪泛）。
	depth5Syms []string                                              // 本连接订阅的深度标的集（config 热启，重连拍读）
	ctvFeed    func(ctx context.Context) (map[string]float64, error) // SWAP ctVal 读件（USD 额换算）
	ctvCache   map[string]float64
	ctvAt      time.Time
	depth5At   map[string]time.Time // per-inst 蒸馏写节流（特征值 5s 鲜度足够）
}

// AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) AttachMeter(m *netmeter.Meter) { c.meter = m }

// AttachAlerter 挂业主告警通道（T-0128 返修②——面死告警走 humanize kind
// 非 log 前缀；nil=测试/无装配 log 退路）。
func (c *Collector) AttachAlerter(a *notify.Alerter) { c.al = a }

type Door = rejects.Door

// NewCollector 默认参数版（生产径）。
func NewCollector(pool *pgxpool.Pool, cfg *config.Client) *Collector {
	return &Collector{
		pool: pool, cfg: cfg, url: PublicURL,
		rej: rejects.New("okxws", "non_usdt_instid", "seq_gap", "parse_fail", "write_err", "symbols_missing",
			"activeset_read_failed", "depth5_no_ctval"),
		queue: make(chan rowItem, 1024),
		Stats: nil,
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
	go c.staleWatch(ctx) // T-0129 面级断流告警（全订阅标的齐冻=面死显形）

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
		c.depth5Syms = c.depth5Symbols(ctx) // 桥面深度集热启（重连拍刷新）
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
		if s == "*" { // 宇宙放开：装配层实读面（SWAP 全谱+季度期货 tick）
			if c.ExpandSymbols == nil {
				return nil, errors.New("okx.symbols=* 但 ExpandSymbols 未挂——fail-closed")
			}
			exp, err := c.ExpandSymbols(ctx)
			if err != nil {
				return nil, fmt.Errorf("宇宙扩读败（fail-closed 不虚构标的集）: %w", err)
			}
			out = append(out, exp...)
			continue
		}
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

// dialLoop 单连接生命周期：dial→订阅→读循环+ping+看门狗+config 代次探针。
// 返回终因（错误/代次变）——外层退避重连。
func (c *Collector) dialLoop(ctx context.Context, syms []string, gen int64) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.url, http.Header{})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()
	// T-0122 返修②：OKX WS 单帧订阅 ≤100 args（全宇宙 ~500 arg 单帧被拍
	// 死——分帧多 op 续发，订阅节流件在 subscribeAll 内）。
	if err := subscribeAll(conn, c.subscribeSet(syms)); err != nil {
		return err
	}
	// T-0139 books5 复活订阅（D-0022 桥面信号域——标的集可配置非全宇宙）。
	if len(c.depth5Syms) > 0 {
		if c.ctvFeed != nil { // ctv 预载（读循环零同步外呼——帧径只查缓存）
			if m, err := c.ctvFeed(ctx); err == nil {
				c.ctvCache = m
				c.ctvAt = time.Now()
			} else {
				c.rej.Inc("depth5_no_ctval")
			}
		}
		args := make([]map[string]string, 0, len(c.depth5Syms))
		for _, s := range c.depth5Syms {
			args = append(args, map[string]string{"channel": "books5", "instId": s})
		}
		if err := subscribeAll(conn, args); err != nil {
			return fmt.Errorf("books5 subscribe: %w", err)
		}
		c.depth5At = map[string]time.Time{}
		log.Printf("[okxws] books5 桥面订阅 syms=%d（信号蒸馏域）", len(c.depth5Syms))
	}
	log.Printf("[okxws] subscribed syms=%d url=%s", len(syms), c.url)
	// T-0132 活跃集 tickers 动态重订（增删订阅帧不重建连接——fin_position
	// 在仓+fin_proposal 在途+fin_strategy 非退役的 inst_id 才配成交价级实时性）。
	// T-0133 幽灵标的滤除+分帧加订+60018 拒收集（件在 subscribe.go——棘轮）。
	subTick := map[string]bool{}
	c.refuseSub = map[string]bool{}
	syncActive := func() { c.syncActiveSet(ctx, conn, syms, subTick, c.refuseSub) }
	syncActive() // 连接建立即加订当前活跃集
	// 看门狗起表（T-0050 根修——订阅成功即起表消盲区：75s 无帧判死重连）。
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
	actTick := time.NewTicker(30 * time.Second) // sched-exempt-ok 活跃集轮询
	defer actTick.Stop()
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
		case <-actTick.C:
			syncActive() // 活跃集变化→增删订阅帧
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

// FrameCount 业务帧累计（status 观测面）。
func (c *Collector) FrameCount() int64 { return c.frames.Load() }

// LastFrame 最近业务帧时刻（status/看门狗观测面）。
func (c *Collector) LastFrame() time.Time { return time.Unix(0, c.lastFrame.Load()) }
