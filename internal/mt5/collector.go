// collector.go —— 员工「汇市巡」（编制册数据域新增卡）：MT5 只读采集岗。
// 职责：boot 首读账户快照（venue='mt5'/USD 本位）+符号清单实读落采集面；
// quote 走 Subscribe+OnQuote 流（D-004 流正道——mt5 无 poll 报价语义）→md_ticks；
// 账户权益走 OnOrderProfit 推送流→md_account_events（持仓期 ~1Hz ProfitUpdate，
// push-on-change 值变才落行；空仓静默=无变动非流死——venue 事件流双轨同 okxpriv）。
// 断线=退避重连（重连=重订阅+账户快照愈合读——事件锚定形）；禁 mutating 面
// （mt5-trading-mutating 卡——client_test.go grep 锚）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"o/internal/platform/netmeter"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/rejects"
	"o/internal/platform/roster"
)

const (
	employeeName  = "汇市巡"
	employeeDept  = "data"
	employeeTitle = "MT5 只读采集岗——quote 流+账户快照（零钱径）"
	beatEvery     = 30 * time.Second
	reconMin      = 2 * time.Second
	reconMax      = 60 * time.Second
	// 候选方言谱（技能卡 Exness=m 尾实拍+裸名双形——经 venue SymbolConfirm 回确后启用）。
	symbolCandidates = "BTCUSDm,ETHUSDm,BTCUSD,ETHUSD"
)

// Collector 汇市巡在岗体。
type Collector struct {
	meter *netmeter.Meter
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	cr    Credentials
	// dial 覆写面（测试注入 fake server；nil=grpc.NewClient 生产径）。
	dial func(addr string) (*Client, error)
	// 会话锚——上会话所在网关+token（token 按网关域有效）。断线重连
	// 先 CheckAlive 复挂：流死≠会话死，活则零新 login 避开券商冲突掐。
	sessAddr string
	sessTok  string
	// balWM 充提轮询水位（unixnano——上拍 OrderHistory 上界；0=未跑过
	// 首拍走 7d 回看补漏窗）。跨会话存活——重连不回看重跑（O-35753）。
	balWM atomic.Int64
}

