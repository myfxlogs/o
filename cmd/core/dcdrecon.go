// dcdrecon.go —— dcd 结算对账拍（工单#22919 钦定修法①自 restlive.go
// 拆出——棘轮 300 行顶线）。fundSaveBalances 读拍驱动、全幂等、写败只
// log 不炸权益面。
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"o/internal/trading/okxexec"
)

// dcdSettleReconcile 结算对账拍（工单#22919 钦定修法①——dcd_live 族无结算
// 消费件根修）。读拍驱动、全幂等、写败只 log 不炸权益面：
//
//	① settled/redeemed 单 → 账本 open exec 仓收敛 closed_at+settled_*
//	  （venue_ord_id 确证键优先，productId 单 open 仓模糊配兜底——旧仓无
//	  ordId 的过渡径；同 productId 多 open 仓=歧义跳过如实记）；
//	② 结算回款币注册 fin_risk_decision op='dcd_settle'（FundCcy funding/earn
//	  域扩集——BTC 回款非册币永久漏计面根修；基线凭据同键退役靠凭据 NOT
//	  EXISTS 排除非改行）；
//	③ venue live∧账本零覆盖（exec 仓∪归属域册∪基线凭据）→反向孤儿升件；
//	④ 结算实拍 md_account_events channel='dcd_settle' 落账（#20904 验收
//	  的到期迁移序列活证——状态/settleSz/settleCcy/settlePx/settleTime 全载）。
func dcdSettleReconcile(ctx context.Context, pool *pgxpool.Pool, ords []okxexec.DcdOrder) {
	for _, od := range ords {
		if !strings.EqualFold(od.State, "settled") && !strings.EqualFold(od.State, "redeemed") {
			continue
		}
		dcdSettleOne(ctx, pool, od)
	}
	for _, od := range ords {
		if !strings.EqualFold(od.State, "live") {
			continue
		}
		dcdOrphanCheck(ctx, pool, od)
	}
}

