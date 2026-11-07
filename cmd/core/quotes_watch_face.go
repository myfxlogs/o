// quotes_watch_face.go —— T-0391 自选报价条读/写面（NOX QuoteStrip/
// WatchlistModal 语义移植——业主令 2026-11-06：「订阅某1个symbol的实时
// 报价，这个symbol需要我来定义，要支持搜索」+ T-0442 件③ 源约束（业主令
// 2026-11-07：「Nox项目中自选报价条编辑可以选择订阅提供报价的broker/cex，
// 引进过来」）：
//
//	· GetQuoteWatchlist：fin_quote_watchlist 成员（ord 保存序——排序全
//	  后端禁前端 sort）×md_ticks 最新行携价+鲜度龄服务端派生；T-0442 起成员
//	  携 venues 源约束集（空=全部源=最新跨源——现语义零破坏），取价 LATERAL
//	  逐员过滤+quote_venue 来源实证面；可选源集 src_sets=md_ticks 实采∪
//	  md_tick_scan 实扫派生（零硬编码词表——0440 通用性律：新 venue 接入
//	  零改本面；md_ticks 无 inst_type 列——Nox venue:venueType 双维单维落
//	  地）；候选域=活符∪目录符+pending 未订阅标（T-0412）；零成员/零行情
//	  如实缺态不假零；
//	· SetQuoteWatchlist：同源闸窄写（earnOriginGate 同规）→归一去重→
//	  逐员选符域校验（活符收/目录符收+待晋升/域外拒——T-0412 两域闸）+
//	  源约束闸（venue 须该符实采∪实扫集内——chip 候选同谓词单源）→
//	  单事务全量替换+源感知晋升提名（DELETE+INSERT ord=数组序+user_intent
//	  行同生共死——约束集内「谱上未采」源逐源提名，空约束沿旧单源径）→
//	  ops_event 审计行 seat=dashboard-ui →真值重读返回（回显禁）；
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

// watchRowSQL 成员行×最新价投影（T-0442 件③——LATERAL 逐员约束过滤代旧
// DISTINCT ON：约束集非空=只取所选 venue 源最新 tick，空=全部源最新跨源
// （旧语义零破坏）；quote_venue=所显价来源实证面；pending=源感知晋升中标
// ——约束集内源在谱（scan）待采时标「晋升中」，谱上皆无=「暂无行情」诚实显缺）。
const watchRowSQL = `SELECT w.symbol,w.venues,COALESCE(t.venue,''),
	t.last IS NOT NULL,t.last::float8,t.bid::float8,t.ask::float8,t.recv_ts,
	COALESCE(EXTRACT(EPOCH FROM now()-t.recv_ts)::bigint,0),
	t.last IS NULL AND EXISTS (SELECT 1 FROM md_tick_scan s
		WHERE s.inst_id=w.symbol AND ` + scanFreshSQL + `
		AND (cardinality(w.venues)=0 OR s.venue=ANY(w.venues)))
	FROM fin_quote_watchlist w
	LEFT JOIN LATERAL (SELECT venue,last,bid,ask,recv_ts FROM md_ticks
		WHERE inst_id=w.symbol
			AND (cardinality(w.venues)=0 OR venue=ANY(w.venues))
		ORDER BY ts DESC LIMIT 1) t ON true
	ORDER BY w.ord`

