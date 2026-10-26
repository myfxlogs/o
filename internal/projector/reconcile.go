// reconcile.go —— 启动/重连/活性对账自愈（T-0168 桩重点/D-0030 §四自愈条款）：
// full=true（boot）逐 id 集差补投影；false（活性兜底）max-id 窗口+10min 重刷片。
// 先修回写态再补正向投影——SG 侧在途态先行收敛，提案投影不被 SG dedupe 闸滞留。
package projector

import (
	"context"
	"fmt"
	"time"
)

// Reconcile 对账：谓词自算→回写态修复→正向缺口补投影→macro diff→ts 面重刷。
func (r *Runner) Reconcile(ctx context.Context, full bool) error {
	if err := r.refreshSets(ctx); err != nil {
		return err
	}
	if err := r.repairProposalStates(ctx); err != nil {
		return fmt.Errorf("回写态修复: %w", err)
	}
	if err := r.repairForwardGap(ctx, full); err != nil {
		return fmt.Errorf("正向缺口: %w", err)
	}
	if err := r.projectMacro(ctx); err != nil {
		return fmt.Errorf("macro diff: %w", err)
	}
	if err := r.replayTsTables(ctx, full); err != nil {
		return err
	}
	return nil
}

// repairForwardGap 提案+引用 eval 缺行补投影（杀连丢投影由此收敛——验收锚
// 「杀连→全表对账→集齐」）。full=两侧全 id 集差（boot 级）；否则 sgMax 后窗口。
func (r *Runner) repairForwardGap(ctx context.Context, full bool) error {
	sgMax := func(tbl string) (int64, error) {
		var m int64
		if err := r.sg.QueryRow(ctx, "SELECT COALESCE(max(id),0) FROM "+tbl).Scan(&m); err != nil {
			return 0, err
		}
		return m, nil
	}
	pMax, err := sgMax("fin_proposal")
	if err != nil {
		return err
	}
	var ids []int64
	if full {
		// 全 id 集差：md 有而 SG 无（杀连窗内丢投影不拘 id 窗）。
		mdIDs := map[int64]bool{}
		rows, err := r.md.Query(ctx, "SELECT id FROM fin_proposal")
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			mdIDs[id] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		sgIDs := map[int64]bool{}
		srows, err := r.sg.Query(ctx, "SELECT id FROM fin_proposal")
		if err != nil {
			return err
		}
		for srows.Next() {
			var id int64
			if err := srows.Scan(&id); err != nil {
				srows.Close()
				return err
			}
			sgIDs[id] = true
		}
		srows.Close()
		if err := srows.Err(); err != nil {
			return err
		}
		for id := range mdIDs {
			if !sgIDs[id] {
				ids = append(ids, id)
			}
		}
	} else {
		if ids, err = r.newProposalIDs(ctx, pMax); err != nil {
			return err
		}
	}
	if err := r.projectProposalsByID(ctx, ids); err != nil {
		return err
	}
	// 引用 eval 缺行（同径——eval 与 proposal 一起丢的杀连窗）。
	eMax, err := sgMax("fin_opp_eval")
	if err != nil {
		return err
	}
	q := `SELECT e.id FROM fin_opp_eval e WHERE e.id>$1
		AND EXISTS (SELECT 1 FROM fin_proposal p WHERE p.eval_id=e.id)`
	rows, err := r.md.Query(ctx, q, eMax)
	if err != nil {
		return err
	}
	var evalIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		evalIDs = append(evalIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(evalIDs) > 0 {
		er, err := r.md.Query(ctx, qEvalByID, evalIDs)
		if err != nil {
			return err
		}
		eb, _, err := scanEvals(er)
		if err != nil {
			return err
		}
		if err := batchExec(ctx, r.sg, uEval, eb); err != nil {
			return err
		}
	}
	return nil
}

// replayTsTables ts 桥面重刷：full=水印归零全量（boot/杀连自愈——同 ts 删行
// 只有归零重拉可见，active 集有界拉取）；否则 max(水印,now-10min) 覆断窗
// （事件失联期写的行由期界对账兜住）。
func (r *Runner) replayTsTables(ctx context.Context, full bool) error {
	since := time.Now().Add(-livenessEvery)
	if full {
		since = time.Time{}
		r.resetWm()
	}
	for _, p := range tsProjections {
		wm := r.getWm(p.wmKey)
		if wm.Before(since) {
			wm = since
		}
		rows, err := r.md.Query(ctx, p.query, wm, p.set(r))
		if err != nil {
			return fmt.Errorf("%s 重刷: %w", p.wmKey, err)
		}
		batch, max, err := p.scan(rows)
		if err != nil {
			return fmt.Errorf("%s 重刷扫描: %w", p.wmKey, err)
		}
		if err := batchExec(ctx, r.sg, p.upsert, batch); err != nil {
			return fmt.Errorf("%s 重刷投影: %w", p.wmKey, err)
		}
		r.bumpWm(p.wmKey, max)
	}
	return nil
}
