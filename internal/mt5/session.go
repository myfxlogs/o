// session.go —— OpenSession 有界化件（T-0053）：懒拨号首 RPC 无死线=OpenSession
// 静默悬挂（SG 生产实证）——全程死线+分阶段 log，悬挂态在日志面显形。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"o/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/proto"
)

// openSessionBudget OpenSession 全程死线（T-0053：懒拨号首 RPC 无死线=静默悬挂；
// 调用方免操心，超死线必返错显形禁静默占岗）。
const openSessionBudget = 30 * time.Second

// OpenSession 独立一次性会话（开连+登录——用毕调用方 Disconnect；facts 采面/
// 探针面用）。分阶段 log（dial→connect→authed 起讫；失败含阶段名+err）。
func OpenSession(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Client, error) {
	addrs, cr, err := endpoints(ctx, pool, cfg)
	if err != nil {
		return nil, fmt.Errorf("阶段=creds: %w", err)
	}
	sctx, cancel := context.WithTimeout(ctx, openSessionBudget)
	defer cancel()
	var lastErr error
	for _, addr := range addrs {
		log.Printf("[mt5] OpenSession dial 起 addr=%s", addr)
		cl, err := Dial(addr)
		if err != nil {
			log.Printf("WARN [mt5] OpenSession 阶段=dial 败 addr=%s: %v", addr, err)
			lastErr = err
			continue
		}
		log.Printf("[mt5] OpenSession connect 起（budget %s）", openSessionBudget)
		if err := cl.Connect(sctx, cr); err != nil {
			cl.Close()
			log.Printf("WARN [mt5] OpenSession 阶段=connect 败 addr=%s（超死线/登录拒非悬挂）: %v", addr, err)
			lastErr = err
			continue
		}
		log.Printf("[mt5] OpenSession authed（会话建，用毕 Disconnect）")
		return cl, nil
	}
	return nil, fmt.Errorf("阶段=connect: %w", lastErr)
}

// endpoints 网关址链+凭据读（SG 侧会话径）。址链=gatewayAddrs 同件——
// OCI 桥腿（env 凭据）与本件（密文凭据）共享同一址源，端点热更两桥同愈。
func endpoints(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) ([]string, Credentials, error) {
	addrs, err := gatewayAddrs(cfg)
	if err != nil {
		return nil, Credentials{}, err
	}
	_, cr, err := readCreds(ctx, pool, cfg)
	if err != nil {
		return nil, Credentials{}, err
	}
	return addrs, cr, nil
}

// gatewayAddrs 网关址链读——mt5.gateway_addr 主址+mt5.gateway_fallbacks JSON
// 字符串列备用址（缺/坏=仅主址；备用址空=单端点旧行为）。每调重读=
// app_config 端点热更免重启（collector 启动缓存坑根修面）。
func gatewayAddrs(cfg *config.Client) ([]string, error) {
	raw, err := cfg.Get("mt5.gateway_addr")
	if err != nil {
		return nil, fmt.Errorf("mt5.gateway_addr: %w", err)
	}
	var addr string
	if err := json.Unmarshal(raw, &addr); err != nil {
		return nil, err
	}
	addrs := []string{addr}
	if raw, err := cfg.Get("mt5.gateway_fallbacks"); err == nil {
		var fb []string
		if json.Unmarshal(raw, &fb) == nil {
			for _, a := range fb {
				if a = strings.TrimSpace(a); a != "" && a != addr {
					addrs = append(addrs, a)
				}
			}
		}
	}
	return addrs, nil
}

// session 一次会话生命周期：Connect→boot（账户快照+符号面+订阅）→OnQuote 流消费。
// 返回=流断/写败（外层退避重连）；成功消费一帧由调用侧无法感知——流常驻直至断。
func (c *Collector) session(ctx context.Context, dial func(string) (*Client, error)) error {
	addrs, cr, err := endpoints(ctx, c.pool, c.cfg)
	if err != nil {
		return err
	}
	c.cr = cr
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
		c.rej.Inc("connect_failed")
		return lastErr
	}
	defer cl.Close()
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
	// 账户权益推送流（OnOrderProfit——持仓期 ~1Hz ProfitUpdate 全摘要帧；
	// push-on-change：空仓静默=无变动非流死。任一框架死=会话退重连）。
	sessCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if n, err := cl.SubscribeOrderProfit(ctx); err != nil {
		c.rej.Inc("subscribe_failed")
		log.Printf("[mt5] SubscribeOrderProfit 败（流实测免订阅自推，不阻塞）: %v", err)
	} else {
		log.Printf("[mt5] order_profit 订阅回执: 在册 %d 仓", n)
	}
	pstr, err := cl.OnOrderProfit(sessCtx)
	if err != nil {
		return fmt.Errorf("OnOrderProfit 开流: %w", err)
	}
	profitErr := make(chan error, 1)
	go func() {
		err := c.profitLoop(ctx, pstr)
		profitErr <- err // 先送因后 cancel——主循环 Recv 断时真因可取
		cancel()
	}()
	stream, err := cl.OnQuote(sessCtx)
	if err != nil {
		return err
	}
	for {
		f, err := stream.Recv()
		if c.meter != nil && f != nil {
			c.meter.Add("mt5/quote", proto.Size(f))
		}
		if err != nil {
			select {
			case pe := <-profitErr:
				return fmt.Errorf("OnOrderProfit 流断: %w", pe)
			default:
			}
			return fmt.Errorf("OnQuote 流断: %w", err)
		}
		select {
		case pe := <-profitErr:
			return fmt.Errorf("OnOrderProfit 流断: %w", pe)
		default:
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
