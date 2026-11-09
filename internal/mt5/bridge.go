// bridge.go —— MT5 quote-only 桥腿（T-0141 P2b/D-0022）：跨所信号的 MT5
// 行情物理层。凭据/址链归 app_config 会话重读（业主令 env 只留 PG 连接件——
// bridge.mt5.{login,password,server,host}+mt5.gateway_{addr,fallbacks}，
// dev 桥仍零 app_secrets 读权；原 env 注入径 T-0141 契约已退役）。
// 零账户面：不 snapshot 不 profit 流——collector_rw
// 零 fin_* 写权，账户帧撞权限墙（票面钦定 quote-only）；零 mutating（mt5
// trading 面零触）。quote 流→md_ticks venue='mt5'——与 SG 汇市巡同键共写
// （D-0018 共写语义：同键最新 ts 赢，桥写与 SG 本地采集零冲突）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/jackc/pgx/v5/pgxpool"

	mt5pb "o/internal/gen/mtapi/mt5"

	"o/internal/platform/config"
	"o/internal/platform/netmeter"
	"o/internal/platform/rejects"
)

// BridgeCollector MT5 桥腿在岗体（quote-only）。
type BridgeCollector struct {
	pool      *pgxpool.Pool
	cfg       *config.Client
	meter     *netmeter.Meter
	rej       *rejects.Door
	cr        Credentials
	dial      func(addr string) (*Client, error)                                                  // 测试注入面
	symFetch  func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) // 测试注入面（FIX-02 消哑失败形态 fixture 缝——nil=生产径）
	symList   func(ctx context.Context, cl *Client) ([]string, error)                             // 目录粗扫缝（T-0408——nil=SymbolList 正径）
	quoteMany func(ctx context.Context, cl *Client, syms []string) map[string]float64             // 目录 mid 快照缝（nil=GetQuoteMany 正径）
}

// NewBridgeCollector 装配（凭据/址链全归 app_config 会话重读——env 凭据径退役；
// 本件不再 fail-closed 于凭据缺，缺件=bridgeSession 返错退避重试同址链缺径）。
func NewBridgeCollector(pool *pgxpool.Pool, cfg *config.Client) (*BridgeCollector, error) {
	return &BridgeCollector{pool: pool, cfg: cfg,
		rej: rejects.New("mt5bridge", "connect_failed", "subscribe_failed",
			"symbols_missing", "tick_write_err", "symbols_read_failed", "symbol_group_nil")}, nil
}

// bridgeAuth 桥腿凭据读——app_config bridge.mt5.{login,password,server,host}
// （凭据归 PG 热更径，D-0005 env 只留 DSN 钦定收口；dev 桥仍零 app_secrets
// 读权——app_config≠密册）。host 可空缺省（server 径户 host 空串合法）。
func bridgeAuth(cfg *config.Client) (Credentials, error) {
	var cr Credentials
	login, err := cfg.GetInt("bridge.mt5.login")
	if err != nil {
		return cr, fmt.Errorf("bridge.mt5.login: %w", err)
	}
	cr.Login = uint64(login)
	if cr.Password, err = cfg.GetString("bridge.mt5.password"); err != nil {
		return cr, fmt.Errorf("bridge.mt5.password: %w", err)
	}
	if cr.Server, err = cfg.GetString("bridge.mt5.server"); err != nil {
		return cr, fmt.Errorf("bridge.mt5.server: %w", err)
	}
	cr.Host, _ = cfg.GetString("bridge.mt5.host")
	return cr, nil
}

// Run 上岗（无 roster 注册——桥面工序岗属采集器机延伸，编制册零新卡；
// 连接循环退避重连同汇市巡形）。
func (c *BridgeCollector) Run(ctx context.Context, errc chan<- error) {
	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	backoff := reconMin
	for ctx.Err() == nil {
		if err := c.bridgeSession(ctx, dial); err != nil && ctx.Err() == nil {
			c.rej.Inc("connect_failed")
			log.Printf("[mt5bridge] 会话终（%.0fs 后重连——OKX 面不连坐）: %v", backoff.Seconds(), err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff): // sched-exempt-ok 断线退避重连（期界等待非轮询拉取——汇市巡同款）
		}
		backoff = min(backoff*2, reconMax)
	}
}

