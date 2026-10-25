package review

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// reviewSettled settled 未复盘仓→判决书（预测 net_apr 自 eval 证据 vs 实际净额；
// 同期排 8h/24h/72h 远期窗；falsified→经验库待归纳+模板修正提案行）。
func (r *Runner) reviewSettled(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT p.id, p.settled_amount, p.notional_usd,
		COALESCE((SELECT e.net_apr FROM fin_strategy s
			JOIN fin_proposal pr ON pr.id=s.proposal_ref
			JOIN fin_opp_eval e ON e.id=pr.eval_id
			WHERE s.id=(SELECT f.strategy_ref FROM fin_execution f WHERE f.id=p.execution_id)
			ORDER BY e.id DESC LIMIT 1), 0) AS pred_apr,
		p.execution_id
		FROM fin_position p
		WHERE p.settled_at IS NOT NULL AND NOT EXISTS(
			SELECT 1 FROM fin_review rv WHERE rv.position_ref=p.id)`)
	if err != nil {
		return err
	}
	type pos struct {
		id, execID             int64
		net, notional, predApr float64
	}
	var ps []pos
	for rows.Next() {
		var x pos
		if err := rows.Scan(&x.id, &x.net, &x.notional, &x.predApr, &x.execID); err != nil {
			rows.Close()
			return err
		}
		ps = append(ps, x)
	}
	rows.Close()
	for _, x := range ps {
		// 预测净额=net_apr×notional×持仓期（settle 窗缺省用 1 日当量——口径写入 detail）。
		pred := x.predApr * x.notional / 365
		var feeShare float64
		var fee float64
		_ = r.pool.QueryRow(ctx,
			"SELECT COALESCE(sum(-amount),0) FROM fin_cashflow WHERE position_ref=$1 AND kind='fee'", x.id).Scan(&fee)
		if x.net != 0 {
			feeShare = fee / abs(x.net)
		}
		verdict := Verdict(pred, x.net)
		attrib := Attribut(pred, x.net, feeShare)
		detail := []byte(fmt.Sprintf(`{"pred_apr":%.6f,"pred_window":"1d","notional":%.4f,"fee":%.4f}`,
			x.predApr, x.notional, fee))
		if _, err := r.pool.Exec(ctx, `INSERT INTO fin_review(position_ref,verdict,pred_net,actual_net,deviation,attribution,detail)
			VALUES($1,$2,$3,$4,$5,$6,$7)`,
			x.id, verdict, pred, x.net, pred-x.net, attrib, detail); err != nil {
			r.rej.Inc("write_err")
			return err
		}
		log.Printf("[review] 仓 #%d 判决书 verdict=%s pred=%.4f actual=%.4f 归因=%s",
			x.id, verdict, pred, x.net, attrib)
		// 远期窗排期（8/24/72h）
		for _, h := range []int{8, 24, 72} {
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_forward_check(obj_kind,obj_ref,window_h,target_ts,anchor_val)
				VALUES('position',$1,$2,now()+$3*interval '1 hour',$4)`,
				x.id, h, float64(h), pred); err != nil {
				return err
			}
		}
		if verdict == "falsified" {
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_review_verdict
				(position_ref,verdict,pred_apr,actual_pnl,attribution)
				VALUES($1,$2,$3,$4,$5) ON CONFLICT (position_ref) DO NOTHING`,
				x.id, verdict, pred, x.net, attrib); err != nil {
				r.rej.Inc("write_err")
				return err
			}
			r.stageLesson(x.id, pred, x.net, attrib)
			if _, err := r.pool.Exec(ctx, `INSERT INTO fin_template_proposal(template,param_key,cur_val,sug_val,evidence)
				VALUES('funding-hedge','research.funding.min_net_apr',$1,$2,$3)`,
				0, 0, detail); err != nil {
				return err
			}
			r.recycleFalsified(ctx, x.execID, attrib)
		}
	}
	return nil
}

// scheduleRejected 否决提案排期复测（若当时执行会怎样——拒对/拒错考据核心锚）。
func (r *Runner) scheduleRejected(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO fin_forward_check(obj_kind,obj_ref,window_h,target_ts,anchor_val)
		SELECT 'rejected_proposal', e.id, 24, e.ts+interval '24 hours',
			COALESCE(e.net_apr,0)*100/365
		FROM fin_opp_eval e WHERE e.verdict='rejected' AND e.template='funding-hedge'
		AND NOT EXISTS(SELECT 1 FROM fin_forward_check c WHERE c.obj_kind='rejected_proposal' AND c.obj_ref=e.id)`)
	return err
}

