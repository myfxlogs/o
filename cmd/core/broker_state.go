// broker_state.go —— O-37450 账户生命周期态迁移面（fin_venue_roster
// state 机人工径——设计 SSOT venue-multi-account.md D6「停/删」行）。
// 会话环自报 connecting/active/degraded；本 RPC 只许三态人工迁移：
//
//	disabled（停）：会话环下拍即收（roster 迭代 state 过滤）+executor
//	  拒单面自见——历史永存不撕页；幂等（已 disabled=回执现态）。
//	active（复活）：仅 disabled→connecting——会话环下拍重接管；
//	  registered/connecting/active/degraded→active 幂等回执现态
//	  （非人工迁移域——会话环自报权威）。
//	removed（墓碑）：硬闸三前置——①state 须先 disabled（跳档拒）；
//	  ②零持仓核（fin_strategy 非 retired=0 ∧ fin_order_leg 未平仓
//	  position=0 ∧ dcd allow 凭据=0 ∧ Earn 停泊本金=0——任一在=拒，
//	  凭据不可恢复）；③过闸同事务删凭据族 `<cred_ns>.%`+state='removed'
//	  墓碑+审计行。removed→任何=终态拒（墓碑不可复活——复挂=新 venue
//	  token 新行）。
package main

import (
	"context"
	"fmt"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgx/v5"

	"o/internal/gen/dashboardv1"
)

// SetBrokerState roster 态迁移窄写面（同源闸→形闸→迁移机→同事务落
// +审计行；removed=凭据族删同事务——不可恢复面全前置核过才动手）。
func (c coreService) SetBrokerState(ctx context.Context,
	req *dashboardv1.SetBrokerStateRequest) (*dashboardv1.SetBrokerStateResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	venue := req.GetVenue()
	if !brokerVenueRE.MatchString(venue) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"venue 形拒——[a-z0-9_-]{1,32} 小写数线横 token")
	}
	target := req.GetState()
	switch target {
	case "disabled", "active", "removed":
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"state 形拒——disabled|active|removed 三值（会话环自报 connecting/active/degraded 非人工迁移域）")
	}
	var prev, credNS string
	err := c.pool.QueryRow(ctx,
		`SELECT state,cred_ns FROM fin_venue_roster WHERE venue=$1`, venue).Scan(&prev, &credNS)
	if err == pgx.ErrNoRows {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"roster 无户 venue="+venue+"——凭据先经 SubmitBrokerCred 录户")
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "roster 读: "+err.Error())
	}
	if prev == "removed" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			"venue="+venue+" 已 removed 墓碑——终态不可迁移（复挂=新 venue token 新行）")
	}
	// 迁移机：disabled 幂等/复活仅 disabled→connecting/removed 仅 disabled 前置。
	var next string
	switch target {
	case "disabled":
		if prev == "disabled" {
			return &dashboardv1.SetBrokerStateResponse{Venue: venue, PrevState: prev, State: prev}, nil // 幂等——重复停=回执现态零审计行
		}
		next = "disabled"
	case "active":
		switch prev {
		case "disabled":
			next = "connecting" // 复活——会话环下拍重接管自报
		case "connecting", "active":
			return &dashboardv1.SetBrokerStateResponse{Venue: venue, PrevState: prev, State: prev}, nil // 幂等
		default:
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				"venue="+venue+" 态 "+prev+" 非 disabled——active 复活径只收人工停用户（会话环自报态非人工迁移域）")
		}
	case "removed":
		if prev != "disabled" {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				"venue="+venue+" 态 "+prev+" 非 disabled——removed 前置硬闸：先停用再核仓再删（跳档拒）")
		}
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx: "+err.Error())
	}
	defer tx.Rollback(ctx)
	if target == "removed" {
		// 零持仓核（过闸才动手——凭据删不可恢复，任一面在=拒）：
		// ①fin_strategy 非 retired 行；②未平仓执行仓腿（leg venue 命中
		// ∧ position state<>'closed'——orphaned 保守计入=venue 实持未结）；
		// ③dcd allow 凭据（fin_risk_decision intent venue 命中）；
		// ④Earn 停泊本金（md_account_events earn_parking 快照 earn_bal>0——
		// O-38013 补漏：dcd allow 只覆盖 DCD 径，Earn 在停本金在册仍删凭据
		// =断供洞；earn_bal 缺/非数值行按 0 不计）。
		var blockers []string
		var n int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM fin_strategy WHERE venue=$1 AND state<>'retired'`, venue).Scan(&n); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "策略核: "+err.Error())
		} else if n > 0 {
			blockers = append(blockers, fmt.Sprintf("策略非退役 %d 行", n))
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM fin_order_leg l
			JOIN fin_position p ON p.execution_id=l.execution_id
			WHERE l.venue=$1 AND p.state<>'closed'`, venue).Scan(&n); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "执行仓核: "+err.Error())
		} else if n > 0 {
			blockers = append(blockers, fmt.Sprintf("未平仓执行仓 %d 腿", n))
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM fin_risk_decision
			WHERE decision='allow' AND intent->>'op'='dcd_baseline' AND intent->>'venue'=$1`, venue).Scan(&n); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "产品仓核: "+err.Error())
		} else if n > 0 {
			blockers = append(blockers, fmt.Sprintf("dcd allow 凭据 %d 笔", n))
		}
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM md_account_events
			WHERE venue=$1 AND channel='earn_parking'
			  AND CASE WHEN data->>'earn_bal' ~ '^-?[0-9]+(\.[0-9]+)?$'
			           THEN (data->>'earn_bal')::numeric ELSE 0 END > 0`, venue).Scan(&n); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "Earn 停泊核: "+err.Error())
		} else if n > 0 {
			blockers = append(blockers, fmt.Sprintf("Earn 停泊本金 %d 币", n))
		}
		if len(blockers) > 0 {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				"venue="+venue+" 零持仓核未过——"+joinBlockers(blockers)+"（先平仓/退役再删）")
		}
		// 凭据族删同事务（cred_ns 前缀全键——不可恢复面最后动）。
		if _, err := tx.Exec(ctx, `DELETE FROM app_secrets WHERE key LIKE $1||'.%'`, credNS); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "凭据族删: "+err.Error())
		}
		next = "removed"
	}
	if _, err := tx.Exec(ctx, `UPDATE fin_venue_roster
		SET state=$2,last_error=NULL,updated_at=now() WHERE venue=$1`,
		venue, next); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "roster 迁移: "+err.Error())
	}
	// 审计行（seat=dashboard-ui 同凭据件口径；login/凭据值永禁入 note）。
	note := fmt.Sprintf("venue=%s %s→%s cred_ns=%s", venue, prev, next, credNS)
	if nb := req.GetNote(); nb != "" {
		note += " 备注=" + nb
	}
	if target == "removed" {
		note += " 凭据族删"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_scope)
		VALUES($1,$2,$3,$4,$5)`, brokerCredActor, "venue态迁移", "O-37450", note, venue); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "审计行写: "+err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx commit: "+err.Error())
	}
	return &dashboardv1.SetBrokerStateResponse{Venue: venue, PrevState: prev, State: next}, nil
}

func joinBlockers(b []string) string {
	out := ""
	for i, s := range b {
		if i > 0 {
			out += "、"
		}
		out += s
	}
	return out
}
