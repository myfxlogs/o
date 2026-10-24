// collector.go —— 员工「汇市巡」（编制册数据域新增卡）：MT5 只读采集岗。
// 职责：boot 首读账户快照（venue='mt5'/USD 本位）+符号清单实读落采集面；
// quote 走 Subscribe+OnQuote 流（D-004 流正道——mt5 无 poll 报价语义）→md_ticks。
// 断线=退避重连（重连=重订阅+账户快照愈合读——事件锚定形）；禁 mutating 面
// （mt5-trading-mutating 卡——client_test.go grep 锚）。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/platform/rejects"
	"phase/internal/platform/roster"
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
	pool *pgxpool.Pool
	cfg  *config.Client
	rej  *rejects.Door
	cr   Credentials
	addr string
	// dial 覆写面（测试注入 fake server；nil=grpc.NewClient 生产径）。
	dial func(addr string) (*Client, error)
}

// OpenSession 独立一次性会话（开连+登录——用毕调用方 Disconnect；facts 采面/
// 探针面用，凭据/网关址与 Collector 同源）。
func OpenSession(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Client, error) {
	addr, cr, err := readCreds(ctx, pool, cfg)
	if err != nil {
		return nil, err
	}
	cl, err := Dial(addr)
	if err != nil {
		return nil, err
	}
	if err := cl.Connect(ctx, cr); err != nil {
		cl.Close()
		return nil, err
	}
	return cl, nil
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
	addr, cr, err := readCreds(ctx, pool, cfg)
	if err != nil {
		return nil, err
	}
	return &Collector{pool: pool, cfg: cfg, addr: addr, cr: cr,
		rej: rejects.New("mt5", "connect_failed", "subscribe_failed", "stream_dead",
			"symbols_read_failed", "snapshot_write_failed", "tick_write_err", "write_err")}, nil
}

// Run 上岗：Register+Beat → 连接循环（Connect→boot 读→订阅→OnQuote 流消费；
// 流断=退避重连同重订阅——订阅镜像在 connectLoop 内重建）。
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
		if err := c.session(ctx, dial); err != nil && ctx.Err() == nil {
			c.rej.Inc("stream_dead")
			log.Printf("[mt5] 会话终止（%.0fs 后重连）: %v", backoff.Seconds(), err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, reconMax)
	}
}

// session 一次会话生命周期：Connect→boot（账户快照+符号面+订阅）→OnQuote 流消费。
// 返回=流断/写败（外层退避重连）；成功消费一帧由调用侧无法感知——流常驻直至断。
func (c *Collector) session(ctx context.Context, dial func(string) (*Client, error)) error {
	cl, err := dial(c.addr)
	if err != nil {
		c.rej.Inc("connect_failed")
		return err
	}
	defer cl.Close()
	if err := cl.Connect(ctx, c.cr); err != nil {
		c.rej.Inc("connect_failed")
		return err
	}
	defer cl.Disconnect(context.Background())
	// boot 账户快照（事件锚定首读——USD 本位；venue_type 闭集 'mt5' 已在 0005）。
	c.snapshotOnce(ctx, cl, "mt5_boot")
	syms := c.symbols(ctx, cl)
	for _, s := range syms {
		if err := cl.Subscribe(ctx, s); err != nil {
			c.rej.Inc("subscribe_failed")
			log.Printf("[mt5] 订阅败 %s: %v", s, err)
		}
	}
	if len(syms) == 0 {
		return fmt.Errorf("采集清单空（mt5.symbols 空且 SymbolList 实读筛选零命中）")
	}
	stream, err := cl.OnQuote(ctx)
	if err != nil {
		return err
	}
	for {
		f, err := stream.Recv()
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
			`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask) VALUES(now(),'mt5',$1,$2,$3,$4)`,
			q.Symbol, q.Last, q.Bid, q.Ask); err != nil {
			c.rej.Inc("tick_write_err")
		}
	}
}

// snapshotOnce 账户快照行（equity/avail——AccountSummary 实读；写败=计数不炸会话）。
func (c *Collector) snapshotOnce(ctx context.Context, cl *Client, src string) {
	eq, fm, _, err := cl.AccountSummary(ctx)
	if err != nil {
		c.rej.Inc("symbols_read_failed")
		log.Printf("[mt5] AccountSummary 读败（快照缺席）: %v", err)
		return
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO fin_venue_account_snapshot(venue,venue_type,ccy,equity,avail,src)
		VALUES('mt5','mt5','USD',$1,$2,$3)
		ON CONFLICT(venue,venue_type,ccy) DO UPDATE SET equity=EXCLUDED.equity,avail=EXCLUDED.avail,src=EXCLUDED.src,updated_at=now()`,
		eq, fm, src); err != nil {
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