// bridgeSession 一次会话：Connect→订阅（bridge.mt5.symbols 热启，缺=fail-closed
// 返错重试）→OnQuote 流→md_ticks venue='mt5'。零账户面（禁 snapshot/profit）。
func (c *BridgeCollector) bridgeSession(ctx context.Context, dial func(string) (*Client, error)) error {
	// T-0143 返修：会话域派生 ctx（Run 的 ctx 跨重连共用——直接传则断连不
	// cancel，swapLoop goroutine 随重连累积）。sessCtx 于会话终（stream 断返
	// 触发 defer）cancel——swapLoop 随会话止，零泄漏。
	sessCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	syms := c.symbols()
	if len(syms) == 0 {
		return fmt.Errorf("bridge.mt5.symbols 缺/空——fail-closed 不虚构标的集")
	}
	cr, err := bridgeAuth(c.cfg)
	if err != nil {
		return err
	}
	c.cr = cr
	addrs, err := gatewayAddrs(c.cfg)
	if err != nil {
		return err
	}
	var cl *Client
	var lastErr error
	for _, a := range addrs {
		cand, derr := dial(a)
		if derr != nil {
			lastErr = derr
			continue
		}
		if cerr := cand.Connect(ctx, c.cr); cerr != nil {
			cand.Close()
			lastErr = cerr
			continue
		}
		cl = cand
		break
	}
	if cl == nil {
		return lastErr
	}
	defer cl.Close()
	defer cl.Disconnect(context.Background())
	for _, s := range syms {
		if err := cl.Subscribe(ctx, s); err != nil {
			c.rej.Inc("subscribe_failed")
			log.Printf("[mt5bridge] 订阅败 %s: %v", s, err)
		}
	}
	log.Printf("[mt5bridge] quote-only 桥腿起 syms=%d（零账户面）", len(syms))
	stream, err := cl.OnQuote(sessCtx)
	if err != nil {
		return err
	}
	// swap 率面低频拍（≤1/4h——SymbolParams 域，日级变量；sched-exempt-ok 慢特征节拍）。
	go c.swapLoop(sessCtx, cl)
	// T-0408①：全目录粗扫拍（366 符 SymbolParams 步进限速+GetQuoteMany mid
	// 快照——低频 RPC 非流，影子评估层事实源；同会话止）。
	go c.catalogLoop(sessCtx, cl)
	for {
		f, err := stream.Recv()
		if c.meter != nil && f != nil {
			c.meter.Add("mt5bridge/quote", proto.Size(f))
		}
		if err != nil {
			return fmt.Errorf("OnQuote 流断: %w", err)
		}
		if f.Result == nil {
			continue
		}
		q := f.Result
		if q.Symbol == "" || (q.Bid == 0 && q.Ask == 0) {
			continue // 空帧（心跳形）不落表
		}
		if _, err := c.pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask) VALUES(now(),'mt5',$1,$2,$3,$4)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
			bid=EXCLUDED.bid,ask=EXCLUDED.ask,recv_ts=now()`,
			q.Symbol, q.Last, q.Bid, q.Ask); err != nil {
			c.rej.Inc("tick_write_err")
		}
	}
}

// AttachMeter 挂带宽计量件（T-0057 惯例——nil=不计量）。
func (c *BridgeCollector) AttachMeter(m *netmeter.Meter) { c.meter = m }

// swapLoop SymbolParams 率面拍：per symbol swap_long/short+SwapType 单位方言
// →md_signal kind='mt5_swap'。年化换算仅 SwapType=PercCurPrice（报价币百分比
// /天——×365×10⁴ bps）；其余单位（点数/币量）缺换算件=meta 记 raw，val=NULL
// （carry_diff 对该 sym 缺席如实——票面钦定）。会话断=拍随会话终（重连重启）。
// FIX-02 消哑：全灭拍（零落行）=可观测事件——WARN 落日志携首败样本
// （ops_event 在 PG-md 无表、collector 无写径——既有告警链取日志 WARN 分支）。
func (c *BridgeCollector) swapLoop(ctx context.Context, cl *Client) {
	syms := c.symbols()
	t := time.NewTicker(swapEvery) // sched-exempt-ok：日级变量低频拍
	defer t.Stop()
	poll := func() {
		okN, sample := c.pollSwap(ctx, cl, syms)
		if len(syms) > 0 && okN == 0 {
			log.Printf("WARN [mt5bridge] swap 拍全灭——%d sym 零落行（面上盲拍，样本: %s）",
				len(syms), sample)
		}
	}
	poll()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			poll()
		}
	}
}

// pollSwap 单拍逐 sym 读算写。返落行数+首败样本（诊断种子——全灭 WARN 携）。
// 全灭（零落行）=可观测事件（FIX-02 消哑）；单 sym 败仅分键计数（票面语义）。
func (c *BridgeCollector) pollSwap(ctx context.Context, cl *Client, syms []string) (int, string) {
	okN, sample := 0, ""
	for _, sym := range syms {
		if why := c.fetchSwap(ctx, cl, sym); why == "" {
			okN++
		} else if sample == "" {
			sample = why
		}
	}
	return okN, sample
}

// symbols 标的集（bridge.mt5.symbols 热启——首集 BTCUSDm/ETHUSDm 与 SG 钱径
// 清单同；扩集走 venue SymbolList 实读方言件另票）。
func (c *BridgeCollector) symbols() []string {
	raw, err := c.cfg.Get("bridge.mt5.symbols")
	if err != nil {
		return nil
	}
	var syms []string
	if json.Unmarshal(raw, &syms) != nil {
		return nil
	}
	return syms
}

// swap 节拍（SymbolParams 率面——≤1/4h 拍，日级变量）。
const swapEvery = 4 * time.Hour

// midOf sym 现价中点（md_ticks venue='mt5' 最新行——换算名义基准；缺=0 不可年化）。
func (c *BridgeCollector) midOf(sym string) float64 {
	var mid float64
	_ = c.pool.QueryRow(context.Background(), `SELECT (bid+ask)/2 FROM md_ticks
		WHERE venue='mt5' AND inst_id=$1 AND bid>0 AND ask>0`, sym).Scan(&mid)
	return mid
}
