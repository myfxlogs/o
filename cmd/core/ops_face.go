// ops_face.go —— 系统健康+复盘读面（T-0170——/md/status、/report 手集 JSON
// 收编入 CoreService 契约，D-0009：有 proto 的面=绝对 proto）。判据全在
// 服务端算定（节拍阈值逐件钦定），前端零判据零计算——所见即所判。
package main

import (
	"context"
	"encoding/json"
	"log"
	"sort"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// compBeat 组件节拍判据：黄/红阈（秒级面 30s/2min 沿 venues.go 同律；事件型
// 面按其拍节钦定——mt5_swap=4h 拍、earn 现率时级、dcd 枚举低频）。
type compBeat struct {
	name, fact, query string
	warn, dead        time.Duration
	zeroState         int32  // 零行态（语义空态组件=0 绿非红）
	zeroNote          string // 零行旁注（空行≠死的面在此显形）
}

// healthComps 采集组件全集（16 件——足迹表+channel 谓词逐件钉死；增组件=增行）。
func healthComps() []compBeat {
	return []compBeat{
		{name: "okxws WS 行情", fact: "md_mark_index mark-price|index-tickers", warn: 30 * time.Second, dead: 2 * time.Minute,
			query: `SELECT max(recv_ts) FROM md_mark_index WHERE channel IN ('mark-price','index-tickers')`},
		{name: "spotpoll REST(spot_last)", fact: "md_mark_index spot_last", warn: 90 * time.Second, dead: 5 * time.Minute,
			query: `SELECT max(recv_ts) FROM md_mark_index WHERE channel='spot_last'`},
		{name: "spotpoll REST(mark_price)", fact: "md_mark_index mark_price", warn: 90 * time.Second, dead: 5 * time.Minute,
			query: `SELECT max(recv_ts) FROM md_mark_index WHERE channel='mark_price'`},
		{name: "funding WS", fact: "md_funding", warn: 30 * time.Minute, dead: 2 * time.Hour,
			query: `SELECT max(recv_ts) FROM md_funding`},
		{name: "买五档 depth5", fact: "md_signal kind=depth5", warn: 30 * time.Second, dead: 2 * time.Minute,
			query: `SELECT max(ts) FROM md_signal WHERE kind='depth5'`},
		{name: "蒸馏 basis_ann", fact: "md_signal kind=basis_ann", warn: 10 * time.Minute, dead: 30 * time.Minute,
			query:     `SELECT max(ts) FROM md_signal WHERE kind='basis_ann'`,
			zeroState: 1, zeroNote: "零行——已知缺席（T-0164 立案另查，非采集死）"},
		{name: "蒸馏 funding_view", fact: "md_signal kind=funding_view", warn: 10 * time.Minute, dead: 30 * time.Minute,
			query: `SELECT max(ts) FROM md_signal WHERE kind='funding_view'`},
		{name: "蒸馏 xvenue_spread", fact: "md_signal kind=xvenue_spread", warn: 10 * time.Minute, dead: 30 * time.Minute,
			query: `SELECT max(ts) FROM md_signal WHERE kind='xvenue_spread'`},
		{name: "蒸馏 carry_diff", fact: "md_signal kind=carry_diff", warn: 5 * time.Hour, dead: 9 * time.Hour,
			query:     `SELECT max(ts) FROM md_signal WHERE kind='carry_diff'`,
			zeroState: 0, zeroNote: "随 mt5_swap 4h 拍刷新——拍间陈为设计语义"},
		{name: "蒸馏 mt5_swap", fact: "md_signal kind=mt5_swap", warn: 5 * time.Hour, dead: 9 * time.Hour,
			query:     `SELECT max(ts) FROM md_signal WHERE kind='mt5_swap'`,
			zeroState: 0, zeroNote: "4h 慢拍设计（SymbolParams 日级变量）"},
		{name: "MT5 桥行情", fact: "md_ticks venue=mt5", warn: 30 * time.Second, dead: 2 * time.Minute,
			query: `SELECT max(ts) FROM md_ticks WHERE venue='mt5'`},
		{name: "MT5 利润流", fact: "md_account_events order_profit", warn: 5 * time.Minute, dead: 30 * time.Minute,
			query:     `SELECT max(ts) FROM md_account_events WHERE channel='order_profit'`,
			zeroState: 0, zeroNote: "空仓语义——无 MT5 持仓即无利润流（活性判据=MT5 行情行+汇市巡）"},
		{name: "OKX 私有 WS", fact: "md_account_events account|balance_and_position", warn: 30 * time.Second, dead: 2 * time.Minute,
			query: `SELECT max(ts) FROM md_account_events WHERE channel IN ('account','balance_and_position')`},
		{name: "dcd 枚举", fact: "md_dcd_product", warn: 2 * time.Hour, dead: 12 * time.Hour,
			query: `SELECT max(ts) FROM md_dcd_product`},
		{name: "earn 现率采样", fact: "md_earn_rate", warn: 2 * time.Hour, dead: 6 * time.Hour,
			query: `SELECT max(ts) FROM md_earn_rate`},
		{name: "行情 ticks(OKX)", fact: "md_ticks venue<>mt5", warn: 30 * time.Second, dead: 2 * time.Minute,
			query: `SELECT max(ts) FROM md_ticks WHERE coalesce(venue,'okx')<>'mt5'`},
	}
}

// GetSystemHealth 采集健康全集：逐件足迹新鲜度（服务端判态）+进程内指标
// （帧计数/最近帧/拒因/带宽——原 /md/status 三件收编）。
func (c coreService) GetSystemHealth(ctx context.Context,
	_ *dashboardv1.GetSystemHealthRequest) (*dashboardv1.GetSystemHealthResponse, error) {
	r := &dashboardv1.GetSystemHealthResponse{}
	for _, b := range healthComps() {
		comp := &dashboardv1.SystemComponent{Component: b.name, Fact: b.fact,
			State: 0, Note: ""}
		var ts *time.Time
		if err := c.pool.QueryRow(ctx, b.query).Scan(&ts); err != nil {
			comp.State, comp.Note = 2, "查询败（读件缺）"
		} else if ts == nil {
			if b.zeroNote != "" { // 语义空态面（空行≠死）——逐件钦定注
				comp.State, comp.Note = b.zeroState, b.zeroNote
			} else { // 常规面零行=读件缺（禁假绿）
				comp.State, comp.Note = 2, "零行——读件缺"
			}
		} else {
			comp.LastWrite = tsProto(*ts)
			switch d := time.Since(*ts); {
			case d > b.dead:
				comp.State = 2
			case d > b.warn:
				comp.State = 1
			}
		}
		r.Components = append(r.Components, comp)
	}
	if c.md != nil {
		r.Frames = c.md.frames()
		r.Rejects = map[string]int64{}
		for k, v := range c.md.rejects() {
			r.Rejects[k] = v
		}
		r.NetKbps = c.md.netKbps()
		r.NetChannels = map[string]int64{}
		for k, v := range c.md.netChannels() {
			r.NetChannels[k] = v
		}
		if t := c.md.lastFrame(); !t.IsZero() {
			r.LastFrame = tsProto(t)
		}
	}
	return r, nil
}

// ListReports fin_report 直出（metrics jsonb 服务端解键值对——键序稳定输出）。
func (c coreService) ListReports(ctx context.Context,
	req *dashboardv1.ListReportsRequest) (*dashboardv1.ListReportsResponse, error) {
	period := req.GetPeriod()
	if period == "" {
		period = "daily"
	}
	n := int(req.GetN())
	if n <= 0 || n > 90 {
		n = 8
	}
	rows, err := c.pool.Query(ctx, `SELECT period,period_start,period_end,metrics,note,generated_at
		FROM fin_report WHERE period=$1 ORDER BY period_start DESC LIMIT $2`, period, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := &dashboardv1.ListReportsResponse{}
	for rows.Next() {
		var p, note string
		var s, e, g time.Time
		var raw json.RawMessage
		if err := rows.Scan(&p, &s, &e, &raw, &note, &g); err != nil {
			return nil, err
		}
		row := &dashboardv1.ReportRow{Period: p, Note: note,
			PeriodStart: tsProto(s), PeriodEnd: tsProto(e), GeneratedAt: tsProto(g)}
		var m map[string]float64
		if err := json.Unmarshal(raw, &m); err != nil {
			log.Printf("[ops] report %s metrics 解码败: %v", p, err)
		}
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			row.Metrics = append(row.Metrics, &dashboardv1.ReportMetric{K: k, V: m[k]})
		}
		r.Reports = append(r.Reports, row)
	}
	return r, rows.Err()
}

// GetEquityCurve NAV 日桶曲线（md_equity_curve ccy='NAV' 行——report 日拍供数；
// last(equity,ts)=timescale 末值聚，当日多拍（重启补拍）取末）。
func (c coreService) GetEquityCurve(ctx context.Context,
	req *dashboardv1.GetEquityCurveRequest) (*dashboardv1.GetEquityCurveResponse, error) {
	days := int(req.GetDays())
	if days <= 0 {
		days = 90
	}
	rows, err := c.pool.Query(ctx, `SELECT time_bucket('1 day',ts) AS day, last(equity,ts)
		FROM md_equity_curve WHERE ccy='NAV' AND ts>now()-make_interval(days=>$1)
		GROUP BY 1 ORDER BY 1`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := &dashboardv1.GetEquityCurveResponse{}
	for rows.Next() {
		var day time.Time
		var eq float64
		if err := rows.Scan(&day, &eq); err != nil {
			return nil, err
		}
		r.Points = append(r.Points, &dashboardv1.EquityPoint{Day: tsProto(day), Nav: eq})
	}
	return r, rows.Err()
}

// ListCashflows 现金流全量下钻（kind 筛+分页下推——复盘面供数）。
func (c coreService) ListCashflows(ctx context.Context,
	req *dashboardv1.ListCashflowsRequest) (*dashboardv1.ListCashflowsResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}
	kind := req.GetKind()
	r := &dashboardv1.ListCashflowsResponse{}
	var total int32
	q := `SELECT count(*) FROM fin_cashflow`
	args := []any{}
	if kind != "" {
		q += ` WHERE kind=$1`
		args = append(args, kind)
	}
	if err := c.pool.QueryRow(ctx, q, args...).Scan(&total); err != nil {
		return nil, err
	}
	r.Total = total
	q = `SELECT id,occurred_at,kind,ccy,amount,coalesce(venue,'') FROM fin_cashflow`
	if kind != "" {
		q += ` WHERE kind=$1`
	}
	q += ` ORDER BY id DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, int(req.GetOffset()))
	rows, err := c.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var at time.Time
		var kind, ccy, venue string
		var amt float64
		if err := rows.Scan(&id, &at, &kind, &ccy, &amt, &venue); err != nil {
			return nil, err
		}
		r.Rows = append(r.Rows, &dashboardv1.CashflowRow{Id: id,
			OccurredAt: tsProto(at), Kind: kind, Ccy: ccy, Amount: amt, Venue: venue})
	}
	return r, rows.Err()
}

func tsProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t.UTC())
}
