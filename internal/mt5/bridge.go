// bridge.go —— MT5 quote-only 桥腿（T-0141 P2b/D-0022）：跨所信号的 MT5
// 行情物理层。凭据契约（决策席钉死）：dev 桥零 app_secrets 读权——env 本地
// 注入 COLLECTO_MT5_{LOGIN,PASSWORD,SERVER,HOST,GATEWAY_ADDR}（ops 件已办，
// 读取契约仅此径禁 DB 径）。零账户面：不 snapshot 不 profit 流——collector_rw
// 零 fin_* 写权，账户帧撞权限墙（票面钦定 quote-only）；零 mutating（mt5
// trading 面零触）。quote 流→md_ticks venue='mt5'——与 SG 汇市巡同键共写
// （D-0018 共写语义：同键最新 ts 赢，桥写与 SG 本地采集零冲突）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
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
	pool     *pgxpool.Pool
	cfg      *config.Client
	meter    *netmeter.Meter
	rej      *rejects.Door
	addr     string
	cr       Credentials
	dial     func(addr string) (*Client, error)                                                  // 测试注入面
	symFetch func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) // 测试注入面（FIX-02 消哑失败形态 fixture 缝——nil=生产径）
}

// NewBridgeCollector env 凭据装配（缺件=fail-closed 上抛——桥腿缺席不拖 OKX 面，
// 装配层 WARN 放行余桥）。
func NewBridgeCollector(pool *pgxpool.Pool, cfg *config.Client) (*BridgeCollector, error) {
	env := func(k string) (string, error) {
		v := os.Getenv("COLLECTO_MT5_" + k)
		if v == "" {
			return "", fmt.Errorf("COLLECTO_MT5_%s 缺——桥腿凭据不齐", k)
		}
		return v, nil
	}
	loginS, err := env("LOGIN")
	if err != nil {
		return nil, err
	}
	login, err := strconv.ParseUint(loginS, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("COLLECTO_MT5_LOGIN 非数字: %w", err)
	}
	pass, err := env("PASSWORD")
	if err != nil {
		return nil, err
	}
	server, err := env("SERVER")
	if err != nil {
		return nil, err
	}
	host := os.Getenv("COLLECTO_MT5_HOST") // 可空（server 径户 host 空串合法）
	addr, err := env("GATEWAY_ADDR")
	if err != nil {
		return nil, err
	}
	return &BridgeCollector{pool: pool, cfg: cfg, addr: addr,
		cr: Credentials{Login: login, Password: pass, Server: server, Host: host},
		rej: rejects.New("mt5bridge", "connect_failed", "subscribe_failed",
			"symbols_missing", "tick_write_err", "symbols_read_failed", "symbol_group_nil")}, nil
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
	cl, err := dial(c.addr)
	if err != nil {
		return err
	}
	defer cl.Close()
	if err := cl.Connect(ctx, c.cr); err != nil {
		return err
	}
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

// fetchSwap 单 sym SymbolParams 读→裁定→换算→md_signal 落行。空返=落行成；
// 非空=败因样本。FIX-02 消哑分键：transport err/in-band err/Result nil→
// symbols_read_failed；SymbolGroup nil→symbol_group_nil（原 177 行静默
// continue 第二哑径——独立键留痕）；写败→tick_write_err（既有键）。
func (c *BridgeCollector) fetchSwap(ctx context.Context, cl *Client, sym string) string {
	fetch := c.symFetch
	if fetch == nil {
		fetch = func(ctx context.Context, cl *Client, sym string) (*mt5pb.SymbolParamsReply, error) {
			return cl.MT5().SymbolParams(cl.Authed(ctx), &mt5pb.SymbolParamsRequest{Id: cl.Token(), Symbol: sym})
		}
	}
	r, err := fetch(ctx, cl, sym)
	switch {
	case err != nil:
		c.rej.Inc("symbols_read_failed")
		return sym + ": " + err.Error()
	case r.GetError() != nil:
		c.rej.Inc("symbols_read_failed")
		return fmt.Sprintf("%s: mtapi err %d %q", sym, r.GetError().GetCode(), r.GetError().GetMessage())
	case r.Result == nil:
		c.rej.Inc("symbols_read_failed")
		return sym + ": Result nil"
	}
	sg := r.Result.SymbolGroup
	if sg == nil {
		c.rej.Inc("symbol_group_nil")
		return sym + ": SymbolGroup nil"
	}
	// 换算链全携 meta（票面钦定——点数→bps 链逐环可查）
	si := r.Result.SymbolInfo
	ann, ok := SwapAnnualBPS(si, sg, c.midOf(sym))
	longAnn, shortAnn, ok2 := SwapAnnualBoth(si, sg, c.midOf(sym)) // 返修①双腿年化
	meta, _ := json.Marshal(map[string]any{
		"swap_long": sg.GetSwapLong(), "swap_short": sg.GetSwapShort(),
		"swap_type": int32(sg.GetSwapType())})
	if si != nil {
		meta, _ = json.Marshal(map[string]any{
			"swap_long": sg.GetSwapLong(), "swap_short": sg.GetSwapShort(),
			"swap_type":  int32(sg.GetSwapType()),
			"tick_value": si.GetTickValue(), "tick_size": si.GetTickSize(),
			"points": si.GetPoints(), "contract_size": si.GetContractSize(),
			"max_volume": float64(sg.GetMaxVolume()), "initial_margin": sg.GetInitialMargin(),
			// T-0144 返修①：双腿年化随 meta（正 funding 径 swap_long 成本端入账源）。
			"swap_long_ann_bps": longAnn, "swap_short_ann_bps": shortAnn, "ann_ok": ok2,
			"mid": c.midOf(sym)})
	}
	var val any // NULL=单位方言不可年化（缺换算件缺席如实）
	if ok {
		val = ann // 空腿年化 bps（负 funding 对冲径正主）
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_signal(kind,key,val,meta,ts)
		VALUES('mt5_swap',$1,$2,$3::jsonb,now())
		ON CONFLICT (kind,key) DO UPDATE SET val=EXCLUDED.val,
		meta=EXCLUDED.meta, ts=EXCLUDED.ts`, sym, val, string(meta)); err != nil {
		c.rej.Inc("tick_write_err")
		return sym + ": write " + err.Error()
	}
	return ""
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
