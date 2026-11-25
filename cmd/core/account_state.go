// account_state.go —— O-39440 账户级生命周期面（fin_account state 机
// 人工径——venue 粒度归 SetBrokerState(fin_venue_roster)；本 RPC=账户
// 粒度：CFD login 户/结构子户的启停删。业主令 2026-11-25：换经纪商是
// 常态——账户册须 frontend 可管,硬编码 venue→account 禁。
//
// 迁移机（同源闸→形闸→迁移机→同事务落+审计行）：
//
//	active（启）：仅 disabled→active；active 幂等回执；closed 终态拒、
//	  frozen(legacy 兜底桶)非生命周期域拒。
//	disabled（停）：active→disabled；disabled 幂等回执。停用语义=运营面
//	  （venue 实读/对拍/NAV 遍历 state='active' 过滤天然缺席）——簿记归户
//	  不受影响（触发器 disabled 档仍归本户——钱照样在,禁猜他户）。
//	closed（删/退役）：仅 disabled 前置（跳档拒——停→核→删审计律同
//	  SetBrokerState）；零引用核（fin_cashflow FK 行+fin_known_residual
//	  软引用行全零）→物理 DELETE 真删（回执 'deleted'）；有史→state=
//	  'closed' 墓碑（历史永存不撕页,归户触发器 closed 档落 <venue>:legacy）。
//	  结构子户（sub_type<>'main'）非生命周期域拒——okx trading/funding/
//	  earn/dcd 桶是同一 API key 的功能分面,停用=venue 粒度 SetBrokerState。
package main

import (
	"context"
	"fmt"
	"regexp"

	"connectrpc.com/connect/v2"
	"github.com/jackc/pgx/v5"

	"o/internal/gen/dashboardv1"
)

// accountIDRE fin_account 域键形闸（venue:account——小写 venue token
// +户号/子户名段；login 数字/CJK 不入——户号为数字或枚举子户名）。
var accountIDRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,31}:[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

// SetAccountState fin_account 态迁移窄写面（earnOriginGate 同源闸——
// dashboard 径只收 same-origin;迁移机+审计行同事务）。
func (c coreService) SetAccountState(ctx context.Context,
	req *dashboardv1.SetAccountStateRequest) (*dashboardv1.SetAccountStateResponse, error) {
	if err := earnOriginGate(ctx); err != nil {
		return nil, err
	}
	acct := req.GetAccountId()
	if !accountIDRE.MatchString(acct) {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"account_id 形拒——<venue>:<account>（小写 venue token+户号/子户名）")
	}
	target := req.GetState()
	switch target {
	case "active", "disabled", "closed":
	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"state 形拒——active|disabled|closed 三值（frozen=legacy 桶非人工迁移域）")
	}
	var prev, subType, venue string
	err := c.pool.QueryRow(ctx,
		`SELECT state,sub_type,venue FROM fin_account WHERE account_id=$1`, acct).Scan(&prev, &subType, &venue)
	if err == pgx.ErrNoRows {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			"fin_account 无户 "+acct+"——账户册行先建（迁移/触发器自动开户/登记径）")
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "fin_account 读: "+err.Error())
	}
	if prev == "closed" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			acct+" 已 closed 终态——不可迁移（复挂=新 account_id 新行）")
	}
	if prev == "frozen" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			acct+" 为 "+venue+":legacy 兜底桶——非账户非生命周期域（frozen 终身标）")
	}
	if subType != "main" {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			acct+" sub_type="+subType+" 结构子户——非账户生命周期域（venue 粒度停用归 SetBrokerState）")
	}
	// 迁移机：幂等双回执/复活仅 disabled→active/closed 仅 disabled 前置。
	var next string
	switch target {
	case "active":
		switch prev {
		case "disabled":
			next = "active"
		case "active":
			return &dashboardv1.SetAccountStateResponse{AccountId: acct, PrevState: prev, State: prev}, nil
		default:
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				acct+" 态 "+prev+" 非 disabled——active 复活径只收人工停用户")
		}
	case "disabled":
		if prev == "disabled" {
			return &dashboardv1.SetAccountStateResponse{AccountId: acct, PrevState: prev, State: prev}, nil
		}
		next = "disabled"
	case "closed":
		if prev != "disabled" {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				acct+" 态 "+prev+" 非 disabled——closed 前置硬闸：先停用再核引用再删（跳档拒）")
		}
	}

	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx: "+err.Error())
	}
	defer tx.Rollback(ctx)
	final := next
	if target == "closed" {
		// 零引用核：fin_cashflow FK 行+fin_known_residual 软引用行——
		// 全零=真删（账户从未被簿记/豁免面引用,行物理 DELETE）；
		// 有史=墓碑 state='closed'（历史永存,归户触发器 closed 档
		// 落 <venue>:legacy——旧户不再收新归属）。
		var refs int
		if err := tx.QueryRow(ctx, `SELECT
			(SELECT count(*) FROM fin_cashflow WHERE account_id=$1) +
			(SELECT count(*) FROM fin_known_residual WHERE account_id=$1)`, acct).Scan(&refs); err != nil {
			return nil, connect.NewError(connect.CodeInternal, "引用核: "+err.Error())
		}
		if refs == 0 {
			if _, err := tx.Exec(ctx, `DELETE FROM fin_account WHERE account_id=$1`, acct); err != nil {
				return nil, connect.NewError(connect.CodeInternal, "账户删: "+err.Error())
			}
			final = "deleted"
		} else {
			if _, err := tx.Exec(ctx, `UPDATE fin_account SET state='closed' WHERE account_id=$1`, acct); err != nil {
				return nil, connect.NewError(connect.CodeInternal, "账户墓碑: "+err.Error())
			}
			final = "closed"
		}
	} else if _, err := tx.Exec(ctx, `UPDATE fin_account SET state=$2 WHERE account_id=$1`,
		acct, next); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "账户迁移: "+err.Error())
	}
	// 审计行（seat=dashboard-ui 同 brokerCredActor 口径；凭据/login 之外
	// 字段全禁入 note——account_id 本身即域键非密文）。
	note := fmt.Sprintf("account=%s venue=%s %s→%s", acct, venue, prev, final)
	if nb := req.GetNote(); nb != "" {
		note += " 备注=" + nb
	}
	if final == "deleted" {
		note += " 零引用物理删"
	} else if final == "closed" {
		note += " 有史墓碑"
	}
	if _, err := tx.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_scope)
		VALUES($1,$2,$3,$4,$5)`, brokerCredActor, "账户态迁移", "O-39440", note, acct); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "审计行写: "+err.Error())
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, "tx commit: "+err.Error())
	}
	return &dashboardv1.SetAccountStateResponse{AccountId: acct, PrevState: prev, State: final}, nil
}