func scanWatchRow(s interface{ Scan(...any) error }) (*dashboardv1.QuoteWatchRow, error) {
	var r dashboardv1.QuoteWatchRow
	var venues []string
	var tickTs *time.Time
	var last, bid, ask *float64
	var ageS int64
	if err := s.Scan(&r.Symbol, &venues, &r.QuoteVenue, &r.HasQuote,
		&last, &bid, &ask, &tickTs, &ageS, &r.Pending); err != nil {
		return nil, err
	}
	r.Venues = venues
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

// srcSetsSQL 可选源集派生（T-0442 件③——chip 候选=实采∪实扫同谓词单源，
// 与选符域 candDomainSQL 同两域口径；零硬编码词表——md_ticks 实值即词表）。
const srcSetsSQL = `SELECT inst_id,array_agg(DISTINCT venue ORDER BY venue) FROM (
	SELECT inst_id,venue FROM md_ticks WHERE ` + tickFreshSQL + `
	UNION ALL SELECT inst_id,venue FROM md_tick_scan WHERE ` + scanFreshSQL + `
	) c GROUP BY inst_id ORDER BY inst_id`

// symSourceVenues 单符可选源集（Set 源约束闸判据——chip 候选同谓词单源）。
// 返 live=实采集（md_ticks 鲜界内）/scan=实扫集（md_tick_scan 鲜界内）。
func (c coreService) symSourceVenues(ctx context.Context, canon string) (live, scan map[string]bool, err error) {
	rows, err := c.pool.Query(ctx, `SELECT venue,true FROM md_ticks
		WHERE inst_id=$1 AND `+tickFreshSQL+`
		UNION SELECT venue,false FROM md_tick_scan
		WHERE inst_id=$1 AND `+scanFreshSQL, canon)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	live, scan = map[string]bool{}, map[string]bool{}
	for rows.Next() {
		var v string
		var l bool
		if err := rows.Scan(&v, &l); err != nil {
			return nil, nil, err
		}
		if l {
			live[v] = true
		} else {
			scan[v] = true
		}
	}
	return live, scan, rows.Err()
}

// GetQuoteWatchlist 报价条成员×最新价+候选域+可选源集（读径合并窗内——
// ev_ticks 帧重取；零成员 rows 空如实缺态，候选域=活符∪目录符携 pending 标
// ——T-0412）。
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
	// 可选源集（T-0442——chip 候选=实采∪实扫派生，成员/候选两处同源消费）
	srows, err := c.pool.Query(ctx, srcSetsSQL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer srows.Close()
	for srows.Next() {
		var sym string
		var venues []string
		if err := srows.Scan(&sym, &venues); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		res.SrcSets = append(res.SrcSets, &dashboardv1.QuoteSrcSet{Symbol: sym, Venues: venues})
	}
	if err := srows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	return res, nil
}

// SetQuoteWatchlist 全量替换成员集（同源闸→归一/去重/在采校验+源约束闸→
// 单事务删旧插新 ord=数组序+源感知晋升提名；写后真值重读；审计行
// seat=dashboard-ui 同建行式）。
func (c coreService) SetQuoteWatchlist(ctx context.Context,
	req *dashboardv1.SetQuoteWatchlistRequest) (*dashboardv1.SetQuoteWatchlistResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	type member struct {
		symbol string
		venues []string
	}
	var syms []member
	seen := map[string]bool{}
	for _, raw := range req.GetItems() {
		s := strings.ToUpper(strings.TrimSpace(raw.GetSymbol()))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		m := member{symbol: s}
		vseen := map[string]bool{}
		for _, v := range raw.GetVenues() {
			v = strings.TrimSpace(v)
			if v == "" || vseen[v] {
				continue
			}
			if strings.Contains(v, ":") { // 裸值禁冒号（D-496 同规——迁移 CHECK 同闸前置）
				return nil, connect.NewError(connect.CodeInvalidArgument,
					fmt.Sprintf("源「%s」须为裸 venue 值", v))
			}
			vseen[v] = true
			m.venues = append(m.venues, v)
		}
		if m.venues == nil {
			m.venues = []string{} // pgx nil 切片=NULL——NOT NULL 列需空集显式
		}
		syms = append(syms, m)
	}
	// T-0412 两域闸同口径（T-0399 改判——与 CreatePriceAlert 同谓词单源）：
	// 逐员活符=收；目录符=收+待晋升；域外=拒。canon 原键落库——mt5 大小写
	// 方言不打坏。T-0442 源约束闸：venue 须该符实采∪实扫集内（chip 候选同
	// 谓词单源——选不到的源不收）。
	promotes := map[string]string{} // canon→venue（空约束目录符待晋升集——旧单源径）
	var venPromotes [][2]string     // (venue,canon)（约束集内谱上未采源——逐源提名）
	for i := range syms {
		m := &syms[i]
		canon, venue, pending, err := c.symDomain(ctx, m.symbol)
		if errors.Is(err, errNoDomain) {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Sprintf("「%s」不在行情目录——自选面收在采/目录内品种", m.symbol))
		}
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		m.symbol = canon
		if len(m.venues) == 0 {
			if pending {
				promotes[canon] = venue
			}
			continue
		}
		live, scan, err := c.symSourceVenues(ctx, canon)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		for _, v := range m.venues {
			if !live[v] && !scan[v] {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					fmt.Sprintf("「%s」无源「%s」——可选源=实采∪实扫派生", canon, v))
			}
			if !live[v] && scan[v] { // 谱上未采——源感知晋升提名（约束内逐源）
				venPromotes = append(venPromotes, [2]string{v, canon})
			}
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
	for i := range syms {
		if _, err := tx.Exec(ctx, `INSERT INTO fin_quote_watchlist(symbol,ord,venues,created_by)
			VALUES($1,$2,$3,$4)`, syms[i].symbol, i, syms[i].venues, alertSeatDash); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	for canon, venue := range promotes { // 目录符提名晋升——与成员行同事务原子落
		if _, err := tx.Exec(ctx, promoteSQL, venue, canon, "quote_watchlist"); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	for _, vp := range venPromotes {
		if _, err := tx.Exec(ctx, promoteSQL, vp[0], vp[1], "quote_watchlist"); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	// 审计载荷携源约束摘要（SYM(源,…)、空约束=SYM(全部源)）
	parts := make([]string, 0, len(syms))
	for _, m := range syms {
		if len(m.venues) == 0 {
			parts = append(parts, m.symbol+"(全部源)")
		} else {
			parts = append(parts, fmt.Sprintf("%s(%s)", m.symbol, strings.Join(m.venues, ",")))
		}
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, alertSeatDash, watchEventSet,
		fmt.Sprintf("自选报价条全量替换 %d 员：%s", len(syms), strings.Join(parts, ","))); err != nil {
		log.Printf("[quotes-face] 自选审计行落败（集已替——痕缺）: %v", err)
	}
	truth, err := c.watchSymbols(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	return &dashboardv1.SetQuoteWatchlistResponse{Symbols: truth}, nil
}
