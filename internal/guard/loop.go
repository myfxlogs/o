// loop.go —— 守卫环自残探针+熔断挂起（T-0423，T-0419 范围 2b 遗留件；
// 业主令 2026-11-07「机器发现不对但不知为何，照开照烧」案熔断补层）。
//
// 病灶（pos34 实证）：closing 循环连绞 attempts=5 烧手续费，中途零告警——
// close_gave_up 仅耗尽才火。钦定语义：
//
//	fin_guard_close state='closing' 且 (attempts≥N 或 now-last_try_at>Xh)
//	  → 探针行（loop_alerted_at 落戳=节流锚，同窗至多一报）
//	  + Alert(guard_close_loop) + ops_event 升件行
//	  + 熔断：同策略开仓权挂起（fin_guard_suspend——禁新开平仓权保留，
//	    解冻唯一径=裁定席判词或业主令 tools/guardlift）
//
// 阈键 guard.loop_alert_attempts/guard.loop_alert_hours（dm 类——缺键
// fail-closed 探针跳过+计数，哨戒主环不炸）。
package guard

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// loopStuck closing 滞留行（探针候选——阈判定在 SQL 谓词内完成）。
type loopStuck struct {
	posID    int64
	stratRef int64
	attempts int64
	since    time.Time  // 滞留锚=COALESCE(last_try_at,建仓刻)
	alerted  *time.Time // loop_alerted_at（NULL=未升过）
}

