// Package okxpriv —— OKX 私有 WS（balance_and_position/account 频道）。
// 契约（okx-private-ws skill）：login 握手 sign=HMAC(secret,ts+"GET"+"/users/self/verify")；
// 事件→md_account_events 事实行+fin_venue_account_snapshot 快照行（src=okx_event，
// 键域归一 unified）；3min 心跳刷末帧写行——活性判据=快照行新鲜度非事件到达率；
// login/权限错误 fail-loud 重连不静默退化；凭据只自 app_secrets 密文列解。
package okxpriv

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"phase/internal/platform/config"
	"phase/internal/platform/cryptoutil"
	"phase/internal/platform/rejects"
	"phase/internal/platform/signutil"
)

const (
	PrivateURL    = "wss://ws.okx.com:8443/ws/v5/private"
	pingEvery     = 20 * time.Second
	dataIdle      = 5 * time.Minute // 私有面事件低频（静户零事件正常）——看门狗只判死连
	heartbeat     = 3 * time.Minute // 快照行刷源拍
	backoffMin    = 2 * time.Second
	backoffMax    = 60 * time.Second
	employeeName  = "私流守"
	employeeDept  = "data"
	employeeTitle = "OKX 私有流/账户快照岗"
)

type Collector struct {
	pool   *pgxpool.Pool
	url    string
	rej    *rejects.Door
	creds  signutil.Credentials
	sim    bool
	lastEq map[string]float64 // ccy→最近 equity（心跳刷源用末帧值）
	// bootFn boot 首读播种器（事件锚定形 REST——装配层注入 okxexec 件；
	// nil=纯流径不播种）。数据域不反向依赖交易域故注入非直 import。
	bootFn  func(ctx context.Context) (map[string]float64, error)
	frames  atomic.Int64
	lastFrm atomic.Int64
}

// NewCollector 凭据自 app_secrets 密文列解（master key 文件径；缺件 fail-closed 上抛）。
func NewCollector(ctx context.Context, pool *pgxpool.Pool, sim bool) (*Collector, error) {
	mk, err := config.ReadMasterKey()
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	c := &Collector{pool: pool, url: PrivateURL, sim: sim,
		rej:    rejects.New("okxpriv", "parse_fail", "write_err", "creds_missing", "login_fail"),
		lastEq: map[string]float64{}}
	for k, dst := range map[string]*string{"okx.api_key": &c.creds.Key, "okx.api_secret": &c.creds.Secret, "okx.passphrase": &c.creds.Passphrase} {
		var ct []byte
		if err := pool.QueryRow(ctx, "SELECT ciphertext FROM app_secrets WHERE key=$1", k).Scan(&ct); err != nil {
			c.rej.Inc("creds_missing")
			return nil, fmt.Errorf("%s 读败: %w", k, err)
		}
		p, err := cryptoutil.Open(mk, ct)
		if err != nil {
			return nil, fmt.Errorf("%s 解密: %w", k, err)
		}
		*dst = p
	}
	return c, nil
}

// newTestRej 测试用计数门（生产由 NewCollector 装）。
func newTestRej() *rejects.Door {
	return rejects.New("okxpriv", "parse_fail", "write_err", "creds_missing", "login_fail")
}

// SetBootFn 注入 boot 首读播种器（装配层 cmd/core 接 okxexec——cmd 不受方向闸）。
func (c *Collector) SetBootFn(fn func(ctx context.Context) (map[string]float64, error)) {
	c.bootFn = fn
}

// Credentials 装配层取凭据（bootFn 接线用——值不出 pkg 外非注入径）。
func (c *Collector) Credentials() signutil.Credentials { return c.creds }
