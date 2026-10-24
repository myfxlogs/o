// Package mt5 —— MT5 数据面只读接入（T-0031：第二 venue 读面——行情+账户事实，
// 零钱径；mt5-trading-mutating 面本票禁用，禁锚测试在 client_test.go）。
// 协议=mtapi gRPC 网关（skills/mt5-gateway-admin 实测契约）：
//   - token 语义=Connect 返回值回填后续每 Request 的 Id 体字段（非 header——header
//     径网关不读实拍）；Connect 时以 metadata id=<临时会话号> 发起。
//   - 双轨寻址：Server 非空→ConnectEx（server 名径）；否则 Host:Port 径。
//   - dial=TLS1.2+keepalive 30s（长流必须，无则被中间件静默割）；Disconnect 后
//     ~200ms 排水再关 conn。
//   - 符号方言：symbol 名一律 SymbolList 实读禁猜（Exness=m 尾实拍）。
//
// 流正道（D-004）：quote 走 Subscribe+Events OnQuote 流（mt5 无 poll 报价语义）；
// 账户快照 boot 首读+流重连愈合（事件锚定形，周期态摄入零新增）。
package mt5

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"

	mt5pb "phase/internal/gen/mtapi/mt5"
)

// Credentials MT5 户三件套（自 app_secrets 密文列解——demo 户如实注）。
type Credentials struct {
	Login    uint64
	Password string
	Server   string // 非空=ConnectEx server 名径；空=Host 径
	Host     string // 逗号分隔多候选（broker_search Access 列表灌入形）
}

// Client mtapi 只读会话客户端（Connect 持会话；token 体域携行）。
type Client struct {
	conn  *grpc.ClientConn
	connM mt5pb.ConnectionClient
	mt5   mt5pb.MT5Client
	subs  mt5pb.SubscriptionsClient
	strm  mt5pb.StreamsClient
	token string
}

// Dial 网关连接（TLS+keepalive——长流参数集）。
func Dial(gatewayAddr string) (*Client, error) {
	cc, err := grpc.NewClient(gatewayAddr, grpc.WithTransportCredentials(credentials.NewTLS(nil)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time: 30 * time.Second, Timeout: 20 * time.Second, PermitWithoutStream: true,
		}), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(16<<20)))
	if err != nil {
		return nil, err
	}
	return &Client{
		conn:  cc,
		connM: mt5pb.NewConnectionClient(cc),
		mt5:   mt5pb.NewMT5Client(cc),
		subs:  mt5pb.NewSubscriptionsClient(cc),
		strm:  mt5pb.NewStreamsClient(cc),
	}, nil
}

// Connect 登录换 token（Server 名径优先；Host 多候选逐试）。
func (c *Client) Connect(ctx context.Context, cr Credentials) error {
	sid := fmt.Sprintf("phase-%d", time.Now().UnixNano())
	loginCtx := metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": sid}))
	if cr.Server != "" {
		resp, err := c.connM.ConnectEx(loginCtx, &mt5pb.ConnectExRequest{
			User: cr.Login, Password: cr.Password, Server: cr.Server,
		})
		if err != nil {
			return fmt.Errorf("ConnectEx: %w", err)
		}
		if resp.Error != nil {
			return fmt.Errorf("ConnectEx err: %s", resp.Error.Message)
		}
		c.token = resp.Result
		return nil
	}
	for _, h := range splitHosts(cr.Host) {
		host, port := splitHostPort(h)
		resp, err := c.connM.Connect(loginCtx, &mt5pb.ConnectRequest{
			User: cr.Login, Password: cr.Password, Host: host, Port: port,
		})
		if err != nil {
			continue // 多候选逐试（skill 装配面契约）
		}
		if resp.Error == nil && resp.Result != "" {
			c.token = resp.Result
			return nil
		}
	}
	return fmt.Errorf("Connect 全候选败（host=%q）", cr.Host)
}

