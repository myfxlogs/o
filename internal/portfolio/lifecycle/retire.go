// retire.go —— 提案队列卫生闸（T-0161 自 run.go 拆立：eval 联动退位+陈旧回收面）。
// 溯源面=fin_proposal.retire_reason/retired_at（0065）——每条退役径必落因。
package lifecycle

import (
	"context"
	"log"
)

// retireEvalRejected eval 翻拒联动退位（T-0161 施工项①）：eval verdict=
// rejected 后，同 eval_id 的 fin_proposed/fin_armed 提案即失据（判定翻案
// =提案空气化），当日退位零滞留。幂等=state 谓词只命中未退行；触发径=
// ev_eval（0064 实质变触发器——verdict 翻面即醒 lifecycle，本拍即扫）。
// live/probe 策略行与持仓零触（提案只是派生凭据，仓位治理归 guard/lifecycle）。
func (r *Runner) retireEvalRejected(ctx context.Context) error {
	tag, err := r.pool.Exec(ctx, `UPDATE fin_proposal p
		SET state='fin_retired', retire_reason='eval_rejected', retired_at=now()
		WHERE p.state IN ('fin_proposed','fin_armed')
		AND EXISTS (SELECT 1 FROM fin_opp_eval e WHERE e.id=p.eval_id AND e.verdict='rejected')`)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[lifecycle] eval 翻拒联动退役 %d 提案（eval_rejected——判定翻案提案失据，当日退位零滞留）", tag.RowsAffected())
	}
	return nil
}

// retireStaleProposed 陈旧提案回收（T-0161 施工项②）：fin_proposed 超阈未
// 被吸收（executor 缺席/关联对象过期）→fin_retired——提案不再无限堆积占位。
// TTL=config lifecycle.proposal_stale_ttl_h，键缺/非正=默认 24h（不配键不瘫，
// armed_ttl_h 同例）。幂等=state+created_at 谓词；退后 eval 仍在——verdict
// 仍 proposed 的合法机会由评估写方下拍重新起草（open 闸只数未退行）。
func (r *Runner) retireStaleProposed(ctx context.Context) error {
	ttl := 24.0
	if v, err := r.cfg.GetFloat("lifecycle.proposal_stale_ttl_h"); err == nil && v > 0 {
		ttl = v
	}
	tag, err := r.pool.Exec(ctx, `UPDATE fin_proposal
		SET state='fin_retired', retire_reason='stale_expired', retired_at=now()
		WHERE state='fin_proposed' AND created_at<now()-make_interval(hours=>$1)`, ttl)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	if tag.RowsAffected() > 0 {
		log.Printf("[lifecycle] fin_proposed 超 %.0fh 陈旧回收 %d 行（stale_expired——提案队列止胀）", ttl, tag.RowsAffected())
	}
	return nil
}
