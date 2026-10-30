package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/internal/research/gapscan"
	"o/internal/trading/okxexec"
)

type coreService struct {
	pool *pgxpool.Pool
	// T-0086 ①④：资金户/Earn 活读件（cmd 装配注入 okxexec——REST 保留面非
	// 快照；nil/读败=该户 0 不误报）。venue=唯一事实源，零落地。
	fundingLive func(ctx context.Context) (map[string]float64, error)
	earnLive    func(ctx context.Context) (map[string]float64, error)
	// T-0096：MT5 户活读件（每真实账号一行净值现读；nil/读败=venue 区显
	// 「读件缺」不藏禁假零）。
	mt5Live func(ctx context.Context) ([]mt5Acct, error)
	// T-0101：dcd 产品持仓活读件（state=live 枚举——已部署资本第三类别；
	// nil/读败=卡面显「读件缺」不假零）。
	dcdLive func(ctx context.Context) ([]okxexec.DcdOrder, error)
	// T-0156：TRX 自营盘读面同源装配（采集臂单例——读面/刷新踢同源）。
	trx *trxWiring
	// T-0170：采集进程内指标源（okxws collector+netmeter 闭包——GetSystemHealth
	// 供数；nil=观测面缺席显「读件缺」不假零）。
	md *mdStats
}

// mdStats 采集观测源束（方法值闭包——collector/meter 单例由 main.go 装配注入）。
type mdStats struct {
	frames      func() int64
	lastFrame   func() time.Time
	rejects     func() map[string]int64
	netKbps     func() float64
	netChannels func() map[string]int64
}

func (coreService) Ping(context.Context, *dashboardv1.PingRequest) (*dashboardv1.PingResponse, error) {
	return &dashboardv1.PingResponse{Status: "ok"}, nil
}

