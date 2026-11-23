// session.go —— OpenSession 有界化件（T-0053）：懒拨号首 RPC 无死线=OpenSession
// 静默悬挂（SG 生产实证）——全程死线+分阶段 log，悬挂态在日志面显形。
package mt5

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"o/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	mt5pb "o/internal/gen/mtapi/mt5"
)

// openSessionBudget OpenSession 全程死线（T-0053：懒拨号首 RPC 无死线=静默悬挂；
// 调用方免操心，超死线必返错显形禁静默占岗）。
const openSessionBudget = 30 * time.Second

// OpenSession 独立一次性会话（开连+登录——用毕调用方 Disconnect；facts 采面/
// 探针面用）。分阶段 log（dial→connect→authed 起讫；失败含阶段名+err）。
func OpenSession(ctx context.Context, pool *pgxpool.Pool, cfg *config.Client) (*Client, error) {
	// 共享先行——共享行活 token=零新 login 复挂（双腿共享会话 churn 减源；
	// shared 标置=调用侧遗留 Disconnect 幂等空转不伤共享会话）。
	if cl, err := sharedAttach(ctx, pool, Dial); err == nil {
		log.Printf("[mt5] OpenSession 复挂共享会话（零新 login）")
		return cl, nil
	}
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
	if c.sessTok == "" {
		if ss, err := sharedSessionRead(ctx, c.pool); err == nil {
			c.sessAddr, c.sessTok = ss.Addr, ss.Token // 重启续挂——共享行自读（零新 login）
		}
	}
	var cl *Client
	var clAddr string
	var lastErr error
	for _, a := range addrs {
		cand, derr := dial(a)
		if derr != nil {
			lastErr = derr
			continue
		}
		if c.sessTok != "" && a == c.sessAddr {
			cand.SetToken(c.sessTok)
			if cand.CheckAlive(ctx) == nil {
				cl, clAddr = cand, a // 复挂——流死≠会话死，零新 login 避开券商冲突掐
				break
			}
		}
		if cerr := cand.Connect(ctx, c.cr); cerr != nil {
			cand.Close()
			lastErr = cerr
			continue
		}
		cl, clAddr = cand, a
		break
	}
	if cl == nil {
		c.rej.Inc("connect_failed")
		return lastErr
	}
	c.sessAddr, c.sessTok = clAddr, cl.Token()
	sharedSessionWrite(ctx, c.pool, clAddr, cl.Token()) // owner 落共享行——消费腿据此复挂
	defer func() {
		// 共享会话不归腿登出——句柄跨腿共享+跨重启续挂（Disconnect=连坐杀
		// 共享会话坑）；僵尸界=券商 TTL 自然收尸。关停/流死皆仅收本地 conn。
		cl.Close()
	}()
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
	// O-35753 充提入账细胞——OrderHistory 轮询→fin_cashflow（会话命同寿，
	// 水位跨会话续；boot 首拍 7d 回看补漏窗，后续 15min 增量）。
	go c.balanceOpsLoop(sessCtx, cl)
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
	// O-34264 死流看门狗阈——mt5.tick_dead_sec 热启（缺省 300s 分钟级，
	// 周末/薄市容忍；订阅清单空态已有独立报错径不叠判）。
	dead := 300 * time.Second
	if v, err := c.cfg.GetFloat("mt5.tick_dead_sec"); err == nil && v > 0 {
		dead = time.Duration(v * float64(time.Second))
	}
	return c.quoteConsume(ctx, sessCtx, cancel, stream, profitErr, dead, 30*time.Second)
}

// quoteConsume OnQuote 流消费环（O-34264：活性=实报价帧——q.Symbol!=""∧
// (bid|ask!=0) 才算数据帧，心跳形 f.Result==nil/空报价帧不续命）。死流态
// A/B（后端会话被连坐杀/空心跳帧照到）Recv 永阻塞不返 err——旁路看门狗
// 每 watchEvery 检 now-lastData>dead 即 cancel sessCtx→Recv 以 ctx err
// 返回→session 走正常死法退→外层退避重连换血（复挂序 sessTok+CheckAlive
// →ConnectEx 不动）。死法判词区分：deadFired=数据帧超时优先归簇（心跳帧
// 续命假活防面——profitErr 真死因仍优先于普通流断）。
func (c *Collector) quoteConsume(ctx context.Context, sessCtx context.Context,
	cancel context.CancelFunc, stream grpc.ServerStreamingClient[mt5pb.OnQuoteReply],
	profitErr <-chan error, dead, watchEvery time.Duration) error {
	var lastData, deadFired atomic.Int64
	lastData.Store(time.Now().UnixNano())
	deadFired.Store(0)
	go func() {
		t := time.NewTicker(watchEvery) // sched-exempt-ok 死流期界巡查（非业务轮询——venue 推流哑侦测，同 bridge tickSilentAfter 先例）
		defer t.Stop()
		for {
			select {
			case <-sessCtx.Done():
				return
			case <-t.C:
				if quiet := time.Since(time.Unix(0, lastData.Load())); quiet > dead {
					deadFired.Store(1)
					log.Printf("[mt5] deadstream 数据帧超时（%s 零实报价帧，阈 %s）——收尸重连",
						quiet.Round(time.Second), dead)
					cancel()
					return
				}
			}
		}
	}()
	for {
		f, err := stream.Recv()
		if c.meter != nil && f != nil {
			c.meter.Add("mt5/quote", proto.Size(f))
		}
		if err != nil {
			if deadFired.Load() != 0 {
				return fmt.Errorf("OnQuote deadstream 数据帧超时: %w", err)
			}
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
			continue // 空帧（心跳形）不落表不续命
		}
		lastData.Store(time.Now().UnixNano())
		if _, err := c.pool.Exec(ctx,
			`INSERT INTO md_ticks(ts,venue,inst_id,last,bid,ask) VALUES(now(),'mt5',$1,$2,$3,$4)
			ON CONFLICT (venue,inst_id) DO UPDATE SET ts=EXCLUDED.ts,last=EXCLUDED.last,
			bid=EXCLUDED.bid,ask=EXCLUDED.ask,recv_ts=now()`,
			q.Symbol, q.Last, q.Bid, q.Ask); err != nil {
			c.rej.Inc("tick_write_err")
		}
	}
}
