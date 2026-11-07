// quotes_alert_write.go —— T-0382 价格提醒写径面（建单/翻位——自 quotes_face.go
// 拆分：读径留原件，写径独立成件防单件超标；T-0442 三件落点）：
//
//	· CreatePriceAlert：T-0442 件② 一品种双阈形制——上限(>=)/下限(<=)各
//	  可空双空禁建，双填=单事务 1-2 行原子落（落行层仍 direction 行零
//	  schema 破）；T-0412 两域闸同事务晋升提名；
//	· SetPriceAlertState：active↔disabled 闭集翻位；T-0442 件① 删除=
//	  软删 disabled 留痕（fired 终态禁手翻=一发即锁——史行只读审计面）；
//	· 红线：同源闸窄写（earnOriginGate 同规）；写后真值重读非回显；
//	  UI 操作落 ops_event 审计行（seat=dashboard-ui 同 earn 闸先例）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"

	connect "connectrpc.com/connect/v2"

	"o/internal/gen/dashboardv1"
)

const (
	alertSeatDash   = "dashboard-ui" // UI 写径席别（earn 闸同例）
	alertEventBuild = "price_alert_build"
	alertEventState = "price_alert_state"
)

// CreatePriceAlert 建行（T-0442 件②一品种双阈形制——上限(>=)/下限(<=)各
// 可空双空禁建，双填=单事务 1-2 行原子落；落行层仍按 direction 逐行写零
// schema 破；同源闸→校验→插行→真值重读；审计行 seat=dashboard-ui）。
func (c coreService) CreatePriceAlert(ctx context.Context,
	req *dashboardv1.CreatePriceAlertRequest) (*dashboardv1.CreatePriceAlertResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	sym := strings.ToUpper(strings.TrimSpace(req.GetSymbol()))
	if sym == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, "品种必填")
	}
	// 双阈形制（业主令「同1品种可以同时设置两种条件，减少条目」——各可空，
	// 双空=禁建；在位标判填非零值判填——0 价本就 >0 闸拒）。
	type thr struct {
		dir string
		tgt float64
	}
	var thrs []thr
	for _, p := range []struct {
		has  bool
		tgt  float64
		dir  string
		name string
	}{{req.GetHasUpper(), req.GetUpperTarget(), ">=", "上限"},
		{req.GetHasLower(), req.GetLowerTarget(), "<=", "下限"}} {
		if !p.has {
			continue
		}
		if math.IsNaN(p.tgt) || math.IsInf(p.tgt, 0) || p.tgt <= 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, p.name+"价须为正数")
		}
		thrs = append(thrs, thr{p.dir, p.tgt})
	}
	if len(thrs) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, "上限/下限至少填一——双空禁建")
	}
	// T-0412 两域闸（T-0399 改判）——在采活符=收；目录活符=收+同事务晋升
	// 提名（fin_sub_request user_intent——建得起提醒永远不评估的僵尸病灶
	// 根修）；两域外=拒（typo 闸不放松）。canon 原键落库大小写方言不打坏。
	canon, venue, pending, err := c.symDomain(ctx, sym)
	if errors.Is(err, errNoDomain) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Sprintf("「%s」不在行情目录——预设面收在采/目录内品种", sym))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	sym = canon
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var ids []int64
	conds := make([]string, 0, len(thrs))
	for _, t := range thrs {
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO fin_price_alert(symbol,direction,target,note,created_by)
			VALUES($1,$2,$3,$4,$5) RETURNING id`,
			sym, t.dir, t.tgt, strings.TrimSpace(req.GetNote()), alertSeatDash).Scan(&id); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		ids = append(ids, id)
		conds = append(conds, fmt.Sprintf("#%d %s %s", id, t.dir,
			strconv.FormatFloat(t.tgt, 'f', -1, 64)))
	}
	if pending { // 目录符提名晋升——同事务原子落（提醒行与订阅请求同生共死）
		if _, err := tx.Exec(ctx, promoteSQL, venue, sym,
			fmt.Sprintf("price_alert#%d", ids[0])); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error())
	}
	if _, err := c.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note)
		VALUES($1,$2,'',$3)`, alertSeatDash, alertEventBuild,
		fmt.Sprintf("%s %s——价格提醒建行%s", sym, strings.Join(conds, " / "),
			map[bool]string{true: "（目录符晋升提名 user_intent 已落）", false: ""}[pending])); err != nil {
		log.Printf("[quotes-face] 建行审计行落败（行已建——痕缺）: %v", err)
	}
	rows := make([]*dashboardv1.PriceAlertRow, 0, len(ids))
	for _, id := range ids {
		row, err := c.alertRow(ctx, id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err.Error())
		}
		rows = append(rows, row)
	}
	return &dashboardv1.CreatePriceAlertResponse{Rows: rows}, nil
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
			map[string]string{"active": "恢复盯梢", "disabled": "停用（软删留痕——已停用区可恢复）"}[st])); err != nil {
		log.Printf("[quotes-face] 翻位审计行落败（态已翻——痕缺）: %v", err)
	}
	return &dashboardv1.SetPriceAlertStateResponse{Row: row}, nil
}
