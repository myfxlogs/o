package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect/v2"
	connecthttp "connectrpc.com/connect/v2/connecthttp"
	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/gen/dashboardv1"
	"o/internal/gen/dashboardv1/dashboardv1connect"
	"o/web"
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
}

func (coreService) Ping(context.Context, *dashboardv1.PingRequest) (*dashboardv1.PingResponse, error) {
	return &dashboardv1.PingResponse{Status: "ok"}, nil
}

func (c coreService) Overview(ctx context.Context, _ *dashboardv1.OverviewRequest) (*dashboardv1.OverviewResponse, error) {
	o := &dashboardv1.OverviewResponse{Ts: time.Now().UTC().Format(time.RFC3339)}
	// T-0100 ①②：NAV 与三区卡面=基金的钱口径账户域作用（FundCcy 唯一真源）
	// ——交易户=USDT∪open 腿基币；资金户/Earn=USDT∪停泊溯源币（residual_park
	// 凭据）；域外同币（Earn 利息尘/腿基币渗漏）=托管尘不返不渲染不入 NAV。
	// 交易户=WS 帧真源 eqUsd，资金户/Earn=REST 活读 amt 估值（缺户=0 不误报）。
	fundT := c.fundCcy(ctx, "okx:trading")
	fundF := c.fundCcy(ctx, "okx:funding")
	fundE := c.fundCcy(ctx, "okx:earn")
	var frame json.RawMessage
	if err := c.pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&frame); err == nil {
		var top struct {
			Details []struct {
				Ccy   string `json:"ccy"`
				EqUsd string `json:"eqUsd"`
			} `json:"details"`
		}
		if json.Unmarshal(frame, &top) == nil {
			for _, d := range top.Details {
				v, _ := strconv.ParseFloat(d.EqUsd, 64)
				if v == 0 || !fundT[d.Ccy] {
					continue
				}
				o.NavTotal += v
				o.Money = append(o.Money, moneyRow("okx", "交易户", d.Ccy,
					nameOf(d.Ccy), v, v))
			}
		}
	}
	// 资金户/Earn 行=原生代币数（展示如实）；NAV 计值=amt估值（T-0086 ①）——
	// USDT/USD 直入，其余按 md_ticks 最新价折算（无 tick=0 不入 NAV 仍上卡）。
	if c.fundingLive != nil {
		if fb, err := c.fundingLive(ctx); err == nil {
			for ccy, v := range fb {
				if v == 0 || !fundF[ccy] {
					continue
				}
				u := valCcy(ctx, c.pool, ccy, v)
				o.NavTotal += u
				o.Money = append(o.Money, moneyRow("okx", "资金户", ccy,
					nameOf(ccy), v, u))
			}
		}
	}
	if c.earnLive != nil {
		if eb, err := c.earnLive(ctx); err == nil {
			for ccy, v := range eb {
				if v == 0 || !fundE[ccy] {
					continue
				}
				u := valCcy(ctx, c.pool, ccy, v)
				o.NavTotal += u
				o.Money = append(o.Money, moneyRow("okx", "Earn", ccy,
					nameOf(ccy), v, u))
			}
		}
	}
	// T-0096：MT5 入户（基金专属券商户=全部基金钱，全计）——每真实账号一行
	// 净值现读；读件缺/读败=该 venue 区显「读件缺」行不藏（禁假零）。
	// nav_total += MT5 净值合计——基金全资产口径归真。
	if c.mt5Live == nil {
		o.Money = append(o.Money, &dashboardv1.MoneyRow{
			Venue: "mt5", Section: "户", Label: "读件缺", Note: "读件缺"})
	} else if accts, aerr := c.mt5Live(ctx); aerr != nil {
		log.Printf("WARN: MT5 户活读败（钱在哪卡显读件缺）: %v", aerr)
		o.Money = append(o.Money, &dashboardv1.MoneyRow{
			Venue: "mt5", Section: "户", Label: "读件缺", Note: "读件缺"})
	} else {
		for _, a := range accts {
			o.NavTotal += a.Equity
			o.Money = append(o.Money, moneyRow("mt5", "户", "USD",
				"户 "+a.Login, a.Equity, a.Equity))
		}
	}
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
	prow, _ := c.pool.Query(ctx, `SELECT id,inst_id,opened_at FROM fin_position
		WHERE closed_at IS NULL ORDER BY id DESC LIMIT 5`)
	for prow.Next() {
		var id int64
		var inst string
		var at time.Time
		_ = prow.Scan(&id, &inst, &at)
		base := strings.SplitN(inst, "-", 2)[0]
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{
			Text:  fmt.Sprintf("%s 对冲仓 #%d — 持仓中，已持 %s", base, id, durHuman(time.Since(at))),
			Stamp: at.UTC().Format(time.RFC3339)})
	}
	prow.Close()
	if len(o.Doing) == 0 {
		o.Doing = append(o.Doing, &dashboardv1.DoingItem{Text: "静持——无开仓在案"})
	}
	rrows, _ := c.pool.Query(ctx, `SELECT ts,kind,amount FROM fin_cashflow
		WHERE kind NOT IN ('earn_in','earn_out') ORDER BY id DESC LIMIT 6`)
	for rrows.Next() {
		var ts time.Time
		var kind string
		var amt float64
		_ = rrows.Scan(&ts, &kind, &amt)
		verb := map[string]string{"funding_settle": "资金费到账", "fee": "交易成本",
			"trade_pnl": "平仓盈亏", "deposit": "入金", "withdraw": "出金",
			"interest": "活期利息", "recon_adjust": "校账调整",
			"spot_buy": "现货买入", "spot_sell": "现货卖出"}[kind]
		o.Recent = append(o.Recent, fmt.Sprintf("%s %s %+.4f USDT",
			ts.Format("15:04"), verb, amt))
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

// Watch SSE 脊推变（ev_* NOTIFY——所见即所得刷新源，事件驱动律守不新造环）。
func (c coreService) Watch(ctx context.Context, _ *dashboardv1.WatchRequest,
	stream dashboardv1connect.CoreServiceWatchServerStream) error {
	conn, err := c.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	for _, ch := range []string{"ev_position", "ev_cashflow", "ev_account", "ev_strategy"} {
		if _, err := conn.Exec(ctx, "LISTEN "+ch); err != nil {
			return err
		}
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if err := stream.Send(&dashboardv1.WatchEvent{
			Kind: n.Channel, Ts: time.Now().UTC().Format(time.RFC3339)}); err != nil {
			return err
		}
	}
}

func durHuman(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// mountCore 挂 ConnectRPC 服务到 mux（connect-go v2：Server 注册 + connecthttp.Mount）。
func mountCore(mux *http.ServeMux, pool *pgxpool.Pool,
	fundingLive func(context.Context) (map[string]float64, error),
	earnLive func(context.Context) (map[string]float64, error),
	mt5Live func(context.Context) ([]mt5Acct, error)) {
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server,
		coreService{pool: pool, fundingLive: fundingLive, earnLive: earnLive, mt5Live: mt5Live})
	connecthttp.Mount(mux, server)
}

// mountUI 总览活页面挂接（静态 embed+JSON/SSE——所见即所得零缓存；活推=ev_* 脊）。
func mountUI(mux *http.ServeMux, pool *pgxpool.Pool,
	fundingLive func(context.Context) (map[string]float64, error),
	earnLive func(context.Context) (map[string]float64, error),
	mt5Live func(context.Context) ([]mt5Acct, error)) {
	mux.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if o, err := (coreService{pool: pool, fundingLive: fundingLive,
			earnLive: earnLive, mt5Live: mt5Live}).Overview(r.Context(), nil); err == nil {
			b, _ := json.Marshal(o)
			_, _ = w.Write(b)
		}
	})
	mux.HandleFunc("/api/watch", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		conn, err := pool.Acquire(r.Context())
		if err != nil {
			return
		}
		defer conn.Release()
		if _, err := conn.Exec(r.Context(), "LISTEN ev_position"); err != nil {
			return
		}
		_, _ = conn.Exec(r.Context(), "LISTEN ev_cashflow")
		_, _ = conn.Exec(r.Context(), "LISTEN ev_account")
		for {
			if _, err := conn.Conn().WaitForNotification(r.Context()); err != nil {
				return
			}
			_, _ = fmt.Fprint(w, "data: tick\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	})
	// 根路径直开（T-0068：GET / → index.html 直返无跳转；/ui/ 目录列表禁——
	// dist 单层文件面逐件挂，禁 FileServer 目录模式）。
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" { // "/" 是兜底全捕——仅根路径返页，API 错路 404 不吃 HTML
			http.NotFound(w, r)
			return
		}
		b, err := web.Dist.ReadFile("dist/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(b)
	})
	// assets 子树挂接（fs.Sub——embed FS 全件带 dist/ 前缀，指根=资产 404 病灶）。
	assets, _ := fs.Sub(web.Dist, "dist")
	mux.Handle("/assets/", http.FileServer(http.FS(assets)))
}