// loopProbe 自残探针一拍（scanOnce 内挂——闸径独立阈键，读败/缺键=探针
// 本面跳过不影响主扫描）。
func (w *Warden) loopProbe(ctx context.Context) {
	// reconcile 先扫（T-0511 僵尸行实证）：position 已 closed 的 'closing'
	// 行=带外闭环（drill 清扫径/人工径写 position.closed_at 未走
	// markClosed 双写）——scanOnce 按 closed_at 滤仓永不再碰，探针
	// 不收它就是对死行每阈升假升件。落 'closed' 免报；阈键缺件也照扫。
	if _, err := w.pool.Exec(ctx, `UPDATE fin_guard_close g
		SET state='closed',closed_at=now(),updated_at=now(),
		    last_error='reconcile:position_closed_outside_guard'
		FROM fin_position p
		WHERE g.position_id=p.id AND g.state='closing' AND p.closed_at IS NOT NULL`); err != nil {
		w.rej.Inc("write_err")
	}
	maxAtt, err1 := w.getF("guard.loop_alert_attempts")
	maxHrs, err2 := w.getF("guard.loop_alert_hours")
	if err1 != nil || err2 != nil {
		w.rej.Inc("missing_param")
		w.note("missing_param:loop", fmt.Sprintf("loop_alert 阈键读败——自残探针跳过: %v/%v", err1, err2))
		return
	}
	rows, err := w.pool.Query(ctx, `
		SELECT g.position_id, p.strategy_ref, g.attempts,
		       COALESCE(g.last_try_at,p.opened_at), g.loop_alerted_at
		FROM fin_guard_close g JOIN fin_position p ON p.id=g.position_id
		WHERE g.state='closing' AND p.closed_at IS NULL
		  AND (g.attempts >= $1
		       OR now()-COALESCE(g.last_try_at,p.opened_at) > make_interval(secs => $2*3600))`,
		int64(maxAtt), maxHrs)
	if err != nil {
		w.rej.Inc("facts_missing")
		log.Printf("WARN [guard] 自残探针扫面读败: %v", err)
		return
	}
	var batch []loopStuck
	for rows.Next() {
		var s loopStuck
		if err := rows.Scan(&s.posID, &s.stratRef, &s.attempts, &s.since, &s.alerted); err != nil {
			rows.Close()
			w.rej.Inc("facts_missing")
			return
		}
		batch = append(batch, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		w.rej.Inc("facts_missing")
		return
	}
	throttle := time.Duration(maxHrs * float64(time.Hour))
	for _, s := range batch {
		w.suspendOpen(ctx, s)
		if s.alerted != nil && time.Since(*s.alerted) < throttle {
			continue // 同窗已报——节流不连拍重复刷（熔断幂等仍每拍保活）
		}
		w.loopEscalate(ctx, s, maxAtt, maxHrs)
	}
}

// suspendOpen 熔断挂起开仓权（幂等——部分唯一索引 ON CONFLICT 兜同策略重复
// 触发；首挂落 ops_event「开仓挂起」行=挂起态非静默）。返 true=本拍新挂。
func (w *Warden) suspendOpen(ctx context.Context, s loopStuck) bool {
	reason := fmt.Sprintf("守卫环自残：仓#%d closing attempts=%d 滞留=%.1fh", s.posID, s.attempts, time.Since(s.since).Hours())
	tag, err := w.pool.Exec(ctx, `INSERT INTO fin_guard_suspend(strategy_ref,position_id,reason)
		VALUES($1,$2,$3)
		ON CONFLICT (strategy_ref) WHERE lifted_at IS NULL DO NOTHING`,
		s.stratRef, s.posID, reason)
	if err != nil {
		w.rej.Inc("write_err")
		return false
	}
	if tag.RowsAffected() == 0 {
		return false
	}
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES('仓巡','开仓挂起',$1)`,
		fmt.Sprintf("策略#%d 仓#%d %s——禁新开平仓权保留（只出不进），解冻唯一径=裁定席判词/业主令",
			s.stratRef, s.posID, reason)); err != nil {
		w.rej.Inc("write_err")
	}
	log.Printf("ALERT [guard] 策略 %d 仓 %d 守卫环自残熔断——开仓权挂起（%s）", s.stratRef, s.posID, reason)
	return true
}

// loopEscalate 探针行落戳+升件行+Alert 三件同拍（升件写败不回头滚戳——
// 戳已落=下拍节流，升件径写败由升件行缺位自显）。
func (w *Warden) loopEscalate(ctx context.Context, s loopStuck, maxAtt, maxHrs float64) {
	stuckH := time.Since(s.since).Hours()
	if _, err := w.pool.Exec(ctx,
		`UPDATE fin_guard_close SET loop_alerted_at=now() WHERE position_id=$1`, s.posID); err != nil {
		w.rej.Inc("write_err")
	}
	w.rej.Inc("loop_alert")
	note := fmt.Sprintf("仓#%d 策略#%d closing 滞留：attempts=%d（阈 %d）滞留 %.1fh（阈 %.0fh）——平仓循环本身是事故不等耗尽；同策略开仓权已挂起，解冻唯一径=裁定席判词/业主令",
		s.posID, s.stratRef, s.attempts, int64(maxAtt), stuckH, maxHrs)
	if _, err := w.pool.Exec(ctx, `INSERT INTO ops_event(seat,event,ticket,note,esc_class,esc_scope)
		VALUES('仓巡','升件','',$1,'守卫环','fin_guard_close')`, note); err != nil {
		w.rej.Inc("write_err")
	}
	w.al.Alert(ctx, "guard_close_loop", note)
	log.Printf("ALERT [guard] %s", note)
}

// LiftSuspend 开仓权解冻（T-0423 熔断条款——解冻唯一径=裁定席判词或业主令；
// tools/guardlift CLI 同调本件）。无在挂行=err；解落 lifted_* 三列+ops_event
// 「开仓解冻」行（解冻非静默）。
func LiftSuspend(ctx context.Context, pool *pgxpool.Pool, strategyRef int64, by, reason string) error {
	if by != "裁定席" && by != "业主" {
		return fmt.Errorf("解冻主体 %q 越权——唯一径=裁定席判词/业主令", by)
	}
	tag, err := pool.Exec(ctx, `UPDATE fin_guard_suspend
		SET lifted_at=now(),lifted_by=$2,lift_reason=$3
		WHERE strategy_ref=$1 AND lifted_at IS NULL`,
		strategyRef, by, reason)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("策略 %d 无在挂行", strategyRef)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO ops_event(seat,event,note)
		VALUES($1,'开仓解冻',$2)`,
		by, fmt.Sprintf("策略#%d 开仓权解冻——%s", strategyRef, reason)); err != nil {
		log.Printf("WARN [guard] 解冻自报行写败（lifted 已落非回滚）: %v", err)
	}
	return nil
}
