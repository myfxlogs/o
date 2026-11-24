// Package okxpriv —— OKX 私有 WS（balance_and_position/account 频道）。
// 契约（okx-private-ws skill）：login 握手 sign=HMAC(secret,ts+"GET"+"/users/self/verify")；
// 事件→md_account_events 事实行+md_account_events append-only 行（src=okx_event，
// 键域归一 unified）；3min 心跳刷末帧写行——活性判据=快照行新鲜度非事件到达率；
// login/权限错误 fail-loud 重连不静默退化；凭据只自 app_secrets 密文列解。
// O-37450：roster 驱动多账户——Supervisor 按 fin_venue_roster proto='cex'
// 活跃户迭代 spawn 每户一 Collector（venue=数据写键一身三任，cred_ns=
// 凭据键前缀；同所第二户=零码，设计 SSOT venue-multi-account.md D5）。
package okxpriv

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/platform/config"
	"o/internal/platform/cryptoutil"
	"o/internal/platform/netmeter"
	"o/internal/platform/rejects"
	"o/internal/platform/signutil"
)

const (
	PrivateURL    = "wss://ws.okx.com:443/ws/v5/private"
	pingEvery     = 20 * time.Second
	dataIdle      = 5 * time.Minute // 私有面事件低频（静户零事件正常）——看门狗只判死连
	backoffMin    = 2 * time.Second
	backoffMax    = 60 * time.Second
	employeeName  = "私流守"
	employeeDept  = "data"
	employeeTitle = "OKX 私有流/账户快照岗"
)

// privateWSURL 会话拨号址（桩测覆写锚——同 rosterPollEvery 先例；生产恒
// PrivateURL）。
var privateWSURL = PrivateURL

type Collector struct {
	pool   *pgxpool.Pool
	url    string
	rej    *rejects.Door
	creds  signutil.Credentials
	venue  string // 名册户 token——md_account_events.venue 写键（空='okx' 存量兼容）
	sim    bool
	meter  *netmeter.Meter
	lastEq map[string]float64 // ccy→最近 equity（心跳刷源用末帧值）
	// lastAvail ccy→最近可用权益（availEq 口径——扫额闲余判据取值源；懒初始化）。
	lastAvail map[string]float64
	// bootFn boot 首读播种器（事件锚定形 REST——装配层注入 okxexec 件；
	// nil=纯流径不播种）。数据域不反向依赖交易域故注入非直 import。
	bootFn func(ctx context.Context) (map[string]float64, error)
	// billsHook bills 记账钩（T-0044 事件驱动：余额/持仓变动帧触发——
	// 装配层注入 funding 记账件；nil=不触发）。
	billsHook func(ctx context.Context)
	lastBills atomic.Int64
	frames    atomic.Int64
	lastFrm   atomic.Int64
}

// readCreds 按 cred_ns 解 CEX 三键凭据（<cred_ns>.{api_key,api_secret,
// passphrase}——okx cred_ns='okx' 存量三键兼容；okx_read/okx2.cex 等
// 名册户各键前缀自洽；缺件 fail-closed 上抛）。
func readCreds(ctx context.Context, pool *pgxpool.Pool, credNS string) (signutil.Credentials, error) {
	mk, err := config.ReadMasterKey()
	if err != nil {
		return signutil.Credentials{}, fmt.Errorf("master key: %w", err)
	}
	var creds signutil.Credentials
	for k, dst := range map[string]*string{
		credNS + ".api_key":    &creds.Key,
		credNS + ".api_secret": &creds.Secret,
		credNS + ".passphrase": &creds.Passphrase,
	} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			return signutil.Credentials{}, fmt.Errorf("%s 读败: %w", k, err)
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			return signutil.Credentials{}, fmt.Errorf("%s 解密: %w", k, err)
		}
		*dst = p
	}
	return creds, nil
}

// newAccountCollector 单户私有采集件（per-account 凭据+venue 写键——
// Supervisor.spawn 径；credNS=名册行 cred_ns）。
func newAccountCollector(ctx context.Context, pool *pgxpool.Pool, sim bool, venue, credNS string) (*Collector, error) {
	creds, err := readCreds(ctx, pool, credNS)
	if err != nil {
		return nil, err
	}
	return &Collector{pool: pool, url: privateWSURL, sim: sim, venue: venue, creds: creds,
		rej:    rejects.New("okxpriv", "parse_fail", "write_err", "creds_missing", "login_fail"),
		lastEq: map[string]float64{}}, nil
}

// newTestRej 测试用计数门（生产由 newAccountCollector 装）。
func newTestRej() *rejects.Door {
	return rejects.New("okxpriv", "parse_fail", "write_err", "creds_missing", "login_fail")
}

// SetBootFn 注入 boot 首读播种器（装配层 cmd/core 接 okxexec——cmd 不受方向闸）。
func (c *Collector) SetBootFn(fn func(ctx context.Context) (map[string]float64, error)) {
	c.bootFn = fn
}

// SetBillsHook 注入 bills 记账钩（cmd 装配层接线——数据域不反向依赖账本域）。
func (c *Collector) SetBillsHook(fn func(ctx context.Context)) { c.billsHook = fn }

// tryBills 事件触发记账拉取（30s 节流——事件洪峰不放大 REST 面；
// 异步+无取消 ctx——WS 读循环禁被 REST 拖死）。
func (c *Collector) tryBills(ctx context.Context) {
	if c.billsHook == nil {
		return
	}
	for {
		last := c.lastBills.Load()
		if time.Now().Unix()-last < 30 {
			return
		}
		if c.lastBills.CompareAndSwap(last, time.Now().Unix()) {
			go c.billsHook(context.WithoutCancel(ctx))
			return
		}
	}
}

// Credentials 装配层取凭据（bootFn 接线用——值不出 pkg 外非注入径）。
func (c *Collector) Credentials() signutil.Credentials { return c.creds }

// AttachMeter 挂带宽计量件（T-0057：nil=不计量）。
func (c *Collector) AttachMeter(m *netmeter.Meter) { c.meter = m }
