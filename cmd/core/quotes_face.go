// quotes_face.go —— T-0382 报价点+价格提醒读/写面（业主令：「想看到
// BTCUSDT/TRXUSDT 实时报价，并想预设一个价格，飞书通知我」）：
//
//	· ListPriceAlerts：fin_price_alert 行 LEFT JOIN md_ticks 最新价（报
//	  价卡 symbol 集=提醒行集——票面「一单两用」语义）+symbols 在采枚
//	  举供表单下拉；行级鲜度龄服务端派生（前端零判据）；
//	· CreatePriceAlert/SetPriceAlertState：同源闸窄写（earnOriginGate
//	  同规——Origin 缺席/跨源=拒）；写后真值重读非回显；
//	· 红线：单向告警非条件单——写径校验 symbol 在采/方向闭集/target>0，
//	  fired 终态禁手翻（一发即锁语义）；UI 操作落 ops_event 审计行
//	  （seat=dashboard-ui 同 earn 闸先例）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
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

const (
	alertSeatDash   = "dashboard-ui" // UI 写径席别（earn 闸同例）
	alertEventBuild = "price_alert_build"
	alertEventState = "price_alert_state"
)

// tickFreshSQL 提醒径鲜度谓词（T-0399——freshByKind 族「源节拍自适」同构
// 不新造档；判据=末 tick recv_ts 落库时刻非 ts 戳，防时钟偏移假象——
// alertRowSQL 鲜度龄同源）：快源钦定 15min 界（对齐动态数据闸口径）；
// mt5 流推源汇市周末 ~49h 休市静默非死源——界=72h（周末窗+边，15min
// 一刀切误伤防=票面红线）。扩 venue 源按其节拍补 CASE 不另开闸件。
const tickFreshSQL = `recv_ts > now()-CASE venue
	WHEN 'mt5' THEN interval '72 hours'
	ELSE interval '15 minutes' END`

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
	// 候选域同闸滤活（T-0399——死符不入下拉，孤行陈符不再过候选闸）
	srows, err := c.pool.Query(ctx, `SELECT DISTINCT inst_id FROM md_ticks
		WHERE `+tickFreshSQL+` ORDER BY inst_id`)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer srows.Close()
	for srows.Next() {
		var s string
		if err := srows.Scan(&s); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		res.Symbols = append(res.Symbols, s)
	}
	if err := srows.Err(); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	return res, nil
}

// CreatePriceAlert 建行（同源闸→校验→插行→真值重读；审计行 seat=dashboard-ui）。
func (c coreService) CreatePriceAlert(ctx context.Context,
	req *dashboardv1.CreatePriceAlertRequest) (*dashboardv1.CreatePriceAlertResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	sym := strings.ToUpper(strings.TrimSpace(req.GetSymbol()))
	dir := strings.TrimSpace(req.GetDirection())
	tgt := req.GetTarget()
	if sym == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "品种必填")
	}
	if dir != ">=" && dir != "<=" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "方向只收 涨到≥ / 跌破≤")
	}
	if math.IsNaN(tgt) || math.IsInf(tgt, 0) || tgt <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, "目标价须为正数")
	}
	// T-0399 鲜度闸同口径——EXISTS 孤行不再放行（270h 陈价过建单闸实证根修），
	// 死符拒建返可读错；upper 匹配取回 canon inst_id 落库——mt5 符号大小写方
	// 言（EURUSDm）不被归一打坏（rows join/提醒拍皆按原键对得上）。
	if err := c.pool.QueryRow(ctx, `SELECT inst_id FROM md_ticks
		WHERE upper(inst_id)=$1 AND `+tickFreshSQL+` LIMIT 1`, sym).Scan(&sym); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Sprintf("「%s」行情已陈或未在采——预设面只收活符品种", sym))
		}
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	var id int64
	if err := c.pool.QueryRow(ctx, `INSERT INTO fin_price_alert(symbol,direction,target,note,created_by)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		sym, dir, tgt, strings.TrimSpace(req.GetNote()), alertSeatDash).Scan(&id); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, alertSeatDash, alertEventBuild,
		fmt.Sprintf("#%d %s %s %s——价格提醒建行", id, sym, dir,
			strconv.FormatFloat(tgt, 'f', -1, 64))); err != nil {
		log.Printf("[quotes-face] 建行审计行落败（行已建——痕缺）: %v", err)
	}
	row, err := c.alertRow(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	return &dashboardv1.CreatePriceAlertResponse{Row: row}, nil
}

// SetPriceAlertState 停用/恢复翻位（active↔disabled 闭集——fired 终态禁手
// 翻=一发即锁语义；fired 行翻位请求=幂等回真值不报错）。
func (c coreService) SetPriceAlertState(ctx context.Context,
	req *dashboardv1.SetPriceAlertStateRequest) (*dashboardv1.SetPriceAlertStateResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	st := strings.TrimSpace(req.GetState())
	if st != "active" && st != "disabled" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"翻位只收 active/disabled——已提醒为终态")
	}
	tag, err := c.pool.Exec(ctx, `UPDATE fin_price_alert SET state=$2
		WHERE id=$1 AND state IN ('active','disabled')`, req.GetId(), st)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	row, err := c.alertRow(ctx, req.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err.Error())
	}
	if tag.RowsAffected() == 0 {
		return &dashboardv1.SetPriceAlertStateResponse{Row: row}, nil // fired 终态——幂等回真值
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, alertSeatDash, alertEventState,
		fmt.Sprintf("#%d %s 翻位 %s", req.GetId(), row.Symbol,
			map[string]string{"active": "恢复盯梢", "disabled": "停用"}[st])); err != nil {
		log.Printf("[quotes-face] 翻位审计行落败（态已翻——痕缺）: %v", err)
	}
	return &dashboardv1.SetPriceAlertStateResponse{Row: row}, nil
}

// —— T-0391 自选报价条面见 quotes_watch_face.go（NOX QuoteStrip 语义移植）——
