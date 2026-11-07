// quotes_face.go —— T-0382 报价点+价格提醒读/写面（业主令：「想看到
// BTCUSDT/TRXUSDT 实时报价，并想预设一个价格，飞书通知我」）：
//
//	· ListPriceAlerts：fin_price_alert 行 LEFT JOIN md_ticks 最新价（报
//	  价卡 symbol 集=提醒行集——票面「一单两用」语义）+选符域枚举供
//	  表单下拉（T-0412=活符∪目录符+pending 未订阅标）；行级鲜度龄服
//	  务端派生（前端零判据）；
//	· CreatePriceAlert/SetPriceAlertState：同源闸窄写（earnOriginGate
//	  同规——Origin 缺席/跨源=拒）；写后真值重读非回显；T-0412 两域
//	  闸——目录符收+同事务晋升提名（user_intent→反向桥→collector
//	  拾取流式采，僵尸提醒病灶根修）；
//	· 红线：单向告警非条件单——写径校验 symbol 选符域内/方向闭集/
//	  target>0，fired 终态禁手翻（一发即锁语义）；UI 操作落 ops_event
//	  审计行（seat=dashboard-ui 同 earn 闸先例）。
package main

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	connect "connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	"o/internal/gen/dashboardv1"
)

// tsProtoPtr 可空时间戳（NULL=nil proto 缺席非零值年1假象——tsProto 旁生）。
func tsProtoPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return tsProto(*t)
}

// tickFreshSQL 提醒径鲜度谓词（T-0399——freshByKind 族「源节拍自适」同构
// 不新造档；判据=末 tick recv_ts 落库时刻非 ts 戳，防时钟偏移假象——
// alertRowSQL 鲜度龄同源）：快源钦定 15min 界（对齐动态数据闸口径）；
// mt5 流推源汇市周末 ~49h 休市静默非死源——界=72h（周末窗+边，15min
// 一刀切误伤防=票面红线）。扩 venue 源按其节拍补 CASE 不另开闸件。
const tickFreshSQL = `recv_ts > now()-CASE venue
	WHEN 'mt5' THEN interval '72 hours'
	ELSE interval '15 minutes' END`

// scanFreshSQL 粗扫目录鲜界（T-0412——REST tickers 60s 拍 1h 宽限；休拍即
// 目录收缩，防陈目录放进僵尸提名——0394 scan 活性探针 10min 界先升件，
// 本界再收编安全边）。
const scanFreshSQL = `recv_ts > now()-interval '1 hour'`

// candDomainSQL 选符域+未订阅标单查（T-0412 钦定域=md_ticks 活符 ∪
// md_tick_scan 目录活符——有行情的+被目录扫到的都可选；bool_or 归并
// 双域命中符算活非 pending）。ListPriceAlerts/GetQuoteWatchlist 候选
// 域同谓词单源——死符/域外符不可选。
const candDomainSQL = `SELECT inst_id,NOT bool_or(live) FROM (
	SELECT inst_id,true AS live FROM md_ticks WHERE ` + tickFreshSQL + `
	UNION ALL SELECT inst_id,false FROM md_tick_scan WHERE ` + scanFreshSQL + `
	) c GROUP BY inst_id ORDER BY inst_id`

// errNoDomain 选符域外判词（上层翻可读文案——typo 闸不放松）。
var errNoDomain = errors.New("选符域外")

// symDomain 单符选符域判词（T-0412 两域闸语义——T-0399 死符拒建改判：
// 在采活符=收（pending=false）；目录活符=收+待晋升（pending=true，
// 同事务插 fin_sub_request user_intent open——SG 正本经 ev_subreq 帧
// 反向桥 PG-md，collector SubRequestArm UNION 拾取流式采）；两域外=
// errNoDomain 拒）。canon 落库 inst_id 原键——mt5 大小写方言不打坏。
func (c coreService) symDomain(ctx context.Context, sym string) (canon, venue string, pending bool, err error) {
	if err = c.pool.QueryRow(ctx, `SELECT inst_id,venue FROM md_ticks
		WHERE upper(inst_id)=$1 AND `+tickFreshSQL+` LIMIT 1`, sym).Scan(&canon, &venue); err == nil {
		return canon, venue, false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, err
	}
	if err = c.pool.QueryRow(ctx, `SELECT inst_id,venue FROM md_tick_scan
		WHERE upper(inst_id)=$1 AND `+scanFreshSQL+` LIMIT 1`, sym).Scan(&canon, &venue); err == nil {
		return canon, venue, true, nil
	} else if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, errNoDomain
	}
	return "", "", false, err
}

