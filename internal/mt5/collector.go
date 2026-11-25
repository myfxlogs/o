// collector.go —— 员工「汇市巡」（编制册数据域新增卡）：MT5 只读采集岗。
// 职责：roster 驱动每户一条 session（O-37448——proto='mt5' 在册户各持
// ConnectEx owner 权一户会话死只退避本户，监督环在 roster.go）；boot 首读
// 账户快照（venue=<户token>/USD 本位）+符号清单实读落采集面（symbols.go
// 三源序）；quote 走 Subscribe+OnQuote 流（D-004 流正道——mt5 无 poll 报价
// 语义）→md_ticks venue=户 token；账户权益走 OnOrderProfit 推送流→
// md_account_events（持仓期 ~1Hz ProfitUpdate，push-on-change 值变才落行；
// 空仓静默=无变动非流死——venue 事件流双轨同 okxpriv）。断线=退避重连
// （重连=重订阅+账户快照愈合读——事件锚定形）；禁 mutating 面
// （mt5-trading-mutating 卡——client_test.go grep 锚）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"o/internal/platform/netmeter"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	mt5pb "o/internal/gen/mtapi/mt5"
	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/rejects"
)

const (
	employeeName  = "汇市巡"
	employeeDept  = "data"
	employeeTitle = "MT5 只读采集岗——quote 流+账户快照（零钱径）"
	beatEvery     = 30 * time.Second
	reconMin      = 2 * time.Second
	reconMax      = 60 * time.Second
)

// acctRun 单户会话运行件（O-37448——凭据/共享会话锚/充提水位全按户隔离，
// 一户会话死只退避本户天然免疫连坐）。
type acctRun struct {
	acct     VenueAcct
	cr       Credentials
	sessAddr string
	sessTok  string
	// balWM 充提轮询水位（unixnano——上拍 OrderHistory 上界；0=未跑过
	// 首拍走 7d 回看补漏窗）。跨会话存活——重连不回看重跑（O-35753）。
	balWM  atomic.Int64
	cancel context.CancelFunc
}

// Collector 汇市巡在岗体——roster 监督器（每户一 acctRun 会话 goroutine）。
type Collector struct {
	meter *netmeter.Meter
	pool  *pgxpool.Pool
	cfg   *config.Client
	rej   *rejects.Door
	// dial 覆写面（测试注入 fake server；nil=grpc.NewClient 生产径）。
	dial func(addr string) (*Client, error)
	mu   sync.Mutex
	runs map[string]*acctRun // venue→户运行件
}

// readCreds 按户凭据+网关址读（cred_ns=roster 行凭据键前缀——
// <cred_ns>.{login,password,server,host}；host 行缺=空串合法 server 径户
// ——XM 无 host 行实证，勿沿用「行缺=err」语义对 host 以外键）。
// 凭据族缺/坏=该户 err 显形不连坐他户（会话环按户隔离）。
func readCreds(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client, credNS string) (string, Credentials, error) {
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
	login, err := get(credNS + ".login")
	if err != nil {
		return "", Credentials{}, err
	}
	var loginN uint64
	if _, err := fmt.Sscanf(login, "%d", &loginN); err != nil {
		return "", Credentials{}, fmt.Errorf("%s.login 非数字（demo 户字样如实注）: %w", credNS, err)
	}
	pass, err := get(credNS + ".password")
	if err != nil {
		return "", Credentials{}, err
	}
	server, err := get(credNS + ".server")
	if err != nil {
		return "", Credentials{}, err
	}
	host, _ := get(credNS + ".host") // 行缺/解密败=空串——server 径户合法（XM 无 host 行实证）
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

// NewCollector 装配（roster 驱动多会话——无单户凭据 fail-closed：roster 可读
// ∨存量 mt5 凭据在，两者皆缺=采集面真空 WARN 缺席同旧径）。
func NewCollector(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Collector, error) {
	if _, err := VenueAccounts(ctx, pool, "cfd"); err != nil {
		if _, _, cerr := readCreds(ctx, pool, cfg, "mt5"); cerr != nil {
			return nil, fmt.Errorf("roster 读败且 mt5 凭据缺: %w / %v", err, cerr)
		}
	}
	return &Collector{pool: pool, cfg: cfg, runs: map[string]*acctRun{},
		rej: rejects.New("mt5", "connect_failed", "subscribe_failed", "stream_dead",
			"symbols_read_failed", "snapshot_write_failed", "tick_write_err", "write_err",
			"balance_ops_err")}, nil
}

// AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) AttachMeter(m *netmeter.Meter) { c.meter = m }

// profitSnap OnOrderProfit 帧去重键——venue ~1Hz 周期推，事件语义只记值变。
type profitSnap struct {
	eq, fm, bal, margin, profit, ml float64
	open                            int
}

// profitLoop OnOrderProfit 帧消费：值变帧落 md_account_events（channel='order_profit'
// append-only，INSERT 自带 ev_account NOTIFY）；同值帧/内嵌 Error 帧不落。
// venue=户 token（O-37448——各户权益事件行分键不互踩）。
// 返回 err=流死——调用侧取消会话走重连径。
func (c *Collector) profitLoop(ctx context.Context, st grpc.ServerStreamingClient[mt5pb.OnOrderProfitReply], a *acctRun) error {
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
			VALUES(now(),$1,'order_profit','USD',$2)
			ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
			data=EXCLUDED.data,recv_ts=now()`, a.acct.Venue, blob); err != nil {
			c.rej.Inc("snapshot_write_failed")
		}
	}
}

// snapshotOnce 账户权益事件行（AccountSummary 实读→md_account_events append-only
// 事实行；写败=计数不炸会话。T-0048：快照表废弃——事件行是同源真源）。
// venue=户 token（O-37448）。
func (c *Collector) snapshotOnce(ctx context.Context, cl *Client, src string, a *acctRun) {
	eq, fm, _, err := cl.AccountSummary(ctx)
	if err != nil {
		c.rej.Inc("symbols_read_failed")
		log.Printf("[mt5] %s AccountSummary 读败（事件行缺席）: %v", a.acct.Venue, err)
		return
	}
	blob, _ := json.Marshal(map[string]any{"eq": eq, "avail": fm, "src": src, "login": a.cr.Login})
	if _, err := c.pool.Exec(ctx, `INSERT INTO md_account_events(ts,venue,channel,ccy,data)
		VALUES(now(),$1,$2,'USD',$3)
		ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
		data=EXCLUDED.data,recv_ts=now()`, a.acct.Venue, src, blob); err != nil {
		c.rej.Inc("snapshot_write_failed")
	}
}
