// saga.go —— 流程级补偿账（工单#25807——#25771 裁定⑤ adopt 方法/reject
// Temporal 框架）。腿级 rolled_back=单腿终态已有；本件=流程编排账：
// 一次开/平仓=多步序（exec_row→earn_redeem→legN→position_row），每步落
// fin_saga.steps——「做到哪步+按序调 undo」；断点恢复=启动臂扫陈 active
// 转 compensating 按 done 步反序调注册 undo。
//
// 恢复策略选型（票面④）：**转 compensating 非续行**——active 陈 saga 语义
// =宿主进程死在中途，步间真态未知（venue 侧或成或未）；续行=再下单可能
// 双开（无幂等位可复用——clOrdId 拍内生成非可推）；补偿=把已证 done 步
// 反序收回，残留歧义靠 rollback 自身歧态势升件兜底——fail-closed 选型。
// earn_redeem 无 undo_fn_ref（EarnMover 面无回购 API——赎回款留交易户，
// 资金侧无敞口风险，停泊回拢归 earn 拍自治）。
package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

func tern(ok bool, a, b string) string {
	if ok {
		return a
	}
	return b
}

type sagaStep struct {
	Name    string     `json:"name"`
	Status  string     `json:"status"`            // pending|done|failed|skipped|undone|undo_failed
	DoneAt  *time.Time `json:"done_at,omitempty"` // 步成时刻
	UndoRef string     `json:"undo_fn_ref"`       // 代码内注册回调名（空=无 undo 面）
	Note    string     `json:"note,omitempty"`    // 败因/处置注
}

// sagaBegin 开流程账：全步 pending 落 active 行（undo_fn_ref=注册回调名录）。
func (r *Runner) sagaBegin(ctx context.Context, flowType string, refID int64, steps []sagaStep) int64 {
	b, _ := json.Marshal(steps)
	var id int64
	if err := r.pool.QueryRow(ctx, `INSERT INTO fin_saga(flow_type,state,ref_id,steps)
		VALUES($1,'active',$2,$3) RETURNING id`, flowType, refID, b).Scan(&id); err != nil {
		r.rej.Inc("write_err")
		log.Printf("[saga] 开账败（流程账缺位——断点恢复不可见）: %v", err)
	}
	return id
}

// sagaMark 步态写回（读改写在 Go 侧——单 saga 单写者序内安全；id=0 跳过=桩测免炸）。
func (r *Runner) sagaMark(ctx context.Context, sagaID int64, name, status, note string) {
	if sagaID == 0 {
		return
	}
	var b []byte
	if err := r.pool.QueryRow(ctx, `SELECT steps FROM fin_saga WHERE id=$1`, sagaID).Scan(&b); err != nil {
		r.rej.Inc("write_err")
		return
	}
	var steps []sagaStep
	if json.Unmarshal(b, &steps) != nil {
		return
	}
	now := time.Now().UTC()
	for i := range steps {
		if steps[i].Name == name {
			steps[i].Status = status
			steps[i].Note = note
			if status == "done" {
				steps[i].DoneAt = &now
			}
		}
	}
	nb, _ := json.Marshal(steps)
	if _, err := r.pool.Exec(ctx, `UPDATE fin_saga SET steps=$2 WHERE id=$1`, sagaID, nb); err != nil {
		r.rej.Inc("write_err")
	}
}

// sagaClose 流程态收口（done|failed——done_at 同钉）。
func (r *Runner) sagaClose(ctx context.Context, sagaID int64, state string) {
	if sagaID == 0 {
		return
	}
	if _, err := r.pool.Exec(ctx, `UPDATE fin_saga SET state=$2,done_at=now() WHERE id=$1 AND state='active'`,
		sagaID, state); err != nil {
		r.rej.Inc("write_err")
	}
}

// sagaSteps 步序读出（补偿序消费面）。
func (r *Runner) sagaSteps(ctx context.Context, sagaID int64) ([]sagaStep, string) {
	var b []byte
	var ft string
	if err := r.pool.QueryRow(ctx, `SELECT steps,flow_type FROM fin_saga WHERE id=$1`, sagaID).Scan(&b, &ft); err != nil {
		return nil, ft
	}
	var steps []sagaStep
	json.Unmarshal(b, &steps)
	return steps, ft
}

// sagaCompensate 显式回调补偿：done 步反序调注册 undo；undo 败=undo_failed
// 钉步+流程 failed+升件留尸（fail-closed 不静默吞——票面③）。
func (r *Runner) sagaCompensate(ctx context.Context, sagaID, execID int64) {
	steps, flowType := r.sagaSteps(ctx, sagaID)
	if _, err := r.pool.Exec(ctx, `UPDATE fin_saga SET state='compensating' WHERE id=$1`, sagaID); err != nil {
		r.rej.Inc("write_err")
	}
	failed := false
	for i := len(steps) - 1; i >= 0; i-- {
		st := steps[i]
		if st.Status != "done" || st.UndoRef == "" {
			continue
		}
		err := r.sagaUndo(ctx, st.UndoRef, execID, st.Name, flowType)
		status := "undone"
		if err != nil {
			status, failed = "undo_failed", true
		}
		r.sagaMark(ctx, sagaID, st.Name, status, fmt.Sprint(err))
	}
	if failed {
		r.sagaCloseComp(ctx, sagaID, "failed")
		r.alert(ctx, "saga_compensate_failed",
			fmt.Sprintf("saga#%d exec#%d 补偿败——留尸人查（undo_failed 步在账）", sagaID, execID))
		return
	}
	r.sagaCloseComp(ctx, sagaID, "done")
}