// promoteSQL 用户意图晋升行（T-0412——SG 正本写：建单同事务插；
// requester='user_intent' 常驻意图——reconcileSubs 扫尾豁免不闭请，
// 正向投影排除免正本互覆）。reason=建单面溯源（price_alert#N/quote_watchlist）。
const promoteSQL = `INSERT INTO fin_sub_request(venue,inst_id,requester,state,reason)
	VALUES($1,$2,'user_intent','open',$3)
	ON CONFLICT (venue,inst_id,requester) DO UPDATE SET state='open',
		reason=EXCLUDED.reason,ts=now()`

// candSets 选符域枚举+未订阅标（List/Get 同径——一单两列免双查）。
func (c coreService) candSets(ctx context.Context) (symbols, pending []string, err error) {
	rows, err := c.pool.Query(ctx, candDomainSQL)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var p bool
		if err := rows.Scan(&s, &p); err != nil {
			return nil, nil, err
		}
		symbols = append(symbols, s)
		if p {
			pending = append(pending, s)
		}
	}
	return symbols, pending, rows.Err()
}

// alertRowSQL 提醒行×最新价投影（建/翻后真值重读与列表同口径单源）：
// DISTINCT ON 取每 symbol 最新 tick（venue 扩面防重行）；无 tick 行=
// has_quote=false 显缺不假零；鲜度龄=now()-recv_ts（采集臂停滞即显陈——
// 用落库时刻非交易所 ts 防时钟偏移假象）服务端派生秒（前端零判据）。
const alertRowSQL = `SELECT a.id,a.symbol,a.direction,a.target::float8,a.state,
	a.note,a.created_by,a.created_at,a.fired_at,
	t.last IS NOT NULL,t.last::float8,t.bid::float8,t.ask::float8,t.recv_ts,
	COALESCE(EXTRACT(EPOCH FROM now()-t.recv_ts)::bigint,0)
	FROM fin_price_alert a
	LEFT JOIN (SELECT DISTINCT ON (inst_id) inst_id,last,bid,ask,recv_ts FROM md_ticks
		ORDER BY inst_id,ts DESC) t ON t.inst_id=a.symbol`

func scanAlertRow(s interface{ Scan(...any) error }) (*dashboardv1.PriceAlertRow, error) {
	var r dashboardv1.PriceAlertRow
	var createdAt time.Time
	var firedAt, tickTs *time.Time
	var last, bid, ask *float64
	var ageS int64
	if err := s.Scan(&r.Id, &r.Symbol, &r.Direction, &r.Target, &r.State,
		&r.Note, &r.CreatedBy, &createdAt, &firedAt,
		&r.HasQuote, &last, &bid, &ask, &tickTs, &ageS); err != nil {
		return nil, err
	}
	if last != nil {
		r.Last, r.Bid, r.Ask = *last, *bid, *ask
	}
	r.CreatedAt = tsProto(createdAt)
	r.QuoteTs = tsProtoPtr(tickTs)
	r.FiredAt = tsProtoPtr(firedAt)
	r.QuoteAgeS = ageS
	return &r, nil
}

// alertRow 单行真值（建/翻后重读——回显禁）。
func (c coreService) alertRow(ctx context.Context, id int64) (*dashboardv1.PriceAlertRow, error) {
	return scanAlertRow(c.pool.QueryRow(ctx, alertRowSQL+" WHERE a.id=$1", id))
}

// ListPriceAlerts 报价卡+提醒面供数（读径合并窗内——ev_ticks 帧重取）。
func (c coreService) ListPriceAlerts(ctx context.Context,
	_ *dashboardv1.ListPriceAlertsRequest) (*dashboardv1.ListPriceAlertsResponse, error) {
	res := &dashboardv1.ListPriceAlertsResponse{AsOf: tsProto(time.Now())}
	rows, err := c.pool.Query(ctx, alertRowSQL+" ORDER BY a.id DESC")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		r, err := scanAlertRow(rows)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		res.Rows = append(res.Rows, r)
	}
	if err := rows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	// 候选域=活符∪目录符（T-0412——目录符携 pending 标，选建单即晋升）
	syms, pending, err := c.candSets(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	res.Symbols, res.PendingSymbols = syms, pending
	return res, nil
}

// —— T-0391 自选报价条面见 quotes_watch_face.go（NOX QuoteStrip 语义移植）——
