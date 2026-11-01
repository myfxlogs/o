// retire.go —— 提案队列+armed_ttl 策略面卫生闸（T-0161 自 run.go 拆立：
// eval 联动退位+陈旧回收面；T-0258 tag3 策略退役归位本件）。
// 溯源面=fin_proposal.retire_reason/retired_at（0065）+策略面
// fin_strategy_transition——每条退役径必落因。
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

// retireStaleArmedStrategies armed_ttl tag3 策略面退役（T-0258 根修）。
// 判据①活动锚=执行侧证据最新戳：GREATEST(entered_at, fin_position 末次
// 开仓, fin_execution 末次尝试, fin_eval_reject 末次闸拒) 超 ttl 才退——
// 被开仓闸持续拒的活策略（拦在开仓前、零 position/execution）拒留痕即
// 活动证据，全拒≠不活（7 条 live funding-hedge 误杀根修）；真 dormant
// 照退语义不丢。fin_eval_reject 无 strategy_ref/venue 列——按
// inst_id+direction 配且限 funding 闸域模板（rejectEvalReason 唯一写方、
// 落件模板值硬记 'funding-hedge'：xvenue-carry 拒行同归此值，故以闸域集
// 配非 s.template_ref 等值）。
// 留痕②：退役逐行走 Transition（degraded 超龄径同例——原裸 UPDATE 零
// transition 行=T-0255 取证断点根修），reason=armed_ttl_expired 同因续用。
func (r *Runner) retireStaleArmedStrategies(ctx context.Context, ttl float64) error {
	var fam []string
	for k := range fundingHedgeFamily {
		fam = append(fam, k)
	}
	rows, err := r.pool.Query(ctx, `SELECT s.id FROM fin_strategy s
		WHERE s.state IN ('live','probe') AND NOT EXISTS(
			SELECT 1 FROM fin_position p WHERE p.strategy_ref=s.id AND p.closed_at IS NULL)
		AND GREATEST(s.entered_at,
			(SELECT max(p2.opened_at) FROM fin_position p2 WHERE p2.strategy_ref=s.id),
			(SELECT max(e.created_at) FROM fin_execution e WHERE e.strategy_ref=s.id),
			(SELECT max(ej.ts) FROM fin_eval_reject ej
				WHERE ej.inst_id=s.inst_id AND ej.direction=s.direction
				AND s.template_ref=ANY($2))
		)<now()-make_interval(hours=>$1)`, ttl, fam)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := Transition(ctx, r.pool, id, "retired", "armed_ttl_expired", false); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		log.Printf("[lifecycle] 策略 #%d armed_ttl 超 %.0fh 退役（armed_ttl_expired——窗内零执行侧活动）", id, ttl)
	}
	return nil
}

// retireStalePaper paper 沙箱超龄退役（T-0286）：paper 行原无任何退役径
// ——滞留 paper 永占同键唯一槽（venue,inst_id,direction,template_ref），
// 上游起草闸「同键无非 retired 策略不起草」+领取端同键吸收径合围=墓碑锁
// （BTC-USD 季度对儿 3 天 24 提案连环吸收实证；degraded 同病 T-0098 先例
// 修）。TTL=config lifecycle.paper_ttl_h，键缺/非正=默认 168h（7d——
// 观测最大毕业龄锚定：286157 paper 滞 139h 后于 11-01 毕业、286894 滞
// 90.7h 同日毕业，72h 缺省会误杀慢毕业行；7d=21 个 funding 结算窗，
// 超窗未毕业=经济性证据已足，留行只锁键不产值；不配键不瘫）。纯龄锚
// 无活动豁免——活动豁免（T-0258 live/probe 径语义）会让「每 8h 一 fill
// 恒负 carry」行永生，锁键病灶复现。退役走 Transition 留痕
// （reason=paper_ttl_expired）；
// paper=沙箱虚拟面，fin_paper_fill 事实行零触、零钱径。
func (r *Runner) retireStalePaper(ctx context.Context) error {
	ttl := 168.0
	if v, err := r.cfg.GetFloat("lifecycle.paper_ttl_h"); err == nil && v > 0 {
		ttl = v
	}
	rows, err := r.pool.Query(ctx, `SELECT id FROM fin_strategy
		WHERE state='paper' AND entered_at < now() - make_interval(hours => $1)`, ttl)
	if err != nil {
		r.rej.Inc("write_err")
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if err := Transition(ctx, r.pool, id, "retired", "paper_ttl_expired", false); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		log.Printf("[lifecycle] #%d paper 超 %.0fh 退役（paper_ttl_expired——沙箱超龄同键槽释放）", id, ttl)
	}
	return nil
}