// sagaCloseComp 补偿径收口（compensating→终态——sagaClose 的 active 谓词不适用）。
func (r *Runner) sagaCloseComp(ctx context.Context, sagaID int64, state string) {
	if _, err := r.pool.Exec(ctx, `UPDATE fin_saga SET state=$2,done_at=now() WHERE id=$1`,
		sagaID, state); err != nil {
		r.rej.Inc("write_err")
	}
}

// sagaUndo 注册回调分派面——undo_fn_ref 闭集（票面③显式回调非自动推导）：
//
//	rollback_leg    = 该步对应 filled 腿反向平仓（复用 rollback 歧态势/升件链）
//	delete_position = 仓行物理删（open 径 position_row 步——断点时仓壳半成）
//	fail_exec       = fin_execution 行 failed 收口（exec_row 步 undo 位）
func (r *Runner) sagaUndo(ctx context.Context, ref string, execID int64, stepName, flowType string) error {
	switch ref {
	case "rollback_leg":
		var leg Leg
		var ticket *int64
		var legNo int
		if _, err := fmt.Sscanf(stepName, "leg%d", &legNo); err != nil {
			return fmt.Errorf("步名非 legN: %s", stepName)
		}
		// 重建腿=腿行在账字段（sz=venue 方言请求量原样回，side/inst/venue 原列；
		// Close 由 flow_type 派生——open 流腿 undo=平(true)，close 流腿 undo=重开(false)）。
		if err := r.pool.QueryRow(ctx, `SELECT venue,inst_id,side,sz::text,venue_ticket
			FROM fin_order_leg WHERE execution_id=$1 AND leg_no=$2 AND state='filled'
			ORDER BY id DESC LIMIT 1`, execID, legNo).Scan(&leg.Venue, &leg.InstID, &leg.Side, &leg.Sz, &ticket); err != nil {
			return fmt.Errorf("filled 腿行缺: %w", err)
		}
		leg.Close = flowType == "open"
		if ticket != nil {
			leg.Ticket = *ticket
		}
		res := OrderResult{FillTicket: leg.Ticket}
		r.rollback(ctx, execID, leg, res)
		// rollback 无返——败径内部 setState('rollback_failed')+升件；补偿败
		// 判定=读回 exec 态（票面③补偿败升件留尸不静默吞）。
		var st string
		if err := r.pool.QueryRow(ctx, `SELECT state FROM fin_execution WHERE id=$1`, execID).Scan(&st); err != nil {
			return err
		}
		if st == "rollback_failed" {
			return fmt.Errorf("rollback_failed（歧态势升件链已发）")
		}
		return nil
	case "delete_position":
		_, err := r.pool.Exec(ctx, `DELETE FROM fin_position WHERE execution_id=$1`, execID)
		return err
	case "fail_exec":
		// 仅收 'open' 残行——leg undo 已收口态（rolled_back/closed）禁覆写。
		_, err := r.pool.Exec(ctx, `UPDATE fin_execution SET state='failed' WHERE id=$1 AND state='open'`, execID)
		return err
	}
	return fmt.Errorf("undo_fn_ref 非闭集: %s", ref)
}

// ResumeStaleSagas 断点恢复臂——启动时扫陈 active saga 转 compensating
// （选型注在文件头：转补偿非续行）。陈阈 10min 硬钉（staleexec 1h 字面量
// 先例——拍内死 vs 崩溃不可分，10min>正常 Execute 全程上界）。返回处置数。
func (r *Runner) ResumeStaleSagas(ctx context.Context) int {
	rows, err := r.pool.Query(ctx, `SELECT id,ref_id FROM fin_saga
		WHERE state='active' AND created_at<now()-interval '10 minutes'`)
	if err != nil {
		r.rej.Inc("write_err")
		return 0
	}
	defer rows.Close()
	type stale struct{ id, ref int64 }
	var list []stale
	for rows.Next() {
		var s stale
		var ref *int64
		if rows.Scan(&s.id, &ref) == nil && ref != nil {
			s.ref = *ref
			list = append(list, s)
		}
	}
	for _, s := range list {
		log.Printf("[saga] #%d 陈 active（exec#%d）——断点恢复转 compensating", s.id, s.ref)
		r.sagaCompensate(ctx, s.id, s.ref)
	}
	return len(list)
}
