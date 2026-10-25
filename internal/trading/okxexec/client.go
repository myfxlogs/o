// Package okxexec —— OKX 下单/账户 REST 钱径最小面（T-0008：接口+单测+模拟径，
// 禁真单）。纪律：凭据只自 app_secrets 密文列解；h1 钉死传输（h2 感染病灶——
// assembleRT 同律：NextProtos=["http/1.1"]+TLSNextProto 空表）；下单超时→clOrdId
// 定态三分支；歧态势禁补单。
package okxexec

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"o/internal/platform/signutil"
)

const (
	BaseURL = "https://www.okx.com"
	// demo 户头 x-simulated-trading:1（真户永久禁跑——mutating 面只走模拟）。
	simulatedHeader = "x-simulated-trading"
)

// Credentials OKX key 三件套（自密文列解出，仅存内存）——定义挂 signutil
// 防数据域反向依赖交易域。
type Credentials = signutil.Credentials

// Client OKX 私有 REST 客户端（h1 钉死）。
type Client struct {
	creds   Credentials
	base    string
	hc      *http.Client
	sim     bool
	ro      bool       // T-0104 D-0013：只读镜像件——venue 写径结构不可达
	dcdMu   sync.Mutex // dcd LIVE 单 60s 单飞窗（限流端点——429 实证）
	dcdAt   time.Time
	dcdOrds []DcdOrder
	dcdErr  error
}

// NewClient h1 钉死 transport（NextProtos/TLSNextProto 双钉——clone 浅继中毒字段
// 族教训，裸 DefaultTransport 禁行）。拨号钉 tcp4（T-0102：OKX key 白名单按源
// IP 判——IPv6 egress 双栈漂移触发 50110 间歇拒，坑条 okx-dcd-endpoint-dialect）。
func NewClient(creds Credentials, simulated bool) *Client {
	d := &net.Dialer{Timeout: 10 * time.Second}
	tr := &http.Transport{
		TLSClientConfig:   &tls.Config{NextProtos: []string{"http/1.1"}},
		TLSNextProto:      map[string]func(string, *tls.Conn) http.RoundTripper{},
		ForceAttemptHTTP2: false,
		DisableKeepAlives: true, // FreshClient 口径：钱径每请新 conn
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if network == "tcp" {
				network = "tcp4" // 白名单按 v4 源判——双栈漂移禁（50110 坑条）
			}
			return d.DialContext(ctx, network, addr)
		},
	}
	return &Client{creds: creds, base: BaseURL, hc: &http.Client{Transport: tr, Timeout: 15 * time.Second}, sim: simulated}
}

// NewClientReadOnly 只读镜像件（T-0104 D-0013——凭据同源同件但 venue 写径
// 结构不可达：do() 硬闸非 GET 全拒，非业务层自律 flag——盲库节点 mutating
// 面从「双跑」降格「结构不跑」）。读径与常规件全同。
func NewClientReadOnly(creds Credentials, simulated bool) *Client {
	c := NewClient(creds, simulated)
	c.ro = true
	return c
}

// ErrReadOnly 只读件写径拒（结构不可达——非 venue 业务拒，类型化供调用方判）。
var ErrReadOnly = fmt.Errorf("只读镜像件——venue 写径结构不可达（D-0013/T-0104）")

// flexCode 信封 code 联合型（T-0102：sfp/dcd 族 code 是 number 非 string——
// string 解析拒=200 真数据被信封方言误杀，坑条 okx-dcd-endpoint-dialect；
// string|number 双收，APIError.Code 保持 string 形不变）。
type flexCode string

func (f *flexCode) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexCode(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexCode(n.String())
	return nil
}