// readCreds 凭据+网关址读（NewCollector/OpenSession 共源）。
func readCreds(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (string, Credentials, error) {
	mk, err := config.ReadMasterKey()
	if err != nil {
		return "", Credentials{}, fmt.Errorf("master key: %w", err)
	}
	get := func(k string) (string, error) {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			return "", fmt.Errorf("%s 读败: %w", k, err)
		}
		return cryptoutil.Open(mk, ct)
	}
	login, err := get("mt5.login")
	if err != nil {
		return "", Credentials{}, err
	}
	var loginN uint64
	if _, err := fmt.Sscanf(login, "%d", &loginN); err != nil {
		return "", Credentials{}, fmt.Errorf("mt5.login 非数字（demo 户字样如实注）: %w", err)
	}
	pass, err := get("mt5.password")
	if err != nil {
		return "", Credentials{}, err
	}
	server, err := get("mt5.server")
	if err != nil {
		return "", Credentials{}, err
	}
	host, err := get("mt5.host")
	if err != nil {
		return "", Credentials{}, err // 密文行在但值可空（server 径户 host 空串合法）
	}
	raw, err := cfg.Get("mt5.gateway_addr")
	if err != nil {
		return "", Credentials{}, fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	var addr string
	if err := json.Unmarshal(raw, &addr); err != nil {
		return "", Credentials{}, err
	}
	return addr, Credentials{Login: loginN, Password: pass, Server: server, Host: host}, nil
}

// NewCollector 凭据自 app_secrets 密文列解+网关址热启（缺件 fail-closed 上抛——
// 装配层 WARN 缺席不炸 boot，与 okxpriv 同构）。
func NewCollector(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Collector, error) {
	_, cr, err := readCreds(ctx, pool, cfg)
	if err != nil {
		return nil, err
	}
	return &Collector{pool: pool, cfg: cfg, cr: cr,
		rej: rejects.New("mt5", "connect_failed", "subscribe_failed", "stream_dead",
			"symbols_read_failed", "snapshot_write_failed", "tick_write_err", "write_err",
			"balance_ops_err")}, nil
}

// Run 上岗：Register+Beat → 连接循环（Connect→boot 读→订阅→OnQuote 流消费；
// 流断=退避重连同重订阅——订阅镜像在 connectLoop 内重建）。
// AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) AttachMeter(m *netmeter.Meter) { c.meter = m }

func (c *Collector) Run(ctx context.Context, errc chan<- error) {
	if err := roster.Register(ctx, c.pool, employeeName, employeeDept, employeeTitle, nil); err != nil {
		errc <- fmt.Errorf("%s register: %w", employeeName, err)
		return
	}
	go roster.BeatLoop(ctx, c.pool, employeeName, beatEvery, errc)
	dial := c.dial
	if dial == nil {
		dial = Dial
	}
	backoff := reconMin
	for {
		wait := backoff
		if err := c.session(ctx, dial); err != nil && ctx.Err() == nil {
			c.rej.Inc("stream_dead")
			if d, ok := SessionBackoff(err); ok && d > wait {
				wait = d // 会话冲突指纹——照 mtapi 冷却睡，勿 2s 节奏硬撞
			}
			log.Printf("[mt5] 会话终止（%.0fs 后重连）: %v", wait.Seconds(), err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait): // sched-exempt-ok 断线退避重连（期界等待非轮询——含 mtapi 冷却自白量）
		}
		backoff = min(backoff*2, reconMax)
	}
}

// profitSnap OnOrderProfit 帧去重键——venue ~1Hz 周期推，事件语义只记值变。
type profitSnap struct {
	eq, fm, bal, margin, profit, ml float64
	open                            int
}

// profitLoop OnOrderProfit 帧消费：值变帧落 md_account_events（channel='order_profit'
// append-only，INSERT 自带 ev_account NOTIFY）；同值帧/内嵌 Error 帧不落。
// 返回 err=流死——调用侧取消会话走重连径。
func (c *Collector) profitLoop(ctx context.Context, st grpc.ServerStreamingClient[mt5pb.OnOrderProfitReply]) error {
	var last profitSnap
	for {
		f, err := st.Recv()
		if err != nil {
			return err
		}
		if f.Error != nil || f.Result == nil {
			continue
		}
		p := f.Result
		cur := profitSnap{p.Equity, p.FreeMargin, p.Balance, p.Margin, p.Profit, p.MarginLevel, len(p.Orders)}
		if cur == last {
			continue
		}
		last = cur
		blob, _ := json.Marshal(map[string]any{
			"eq": p.Equity, "avail": p.FreeMargin, "bal": p.Balance,
			"margin": p.Margin, "profit": p.Profit, "margin_level": p.MarginLevel,
			"open": len(p.Orders), "src": "on_order_profit"})
		if _, err := c.pool.Exec(ctx,
			`INSERT INTO md_account_events(ts,venue,channel,ccy,data)
			VALUES(now(),'mt5','order_profit','USD',$1)
			ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
			data=EXCLUDED.data,recv_ts=now()`, blob); err != nil {
			c.rej.Inc("snapshot_write_failed")
		}
	}
}

// snapshotOnce 账户权益事件行（AccountSummary 实读→md_account_events append-only
// 事实行；写败=计数不炸会话。T-0048：快照表废弃——事件行是同源真源）。
func (c *Collector) snapshotOnce(ctx context.Context, cl *Client, src string) {
	eq, fm, _, err := cl.AccountSummary(ctx)
	if err != nil {
		c.rej.Inc("symbols_read_failed")
		log.Printf("[mt5] AccountSummary 读败（事件行缺席）: %v", err)
		return
	}
	blob, _ := json.Marshal(map[string]any{"eq": eq, "avail": fm, "src": src, "login": c.cr.Login})
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_account_events(ts,venue,channel,ccy,data)
		VALUES(now(),'mt5',$1,'USD',$2)
		ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
		data=EXCLUDED.data,recv_ts=now()`, src, blob); err != nil {
		c.rej.Inc("snapshot_write_failed")
	}
}

// symbols 采集清单（mt5.symbols 热启优先；空=候选方言谱经 venue 回确——SymbolConfirm
// 确认制：回确子集才启用，未认候选弃）。
func (c *Collector) symbols(ctx context.Context, cl *Client) []string {
	raw, err := c.cfg.Get("mt5.symbols")
	if err == nil {
		var syms []string
		if json.Unmarshal(raw, &syms) == nil && len(syms) > 0 {
			return syms
		}
	}
	confirmed := cl.SymbolConfirm(ctx, strings.Split(symbolCandidates, ","))
	log.Printf("[mt5] 方言回确命中: %v", confirmed)
	return confirmed
}
