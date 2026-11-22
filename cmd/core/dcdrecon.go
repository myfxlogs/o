// dcdrecon.go —— dcd 结算对账拍（工单#22919 钦定修法①自 restlive.go
// 拆出——棘轮 300 行顶线；O-33362 簿记机械环——孤儿升级件退役改
// 自动补挂+现金腿配对）。fundSaveBalances 读拍驱动、全幂等、写败只
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
// 消费件根修；O-33362：③孤儿升件退役改 dcdLiveBook 自动补挂环）。
// 读拍驱动、全幂等、写败只 log 不炸权益面：
//
//	① settled/redeemed 单 → 账本 open exec 仓收敛 closed_at+settled_*
//	  （venue_ord_id 确证键优先，productId 单 open 仓模糊配兜底——旧仓无
//	  ordId 的过渡径；同 productId 多 open 仓=歧义跳过如实记）；
//	② 结算回款币注册 fin_risk_decision op='dcd_settle' + counted 回款腿
//	  同事务（FundCcy funding/earn 域扩集——BTC 回款非册币永久漏计面根修；
//	  基线凭据同键退役靠凭据 NOT EXISTS 排除非改行）；
//	③ venue 名义在持（live∪pending 族）∧账本零覆盖 → dcdLiveBook 自动
//	  补挂 dcd_baseline 凭据+申购 counted 腿（写败才升 dcd_baseline_fail）；
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
		if !dcdNotionalLive(od) {
			continue
		}
		dcdLiveBook(ctx, pool, od)
	}
}

// dcdNotionalLive 名义在持集（restlive addOrd 同集——live/pending_settle/
// pending_redeem 三态名义照计；settled/redeemed 走回款 settleCcy 径）。
func dcdNotionalLive(od okxexec.DcdOrder) bool {
	switch {
	case strings.EqualFold(od.State, "live"),
		strings.EqualFold(od.State, "pending_settle"),
		strings.EqualFold(od.State, "pending_redeem"):
		return true
	}
	return false
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
		// 工单#24798：venue_ord_id 确证键回填与 exec 态迁解耦——孤儿裁定仓
		// exec 早已 closed（仓42 实证形），原 WHERE state='open' 把 ordId 永
		// 久留空=确证键断根；已闭 exec 只补 ordId 不动态，open exec 同拍迁 closed。
		if _, err := tx.Exec(ctx, `UPDATE fin_execution SET state=CASE WHEN state='open' THEN 'closed' ELSE state END,
			venue_ord_id=COALESCE(NULLIF(venue_ord_id,''),$2)
			WHERE id=$1`, eid, od.OrdID); err != nil {
			log.Printf("[dcd] settle exec 收敛写败 exec=%d ordId=%s: %v", eid, od.OrdID, err)
			_ = tx.Rollback(ctx)
			return
		}
		if err := tx.Commit(ctx); err != nil {
			log.Printf("[dcd] settle 收敛 commit 败 ordId=%s: %v", od.OrdID, err)
			return
		}
	}
	// ② 回款币注册+counted 回款腿同事务（dcd_settle 凭据幂等——ordId 在册
	// 即整组跳过不重复落腿；O-33362：凭据与现金腿同生同灭零半账）。
	var have bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM fin_risk_decision
		WHERE intent->>'op'='dcd_settle' AND intent->>'ordId'=$1)`, od.OrdID).Scan(&have); err != nil {
		log.Printf("[dcd] settle 凭据查败 ordId=%s: %v", od.OrdID, err)
		return
	}
	if !have {
		if err := dcdSettleWrite(ctx, pool, od); err != nil {
			log.Printf("[dcd] settle 凭据/回款腿写败 ordId=%s: %v", od.OrdID, err)
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

// dcdLiveBook venue 名义在持单账本覆盖环（O-33362 簿记机械环——孤儿升级件
// 退役改自动补挂）：exec 活仓（open/closing——orphaned=脱钩探针态非健康
// 覆盖，staleexec 另径收编）∪归属域册∪基线凭据三覆盖源全缺 → 同事务落
// dcd_baseline 凭据+申购 counted 腿；写败才升 dcd_baseline_fail 专属告警
// （24h 同 ordId 去重窗）。
func dcdLiveBook(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder) {
	var covered bool
	if err := pool.QueryRow(ctx, `SELECT
		EXISTS(SELECT 1 FROM fin_position p JOIN fin_execution e ON e.id=p.execution_id
		       WHERE p.inst_id=$1 AND p.state NOT IN ('closed','orphaned'))
		OR EXISTS(SELECT 1 FROM fin_position_domain WHERE ord_id=$2 OR inst_id=$1)
		OR EXISTS(SELECT 1 FROM fin_risk_decision
		       WHERE intent->>'op'='dcd_baseline' AND intent->>'ordId'=$2)`,
		od.ProductID, od.OrdID).Scan(&covered); err != nil {
		log.Printf("[dcd] 覆盖查败 ordId=%s: %v", od.OrdID, err)
		return
	}
	if covered {
		return
	}
	if err := dcdBaselineWrite(ctx, pool, od); err != nil {
		log.Printf("[dcd] 基线补挂写败 ordId=%s: %v", od.OrdID, err)
		dcdBaselineFailEsc(ctx, pool, od, err)
	}
}

// dcdBaselineFailEsc 基线补挂写败专属告警——venue 名义在持单账本补挂失败
// =簿记环断需人工介入（孤儿升件退役后唯一升件径；24h 同 ordId 去重）。
func dcdBaselineFailEsc(ctx context.Context, pool *pgxpool.Pool, od okxexec.DcdOrder, werr error) {
	var dup bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM ops_event
		WHERE event='升件' AND esc_class='dcd_baseline_fail' AND esc_scope=$1
		AND ts>now()-interval '24 hours')`, od.OrdID).Scan(&dup); err != nil || dup {
		return
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('探测席','升件','',$1,'dcd_baseline_fail',$2)`,
		fmt.Sprintf("dcd 基线补挂写败：venue 名义在持 ordId=%s productId=%s notional=%s %s——三覆盖源全缺且自动补挂失败（%v）——簿记环断需人工介入",
			od.OrdID, od.ProductID, od.NotionalSz, od.NotionalCcy, werr),
		od.OrdID); err != nil {
		log.Printf("[dcd] 补挂败升件写败 ordId=%s: %v", od.OrdID, err)
	}
}