// authed 携 token 的请求 ctx（Id 体字段由各 RPC 调用方填——此处只备会话元数据）。
func (c *Client) authed(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.New(map[string]string{"id": c.token}))
}

// AccountSummary 权益/保证金实读（fin_venue_account_snapshot 源——USD 本位）。
func (c *Client) AccountSummary(ctx context.Context) (equity, freeMargin, balance float64, err error) {
	s, err := c.mt5.AccountSummary(c.authed(ctx), &mt5pb.AccountSummaryRequest{Id: c.token})
	if err != nil {
		return 0, 0, 0, err
	}
	if s.Result != nil {
		return s.Result.Equity, s.Result.FreeMargin, s.Result.Balance, nil
	}
	if a, err := c.mt5.Account(c.authed(ctx), &mt5pb.AccountRequest{Id: c.token}); err == nil && a.Result != nil {
		return a.Result.Balance, 0, a.Result.Balance, nil
	}
	return 0, 0, 0, fmt.Errorf("AccountSummary 空体")
}

// SymbolConfirm 候选符号 venue 确认制（方言名源）：SymbolsReply.SymbolInfo 无名字段、
// SymbolParamsMany 整批带入任一不存在符号=全拒（"Symbol not found"——Exness 实拍），
// 名字唯一读径=逐候选单查 SymbolParams→存在者回确。候选=技能卡方言谱（Exness=m 尾）
// +裸名双形；**返回者皆 venue 实认存在**，未认候选弃（禁猜名字面量直用的落地面）。
func (c *Client) SymbolConfirm(ctx context.Context, candidates []string) []string {
	var out []string
	for _, sym := range candidates {
		r, err := c.mt5.SymbolParams(c.authed(ctx), &mt5pb.SymbolParamsRequest{Id: c.token, Symbol: sym})
		if err != nil || r.Error != nil || r.Result == nil {
			continue // 不存在候选弃（venue 判）
		}
		out = append(out, sym)
	}
	return out
}

// Subscribe 行情订阅（幂等——会话订阅状态在网关侧）。
func (c *Client) Subscribe(ctx context.Context, symbol string) error {
	r, err := c.subs.Subscribe(c.authed(ctx), &mt5pb.SubscribeRequest{Id: c.token, Symbol: symbol})
	if err != nil {
		return err
	}
	if r.Error != nil {
		return fmt.Errorf("subscribe %s: %s", symbol, r.Error.Message)
	}
	return nil
}

// OnQuote 报价流（先 Subscribe 后流——流正道 D-004；断线=err 返回由调用方重连）。
func (c *Client) OnQuote(ctx context.Context) (grpc.ServerStreamingClient[mt5pb.OnQuoteReply], error) {
	return c.strm.OnQuote(c.authed(ctx), &mt5pb.OnQuoteRequest{Id: c.token})
}

// Disconnect 会话登出（200ms 排水窗后由 Close 收 conn）。
func (c *Client) Disconnect(ctx context.Context) {
	if c.token == "" {
		return
	}
	_, _ = c.connM.Disconnect(c.authed(ctx), &mt5pb.DisconnectRequest{Id: c.token})
	time.Sleep(200 * time.Millisecond) // 在飞 Recv 观察取消窗（QUOTE-RECONNECT-LOOP 坑）
	c.token = ""
}

// Close 收 conn。
func (c *Client) Close() { _ = c.conn.Close() }

func splitHosts(h string) []string {
	var out []string
	for _, p := range strings.Split(h, ",") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitHostPort 末冒号剥端口（缺省 443——skill 装配面契约）。
func splitHostPort(h string) (string, int32) {
	for i := len(h) - 1; i >= 0; i-- {
		if h[i] == ':' {
			if p, err := strconv.Atoi(h[i+1:]); err == nil {
				return h[:i], int32(p)
			}
			break
		}
	}
	return h, 443
}