// judgeDue 到期填实际值+判对/错：已执行仓=Σfunding_settle 窗内实收；
// 否决提案=假设执行应得=Σmd_funding rate×notional−成本（判 anchor 符号对错）。
func (r *Runner) judgeDue(ctx context.Context) error {
	rows, err := r.pool.Query(ctx, `SELECT id,obj_kind,obj_ref,anchor_val,target_ts
		FROM fin_forward_check WHERE judged=false AND target_ts<=now()`)
	if err != nil {
		return err
	}
	type chk struct {
		id, ref int64
		kind    string
		anchor  float64
		target  time.Time
	}
	var cs []chk
	for rows.Next() {
		var c chk
		if err := rows.Scan(&c.id, &c.kind, &c.ref, &c.anchor, &c.target); err != nil {
			rows.Close()
			return err
		}
		cs = append(cs, c)
	}
	rows.Close()
	for _, c := range cs {
		var actual float64
		var err error
		if c.kind == "position" {
			err = r.pool.QueryRow(ctx, `SELECT COALESCE(sum(amount),0) FROM fin_cashflow
				WHERE position_ref=$1 AND kind='funding_settle' AND occurred_at<=$2`, c.ref, c.target).Scan(&actual)
		} else {
			// 否决提案假设执行：inst_id 自 eval 行，notional 按锚值口径 100
			var instID string
			if err2 := r.pool.QueryRow(ctx,
				"SELECT inst_id FROM fin_opp_eval WHERE id=$1", c.ref).Scan(&instID); err2 != nil {
				continue
			}
			err = r.pool.QueryRow(ctx, `SELECT COALESCE(sum(funding_rate),0)*100 FROM md_funding
				WHERE inst_id=$1 AND ts<=$2`, instID, c.target).Scan(&actual)
		}
		if err != nil {
			continue
		}
		ok := (c.anchor > 0) == (actual > 0) // 判对=符号一致（方向判据对错）
		if _, err := r.pool.Exec(ctx,
			"UPDATE fin_forward_check SET actual_val=$2,judged=true,judged_ok=$3,judged_at=now() WHERE id=$1",
			c.id, actual, ok); err != nil {
			return err
		}
		log.Printf("[review] forward_check #%d %s/%d anchor=%.4f actual=%.4f judged_ok=%v",
			c.id, c.kind, c.ref, c.anchor, actual, ok)
	}
	return nil
}

// stageLesson falsified→待归纳派生镜像（best-effort——T-0066：权威在
// fin_review_verdict 库表；容器 FS 易失，写败只 log 一次/进程不刷屏）。
func (r *Runner) stageLesson(posID int64, pred, actual float64, attrib string) {
	if r.stagingLogged {
		return // 写败已记过一次——镜像降级静默不刷屏
	}
	marker := fmt.Sprintf("<!-- pos:%d -->", posID)
	f, err := os.OpenFile(stagingFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		r.stagingLogged = true
		log.Printf("[review] 待归纳镜像写败（权威在 fin_review_verdict——镜像降级静默）: %v", err)
		return
	}
	defer f.Close()
	if _, err := fmt.Fprintf(f, "%s\n- falsified 复盘 pos#%d：pred_net=%.4f actual_net=%.4f 归因=%s（T-0017 自动行）\n",
		marker, posID, pred, actual, attrib); err != nil {
		r.stagingLogged = true
		log.Printf("[review] 待归纳镜像写败（权威在库——镜像降级静默）: %v", err)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// recycleFalsified falsified 判决回炉（T-0103 ②）：判决落库即联动策略状态位
// ——armed 同键提案退役+策略回 degraded 待复盘（live→degraded 合法径），不再
// 原地待命被同键 armed 续开（#1662 判据分裂振荡病灶）。
func (r *Runner) recycleFalsified(ctx context.Context, execID int64, attrib string) {
	var sid int64
	var inst, dir string
	if err := r.pool.QueryRow(ctx, `SELECT s.id,s.inst_id,s.direction FROM fin_strategy s
		JOIN fin_execution f ON f.strategy_ref=s.id WHERE f.id=$1`,
		execID).Scan(&sid, &inst, &dir); err != nil {
		r.rej.Inc("write_err")
		log.Printf("ALERT [review] falsified 回炉策略行查败（exec=%d）: %v", execID, err)
		return
	}
	if tag, err := r.pool.Exec(ctx, `UPDATE fin_proposal SET state='fin_retired'
		WHERE inst_id=$1 AND direction=$2 AND state='fin_armed'`, inst, dir); err == nil && tag.RowsAffected() > 0 {
		log.Printf("[review] falsified 回炉——同键 armed 提案退役 %d 行（%s %s）", tag.RowsAffected(), inst, dir)
	}
	// live→degraded 合法径内联迁（review→lifecycle 跨包向禁——D-514）：
	// state='live' 谓词=只迁 live 行，非 live 行零动不误迁。
	tag, err := r.pool.Exec(ctx, `UPDATE fin_strategy SET state='degraded',entered_at=now()
		WHERE id=$1 AND state='live'`, sid)
	if err != nil {
		r.rej.Inc("write_err")
		log.Printf("ALERT [review] 策略 #%d falsified 回炉降级败: %v", sid, err)
		return
	}
	if tag.RowsAffected() == 0 {
		return // 非 live 态（已退役/降级/未晋）——回炉径天然无事
	}
	if _, err := r.pool.Exec(ctx, `INSERT INTO fin_strategy_transition(strategy_id,from_state,to_state,reason)
		VALUES($1,'live','degraded',$2)`, sid,
		fmt.Sprintf("falsified 判决回炉（归因=%s）——降级待复盘非原地待命", attrib)); err != nil {
		r.rej.Inc("write_err")
		return
	}
	log.Printf("[review] 策略 #%d %s falsified（归因=%s）→degraded 回炉", sid, inst, attrib)
}
