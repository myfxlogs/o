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

	"phase/internal/gen/dashboardv1"
	"phase/internal/gen/dashboardv1/dashboardv1connect"
	"phase/web"
)

type coreService struct{ pool *pgxpool.Pool }

func (coreService) Ping(context.Context, *dashboardv1.PingRequest) (*dashboardv1.PingResponse, error) {
	return &dashboardv1.PingResponse{Status: "ok"}, nil
}

// Overview 总览四联活读（所见即所得——零快照零缓存；数据出生证=查询时戳）。
func (c coreService) Overview(ctx context.Context, _ *dashboardv1.OverviewRequest) (*dashboardv1.OverviewResponse, error) {
	o := &dashboardv1.OverviewResponse{Ts: time.Now().UTC().Format(time.RFC3339)}
	// T-0072+0073 口径正名：nav=venue 帧总权益（交易户 details 全币 eqUsd
	// 求和）+资金户/Earn 余额——全账户合计非交易户窄口径；deposit 本金不计资产。
	var frame json.RawMessage
	if err := c.pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel IN ('account','balance_and_position') ORDER BY ts DESC LIMIT 1`).Scan(&frame); err == nil {
		var top struct {
			EqUsd   string `json:"eqUsd"`
			Details []struct {
				EqUsd string `json:"eqUsd"`
			} `json:"details"`
		}
		if json.Unmarshal(frame, &top) == nil {
			if len(top.Details) > 0 {
				for _, d := range top.Details {
					v, _ := strconv.ParseFloat(d.EqUsd, 64)
					o.NavTotal += v
				}
			} else {
				o.NavTotal, _ = strconv.ParseFloat(top.EqUsd, 64)
			}
		}
	}
	// 资金户+Earn 面（T-0073：earn_parking 帧 earn_bal+funding 帧 top 值——
	// 缺帧=0 不误报，与「钱在哪」卡同源自洽）。
	var earnRaw json.RawMessage
	if err := c.pool.QueryRow(ctx, `SELECT data FROM md_account_events
		WHERE channel='earn_parking' ORDER BY ts DESC LIMIT 1`).Scan(&earnRaw); err == nil {
		var ev map[string]json.RawMessage
		if json.Unmarshal(earnRaw, &ev) == nil {
			if v, err := strconv.ParseFloat(strings.Trim(string(ev["earn_bal"]), `"`), 64); err == nil {
				o.NavTotal += v
			}
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
	mrows, _ := c.pool.Query(ctx, `SELECT (data->>'earn_bal')::float FROM md_account_events
		WHERE channel='earn_parking' ORDER BY ts DESC LIMIT 1`)
	for mrows.Next() {
		var v float64
		_ = mrows.Scan(&v)
		o.Money = append(o.Money, &dashboardv1.MoneyRow{Label: "活期理财 Earn", Amount: v})
	}
	mrows.Close()
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
func mountCore(mux *http.ServeMux, pool *pgxpool.Pool) {
	server := connect.NewServer()
	dashboardv1connect.RegisterCoreServiceHandler(server, coreService{pool: pool})
	connecthttp.Mount(mux, server)
}

// mountUI 总览活页面挂接（静态 embed+JSON/SSE——所见即所得零缓存；活推=ev_* 脊）。
func mountUI(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("/api/overview", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if o, err := (coreService{pool: pool}).Overview(r.Context(), nil); err == nil {
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