// do 签名请求：四头 + demo 头（sim 时）。返回 body 字节；HTTP 非 2xx 或 code≠"0" 上抛。
func (c *Client) do(ctx context.Context, method, pathWithQuery string, body []byte) ([]byte, error) {
	if c.ro && method != http.MethodGet {
		return nil, ErrReadOnly // T-0104：写径签名前硬闸——非 GET 一律不达 venue
	}
	ts := signutil.OkxTS(time.Now())
	req, err := http.NewRequestWithContext(ctx, method, c.base+pathWithQuery, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("OK-ACCESS-KEY", c.creds.Key)
	req.Header.Set("OK-ACCESS-SIGN", signutil.OkxSign(c.creds.Secret, ts, method, pathWithQuery, string(body)))
	req.Header.Set("OK-ACCESS-TIMESTAMP", ts)
	req.Header.Set("OK-ACCESS-PASSPHRASE", c.creds.Passphrase)
	req.Header.Set("Content-Type", "application/json")
	if c.sim {
		req.Header.Set(simulatedHeader, "1")
	}
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("httpStatusError: %w", err) // 传输败≠业务拒——clOrdId 定态径消费
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	var env struct {
		Code flexCode        `json:"code"`
		Msg  string          `json:"msg"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(b, &env); err != nil {
		return nil, fmt.Errorf("resp parse: %w", err)
	}
	if env.Code != "0" {
		ae := &APIError{Code: string(env.Code), Msg: env.Msg}
		var inner []struct {
			SCode string `json:"sCode"`
			SMsg  string `json:"sMsg"`
		}
		if json.Unmarshal(env.Data, &inner) == nil && len(inner) > 0 {
			ae.SCode, ae.SMsg = inner[0].SCode, inner[0].SMsg
		}
		return nil, ae
	}
	return env.Data, nil
}

// Get 签名 GET（事务按需单发——无轮询用法）。
func (c *Client) Get(ctx context.Context, pathWithQuery string) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, pathWithQuery, nil)
}

// Post 签名 POST（事务面：set-leverage/transfer 族——body 为完整 JSON）。
func (c *Client) Post(ctx context.Context, path string, body []byte) (json.RawMessage, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// BalanceEquity GET /account/balance → ccy→eqUsd 表（boot 锚定首读用——
// 事务按需单发，非轮询面）。
func BalanceEquity(ctx context.Context, creds Credentials, simulated bool) (map[string]float64, error) {
	raw, err := NewClient(creds, simulated).Get(ctx, "/api/v5/account/balance")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Details []struct {
			Ccy   string `json:"ccy"`
			EqUsd string `json:"eqUsd"`
			Eq    string `json:"eq"`
		} `json:"details"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return nil, fmt.Errorf("balance parse: %w rows=%d", err, len(rows))
	}
	out := map[string]float64{}
	for _, d := range rows[0].Details {
		if v := parseF(d.EqUsd); v != 0 {
			out[d.Ccy] = v
		} else if v = parseF(d.Eq); v != 0 {
			out[d.Ccy] = v
		}
	}
	return out, nil
}

// TradingAvailBal GET /account/balance → ccy availBal（交易户闲余口径——
// T-0048 返修：earn 摆渡自 AcctTrading，量资金户=错户；钦定事务面单发）。
func (c *Client) TradingAvailBal(ctx context.Context, ccy string) (float64, error) {
	raw, err := c.Get(ctx, "/api/v5/account/balance")
	if err != nil {
		return 0, err
	}
	var rows []struct {
		Details []struct {
			Ccy      string `json:"ccy"`
			AvailBal string `json:"availBal"`
		} `json:"details"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return 0, fmt.Errorf("balance parse: %w", err)
	}
	for _, d := range rows[0].Details {
		if d.Ccy == ccy {
			return parseF(d.AvailBal), nil
		}
	}
	return 0, fmt.Errorf("ccy %s 缺席", ccy)
}

// AssetBalance /asset/balances（资金户——funding 户无推送频道，钦定保留面）。
func (c *Client) AssetBalances(ctx context.Context) ([]struct {
	Ccy, Bal, AvailBal, FrozenBal string
}, error) {
	raw, err := c.Get(ctx, "/api/v5/asset/balances")
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Ccy       string `json:"ccy"`
		Bal       string `json:"bal"`
		AvailBal  string `json:"availBal"`
		FrozenBal string `json:"frozenBal"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	out := make([]struct{ Ccy, Bal, AvailBal, FrozenBal string }, len(rows))
	for i, r := range rows {
		out[i].Ccy, out[i].Bal, out[i].AvailBal, out[i].FrozenBal = r.Ccy, r.Bal, r.AvailBal, r.FrozenBal
	}
	return out, nil
}

// SavingsBalance /finance/savings/balance（earn 活期货产——钦定保留面）。
func (c *Client) SavingsBalance(ctx context.Context) (json.RawMessage, error) {
	return c.Get(ctx, "/api/v5/finance/savings/balance")
}

// PublicGet 公开端点无签名径（instruments 等钦定保留面——零凭据）。
func (c *Client) PublicGet(ctx context.Context, pathWithQuery string) (json.RawMessage, error) {
	return c.Get(ctx, pathWithQuery)
}

// InstrumentMeta /public/instruments 取 ctVal/lotSz/minSz（公开只读——instruments
// bootstrap 钦定面非轮询病灶）。
func (c *Client) InstrumentMeta(ctx context.Context, instType, instID string) (struct {
	CtVal, LotSz, MinSz, TickSz float64
	CtValCcy, State             string
}, error) {
	var out struct {
		CtVal, LotSz, MinSz, TickSz float64
		CtValCcy, State             string
	}
	raw, err := c.Get(ctx, "/api/v5/public/instruments?instType="+instType+"&instId="+instID)
	if err != nil {
		return out, err
	}
	var rows []struct {
		CtVal    string `json:"ctVal"`
		LotSz    string `json:"lotSz"`
		MinSz    string `json:"minSz"`
		TickSz   string `json:"tickSz"`
		CtValCcy string `json:"ctValCcy"`
		State    string `json:"state"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil || len(rows) == 0 {
		return out, fmt.Errorf("instruments 空: %w", err)
	}
	out.CtVal, _ = strconv.ParseFloat(rows[0].CtVal, 64)
	out.LotSz, _ = strconv.ParseFloat(rows[0].LotSz, 64)
	out.MinSz, _ = strconv.ParseFloat(rows[0].MinSz, 64)
	out.CtValCcy = rows[0].CtValCcy
	out.TickSz, _ = strconv.ParseFloat(rows[0].TickSz, 64)
	out.State = rows[0].State
	return out, nil
}

// Creds 凭据取件（boot 首读/facts 采面装配注入用——值不出 pkg 非注入径）。
func (c *Client) Creds() Credentials { return c.creds }