func (c coreService) Overview(ctx context.Context, _ *dashboardv1.OverviewRequest) (*dashboardv1.OverviewResponse, error) {
	o := &dashboardv1.OverviewResponse{Ts: time.Now().UTC().Format(time.RFC3339)}
	// 钱卡+NAV 组装迁 money_card.go（行数棘轮——T-0100/0096/0101 三区历史在彼）。
	c.moneyRows(ctx, o)

	// 盈亏=经营性四件（interest/funding_settle/trade_pnl/fee）——
	// deposit/withdraw/earn_*/recon_adjust 全出局（口径正名锚）。
	const opKinds = "('interest','funding_settle','trade_pnl','fee')"
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE occurred_at::date=CURRENT_DATE AND kind IN `+opKinds).Scan(&o.PnlToday)
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
		WHERE kind IN `+opKinds).Scan(&o.PnlCum)
	// T-0083：平仓仓现货腿净额=真价差入盈亏（持仓期买腿流出=资产置换非亏损
	// ——只计已平仓仓腿行，防置换流出当亏）。
	var legToday, legCum float64
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(c.amount),0) FROM fin_cashflow c
		JOIN fin_position p ON p.id=c.position_ref
		WHERE c.occurred_at::date=CURRENT_DATE AND c.kind IN ('spot_buy','spot_sell')
		  AND p.closed_at IS NOT NULL`).Scan(&legToday)
	_ = c.pool.QueryRow(ctx, `SELECT COALESCE(sum(c.amount),0) FROM fin_cashflow c
		JOIN fin_position p ON p.id=c.position_ref
		WHERE c.kind IN ('spot_buy','spot_sell') AND p.closed_at IS NOT NULL`).Scan(&legCum)
	o.PnlToday += legToday
	o.PnlCum += legCum
	// T-0102 v3：面差待裁计数露一行（业主可见「系统发现了几件还没裁的事」）。
	if n, gerr := gapscan.PendingCount(ctx, c.pool); gerr == nil && n > 0 {
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Stamp: time.Now().UTC().Format(time.RFC3339),
			Text: fmt.Sprintf("待办：发现 %d 个未用的交易所功能——等裁决", n)})
	}
	// T-0105 在干列表全集化：策略对冲仓+结构性持仓（DCD live）两源入列
	// （所见即所得——DCD 是基金第二大资产块却不在「持仓中」=业主令病灶）；
	// 展示引用不重复入 NAV（钱卡 dcdRows 已计值）。静持空态判据=基金真空
	// 持仓（无 open 仓且无 DCD live 单）；DCD 读件缺显形不静默（不假零）。
	prow, _ := c.pool.Query(ctx, `SELECT id,inst_id,opened_at FROM fin_position
		WHERE closed_at IS NULL ORDER BY id DESC LIMIT 5`)
	posN := 0
	for prow.Next() {
		var id int64
		var inst string
		var at time.Time
		_ = prow.Scan(&id, &inst, &at)
		posN++
		base := strings.SplitN(inst, "-", 2)[0]
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{
			Text:  fmt.Sprintf("%s 对冲仓 #%d — 持仓中，已持 %s", base, id, durHuman(time.Since(at))),
			Stamp: at.UTC().Format(time.RFC3339)})
	}
	prow.Close()
	dcds, ok := c.dcdOrds(ctx)
	for _, od := range dcds {
		u := valCcy(ctx, c.pool, od.NotionalCcy, okxexec.ParseF(od.NotionalSz))
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{
			Text: fmt.Sprintf("DCD %s — 持仓中，估值 $%.2f",
				od.ProductID, u),
			Stamp: msStamp(od.CTime)})
	}
	if !ok {
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Stamp: time.Now().UTC().Format(time.RFC3339),
			Text: "结构性持仓——读件缺（dcd 活读缺席/读败，不假零）"})
	}
	if posN == 0 && ok && len(dcds) == 0 {
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Text: "静持——无开仓在案"})
	}
	rrows, _ := c.pool.Query(ctx, `SELECT ts,kind,amount,coalesce(venue,'') FROM fin_cashflow
		WHERE kind NOT IN ('earn_in','earn_out') ORDER BY id DESC LIMIT 6`)
	for rrows.Next() {
		var ts time.Time
		var kind, venue string
		var amt float64
		_ = rrows.Scan(&ts, &kind, &amt, &venue)
		verb := map[string]string{"funding_settle": "资金费到账", "fee": "交易成本",
			"trade_pnl": "平仓盈亏", "deposit": "入金", "withdraw": "出金",
			"interest": "活期利息", "recon_adjust": "校账调整",
			"spot_buy": "现货买入", "spot_sell": "现货卖出"}[kind]
		src := map[string]string{"okx": "OKX", "mt5": "MT5"}[venue]
		if src != "" {
			// 钱包/账号细化（业主令）：OKX 按 kind 语义映钱包（利息=Earn 停泊/
			// 入出金=资金户/交易类=交易户）；MT5=在册 login 单户直标（多户时
			// 退化为平台名——治本=写账落 account 列，另票）。
			wallet := map[string]string{"interest": "Earn 停泊",
				"funding_settle": "交易户", "trade_pnl": "交易户", "fee": "交易户",
				"deposit": "资金户", "withdraw": "资金户"}[kind]
			if venue == "mt5" {
				var login string
				_ = c.pool.QueryRow(ctx, `SELECT data->>'login' FROM md_account_events
					WHERE venue='mt5' AND data ? 'login' ORDER BY ts DESC LIMIT 1`).Scan(&login)
				if login != "" {
					src += "·户" + login
				}
			} else if wallet != "" {
				src += "·" + wallet
			}
			src = " · 来自" + src
		}
		o.Recent = append(o.Recent, fmt.Sprintf("%s %s %+.4f USDT%s",
			ts.Format("15:04"), verb, amt, src))
	}
	rrows.Close()
	// T-0078：列名正名 duty→title（旧列缺席=静默空集病灶——Scan 败必 log
	// 一次不静默吞）。
	srows, err := c.pool.Query(ctx, `SELECT name,title,last_beat FROM fin_employee_registry
		WHERE status='active' ORDER BY name`)
	if err != nil {
		log.Printf("staff 花名册查询败: %v", err)
		return o, nil // 花名册读败=staff 卡藏如实（不炸总览）
	}
	for srows.Next() {
		var name, duty string
		var beat time.Time
		if err := srows.Scan(&name, &duty, &beat); err != nil {
			srows.Close()
			return o, fmt.Errorf("staff 行扫败: %w", err)
		}
		st := int32(0)
		if time.Since(beat) > 5*time.Minute {
			st = 2
		} else if time.Since(beat) > time.Minute {
			st = 1
		}
		o.Staff = append(o.Staff, &dashboardv1.StaffRow{Name: name, Duty: duty, State: st})
	}
	srows.Close()
	o.Venues = c.venues(ctx)
	return o, nil
}

// watchHeartbeat 心跳期界（T-0269——锚：帧间隔 ≤20s，取 15s 留测量余量；
// var=测试缝——桩测缩期界验节拍语义，产态恒 15s）。
var watchHeartbeat = 15 * time.Second

// Watch SSE 脊推变（ev_* NOTIFY——所见即所得刷新源，事件驱动律守不新造环）。
func (c coreService) Watch(ctx context.Context, _ *dashboardv1.WatchRequest,
	stream dashboardv1connect.CoreServiceWatchServerStream) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	for _, ch := range []string{"ev_position", "ev_cashflow", "ev_account", "ev_strategy", "ev_hostmetrics", "ev_headless"} {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return err
		}
	}
	// watch_open 初始帧=就绪信号（LISTEN 注册完成后客户端才收到首帧——
	// 连接即回填，前端借帧刷新快照；非 ev_* 通道，数据事件语义不混）。
	// connect 服务端流首写前不出响应头——无此帧客户端 Watch() 悬挂至首事件。
	if err := stream.Send(&dashboardv1.WatchEvent{Kind: "watch_open",
		Ts: time.Now().UTC().Format(time.RFC3339)}); err != nil {
		return err
	}
	// T-0269 应用层心跳：事件静默期 ≤15s 一帧（CF h2 idle 杀流残余风险关窗
	// ——期界等待形非周期定时器，事件到达仍即时推不阻塞）。
	for {
		wctx, cancel := context.WithTimeout(ctx, watchHeartbeat)
		n, err := conn.Conn().WaitForNotification(wctx)
		cancel()
		if n != nil {
			ev := &dashboardv1.WatchEvent{Kind: n.Channel, Ts: time.Now().UTC().Format(time.RFC3339)}
			// T-0220a 一跳推数：NOTIFY 即按 kind 查表带载荷（查/转败=帧照发+log
			// 显形——信号语义保留，不炸流不静默；无载荷面 kind=纯信号帧）。
			st, perr := c.watchPayload(ctx, n.Channel)
			if perr != nil {
				log.Printf("watch 载荷 %s 查败（帧照发）: %v", n.Channel, perr)
			} else if st != nil {
				ev.Data = st
			}
			if serr := stream.Send(ev); serr != nil {
				return serr
			}
		}
		if err == nil {
			continue
		}
		if !errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return err
		}
		// 期界到=保真帧（kind=watch_hb 纯信号无载荷——前端对未知 kind 天然
		// 忽略，不入数据语义）。
		if serr := stream.Send(&dashboardv1.WatchEvent{
			Kind: "watch_hb", Ts: time.Now().UTC().Format(time.RFC3339)}); serr != nil {
			return serr
		}
	}
}

func durHuman(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// mountCore 挂 ConnectRPC 服务到 mux（connect-go v2：Server 注册 + connecthttp.Mount；
// T-0156：REST 双轨全撤——CoreService 单源，trx 读面同 server 注册）。
func mountCore(mux *http.ServeMux, svc coreService) {
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, svc)
	connecthttp.Mount(mux, server)
}

// assemble 复盘域读面单例装配（T-0170：coreService 一处装配两处消费——
// CoreService 挂载与 NAV 日拍闭包同源，live 读件不重复造）。
func assemble(pool *pgxpool.Pool,
	fundingLive func(context.Context) (map[string]float64, error),
	earnLive func(context.Context) (map[string]float64, error),
	mt5Live func(context.Context) ([]mt5Acct, error),
	dcdLive func(context.Context) ([]okxexec.DcdOrder, error), trxW *trxWiring,
	md *mdStats) (coreService, func(context.Context) (float64, error)) {
	svc := coreService{pool: pool, fundingLive: fundingLive, earnLive: earnLive,
		mt5Live: mt5Live, dcdLive: dcdLive, trx: trxW, md: md}
	return svc, func(ctx context.Context) (float64, error) {
		o := &dashboardv1.OverviewResponse{}
		svc.moneyRows(ctx, o)
		return o.NavTotal, nil
	}
}

// mountUI 迁 ui.go（T-0269——favicon 挂接增量撑行，静态面自成件）。