// dcdSettleOne 单结算收敛（幂等——closed_at 置位/凭据在册即跳）。
func dcdSettleOne(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder) {
	settleTs := "now()"
	if ms := okxexec.ParseF(od.SettleTime); ms > 0 {
		settleTs = fmt.Sprintf("to_timestamp(%v/1000.0)", ms)
	}
	// ① exec 仓收敛：venue_ord_id 确证键→productId 单 open 仓模糊配。
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Printf("[dcd] settle 收敛 tx 起败 ordId=%s: %v", od.OrdID, err)
		return
	}
	defer tx.Rollback(ctx)
	rows, qerr := tx.Query(ctx, `SELECT p.id,p.execution_id FROM fin_position p
		JOIN fin_execution e ON e.id=p.execution_id
		WHERE p.state<>'closed' AND (
			e.venue_ord_id=$1 OR (e.venue_ord_id IS NULL AND p.inst_id=$2))
		ORDER BY (e.venue_ord_id=$1) DESC`, od.OrdID, od.ProductID)
	var pid, eid int64
	var nmatch int
	if qerr == nil {
		for rows.Next() {
			nmatch++
			if nmatch == 1 {
				_ = rows.Scan(&pid, &eid)
			} else {
				var a, b int64
				_ = rows.Scan(&a, &b)
			}
		}
		rows.Close()
		qerr = rows.Err()
	}
	if qerr != nil || nmatch == 0 || nmatch > 1 {
		// 零配=凭据域单（业主 dcd_baseline）或账外单；多配=同 productId
		// 歧义禁猜收敛——皆跳过不炸（实拍行载 execPosId=0 留证）。
		if nmatch > 1 {
			log.Printf("[dcd] settle ordId=%s productId=%s 多 open 仓歧义(%d)禁收敛", od.OrdID, od.ProductID, nmatch)
		}
		_ = tx.Rollback(ctx)
	} else {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE fin_position
			SET closed_at=%s, settled_at=%s, settled_amount=$1, settled_ccy=$2
			WHERE id=$3 AND state<>'closed'`, settleTs, settleTs),
			okxexec.ParseF(od.SettleSz), od.SettleCcy, pid); err != nil {
			log.Printf("[dcd] settle 仓收敛写败 pos=%d ordId=%s: %v", pid, od.OrdID, err)
			_ = tx.Rollback(ctx)
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state='closed', venue_ord_id=$2
			WHERE id=$1 AND state='open'`, eid, od.OrdID); err != nil {
			log.Printf("[dcd] settle exec 收敛写败 exec=%d ordId=%s: %v", eid, od.OrdID, err)
			_ = tx.Rollback(ctx)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			log.Printf("[dcd] settle 收敛 commit 败 ordId=%s: %v", od.OrdID, err)
			return
		}
	}
	// ② 回款币注册（dcd_settle 凭据幂等——ordId 在册即跳；FundCcy
	// funding/earn 域据此扩集，venue 回款到账即被权益面接管）。
	var have bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle' AND intent->>'ordId'=$1)`, od.OrdID).Scan(&have); err != nil {
		log.Printf("[dcd] settle 凭据查败 ordId=%s: %v", od.OrdID, err)
		return
	}
	if !have {
		intent, _ := json.Marshal(map[string]any{
			"op": "dcd_settle", "kind": "transfer", "venue": "okx",
			"ccy": od.SettleCcy, "amt": okxexec.ParseF(od.SettleSz),
			"ordId": od.OrdID, "productId": od.ProductID,
			"settlePx": od.SettlePx, "settleTime": od.SettleTime})
		if _, err := pool.Exec(ctx, `INSERT INTO fin_risk_decision(intent,decision,reason,config_gen,kind)
			VALUES($1,'allow','transfer',1,'open')`, intent); err != nil {
			log.Printf("[dcd] settle 凭据写败 ordId=%s: %v", od.OrdID, err)
		}
	}
	// ③ 实拍账（全载结算落果——回款币种/价/量/到期刻 venue 原值）。
	ev, _ := json.Marshal(map[string]any{
		"ordId": od.OrdID, "productId": od.ProductID, "state": od.State,
		"notionalCcy": od.NotionalCcy, "notionalSz": od.NotionalSz,
		"settleCcy": od.SettleCcy, "settleSz": od.SettleSz, "settlePx": od.SettlePx,
		"yieldCcy": od.YieldCcy, "yieldSz": od.YieldSz,
		"settleTime": od.SettleTime, "expTime": od.ExpTime, "execPosId": pid})
	if _, err := pool.Exec(ctx, `INSERT INTO md_account_events(venue,channel,ccy,inst_id,data,ts)
		VALUES('okx','dcd_settle',$1,$2,$3,now())
		ON CONFLICT (venue,channel,ccy,inst_id) DO UPDATE SET ts=EXCLUDED.ts,
		data=EXCLUDED.data,recv_ts=now()`, od.SettleCcy, od.OrdID, ev); err != nil {
		log.Printf("[dcd] settle 实拍行写败 ordId=%s: %v", od.OrdID, err)
	}
}

// dcdOrphanCheck venue live∧账本零覆盖=反向孤儿升件（仓外漂无敞口账面——
// exec open 仓∪归属域册 ord_id∪dcd_baseline 凭据 ordId 三覆盖源全缺即升；
// 24h 同 ordId 去重窗防连拍炸件）。
func dcdOrphanCheck(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder) {
	var covered bool
	if err := pool.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM fin_position p JOIN fin_execution e ON e.id=p.execution_id
		       WHERE p.inst_id=$1 AND p.state<>'closed')
		OR EXISTS(SELECT 1 FROM fin_position_domain WHERE ord_id=$2 OR inst_id=$1)
		OR EXISTS(SELECT 1 FROM fin_risk_decision
		       WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'=$2)`,
		od.ProductID, od.OrdID).Scan(&covered); err != nil || covered {
		if err != nil {
			log.Printf("[dcd] 孤儿覆盖查败 ordId=%s: %v", od.OrdID, err)
		}
		return
	}
	var dup bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class='dcd_orphan' AND esc_scope=$1
		AND ts>now()-interval '24 hours')`, od.OrdID).Scan(&dup); err != nil || dup {
		return
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','工单/22919',$1,'dcd_orphan',$2)`,
		fmt.Sprintf("dcd 反向孤儿：venue LIVE ordId=%s productId=%s notional=%s %s——账本 exec 仓/归属域册/基线凭据三覆盖源全缺（结算对账拍发现，仓账外漂无敞口账）",
			od.OrdID, od.ProductID, od.NotionalSz, od.NotionalCcy),
		od.OrdID); err != nil {
		log.Printf("[dcd] 孤儿升件写败 ordId=%s: %v", od.OrdID, err)
	}
}
