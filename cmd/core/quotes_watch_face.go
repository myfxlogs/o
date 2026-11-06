// quotes_watch_face.go —— T-0391 自选报价条读/写面（NOX QuoteStrip/
// WatchlistModal 语义移植——业主令 2026-11-06：「订阅某1个symbol的实时
// 报价，这个symbol需要我来定义，要支持搜索」）：
//
//	· GetQuoteWatchlist：fin_quote_watchlist 成员（ord 保存序——排序全
//	  后端禁前端 sort）LEFT JOIN md_ticks 最新行携价+鲜度龄服务端派生；
//	  候选域=活符∪目录符+pending 未订阅标（T-0412——NOX ListLatestQuotes
//	  义扩到粗扫谱）；零成员/零行情如实缺态不假零；
//	· SetQuoteWatchlist：同源闸窄写（earnOriginGate 同规）→归一去重→
//	  逐员选符域校验（活符收/目录符收+待晋升/域外拒——T-0412 两域闸）
//	  →单事务全量替换+目录符晋升提名（DELETE+INSERT ord=数组序+user_intent
//	  行同生共死）→ops_event 审计行 seat=dashboard-ui
//	  →真值重读返回（回显禁）；
//	· 红线：纯展示面——不喂判定/策略/采集域零动。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	connect "connectrpc.com/connect/v2"

	"o/internal/gen/dashboardv1"
)

const watchEventSet = "quote_watchlist_set"

// watchRowSQL 成员行×最新价投影（alertRowSQL 同构——DISTINCT ON 每 symbol
// 最新 tick；成员序=fin_quote_watchlist.ord 保存序全后端零前端 sort）。
const watchRowSQL = `SELECT w.symbol,
	t.last IS NOT NULL,t.last::float8,t.bid::float8,t.ask::float8,t.recv_ts,
	COALESCE(EXTRACT(EPOCH FROM now()-t.recv_ts)::bigint,0)
	FROM fin_quote_watchlist w
	LEFT JOIN (SELECT DISTINCT ON (inst_id) inst_id,last,bid,ask,recv_ts FROM md_ticks
		ORDER BY inst_id,ts DESC) t ON t.inst_id=w.symbol
	ORDER BY w.ord`

func scanWatchRow(s interface{ Scan(...any) error }) (*dashboardv1.QuoteWatchRow, error) {
	var r dashboardv1.QuoteWatchRow
	var tickTs *time.Time
	var last, bid, ask *float64
	var ageS int64
	if err := s.Scan(&r.Symbol, &r.HasQuote, &last, &bid, &ask, &tickTs, &ageS); err != nil {
		return nil, err
	}
	if last != nil {
		r.Last, r.Bid, r.Ask = *last, *bid, *ask
	}
	r.QuoteTs = tsProtoPtr(tickTs)
	r.QuoteAgeS = ageS
	return &r, nil
}

// watchSymbols 成员序真值（Set 写后重读——回显禁）。
func (c coreService) watchSymbols(ctx context.Context) ([]string, error) {
	rows, err := c.pool.Query(ctx, `SELECT symbol FROM fin_quote_watchlist ORDER BY ord`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetQuoteWatchlist 报价条成员×最新价+候选域（读径合并窗内——ev_ticks 帧重取；
// 零成员 rows 空如实缺态，候选域=活符∪目录符携 pending 标——T-0412）。
func (c coreService) GetQuoteWatchlist(ctx context.Context,
	_ *dashboardv1.GetQuoteWatchlistRequest) (*dashboardv1.GetQuoteWatchlistResponse, error) {
	res := &dashboardv1.GetQuoteWatchlistResponse{AsOf: tsProto(time.Now())}
	rows, err := c.pool.Query(ctx, watchRowSQL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanWatchRow(rows)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		res.Rows = append(res.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	// 候选域=活符∪目录符（T-0412——与 ListPriceAlerts 同谓词单源；目录符
	// 携 pending 未订阅标——选入即晋升提名）
	syms, pending, err := c.candSets(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	res.Symbols, res.PendingSymbols = syms, pending
	return res, nil
}

// SetQuoteWatchlist 全量替换成员集（同源闸→归一/去重/在采校验→单事务删旧
// 插新 ord=数组序；写后真值重读；审计行 seat=dashboard-ui 同建行式）。
func (c coreService) SetQuoteWatchlist(ctx context.Context,
	req *dashboardv1.SetQuoteWatchlistRequest) (*dashboardv1.SetQuoteWatchlistResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	var syms []string
	seen := map[string]bool{}
	for _, raw := range req.GetSymbols() {
		s := strings.ToUpper(strings.TrimSpace(raw))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		syms = append(syms, s)
	}
	// T-0412 两域闸同口径（T-0399 改判——与 CreatePriceAlert 同谓词单源）：
	// 逐员活符=收；目录符=收+待晋升（同事务提名 user_intent）；域外=拒。
	// canon 原键落库——mt5 大小写方言不打坏。
	promotes := map[string]string{} // canon→venue（目录符待晋升集）
	for i, s := range syms {
		canon, venue, pending, err := c.symDomain(ctx, s)
		if errors.Is(err, errNoDomain) {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Sprintf("「%s」不在行情目录——自选面收在采/目录内品种", s))
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		syms[i] = canon
		if pending {
			promotes[canon] = venue
		}
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM fin_quote_watchlist`); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	for i, s := range syms {
		if _, err := tx.Exec(ctx, `INSERT INTO fin_quote_watchlist(symbol,ord,created_by)
			VALUES($1,$2,$3)`, s, i, alertSeatDash); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	for canon, venue := range promotes { // 目录符提名晋升——与成员行同事务原子落
		if _, err := tx.Exec(ctx, promoteSQL, venue, canon, "quote_watchlist"); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, alertSeatDash, watchEventSet,
		fmt.Sprintf("自选报价条全量替换 %d 员：%s", len(syms), strings.Join(syms, ","))); err != nil {
		log.Printf("[quotes-face] 自选审计行落败（集已替——痕缺）: %v", err)
	}
	truth, err := c.watchSymbols(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	return &dashboardv1.SetQuoteWatchlistResponse{Symbols: truth}, nil
}
