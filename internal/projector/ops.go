// ops.go —— 投影执行面（proposal/eval/macro/回写）：批量 upsert（SG 镜像）+
// 反向回写。水印推进使 ev_* 重复唤醒零代价（空集即返）；ts 判桥表在 tsops.go。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// projectEvals 评估桥投影（谓词=仅被 fin_proposal 引用的行——D-0030 §三）。
func (r *Runner) projectEvals(ctx context.Context) error {
	wm := r.getWm("fin_opp_eval")
	rows, err := r.md.Query(ctx, qEvalFresh, wm)
	if err != nil {
		return fmt.Errorf("eval 查询: %w", err)
	}
	batch, max, err := scanEvals(rows)
	if err != nil {
		return err
	}
	if err := batchExec(ctx, r.sg, uEval, batch); err != nil {
		return fmt.Errorf("eval 投影: %w", err)
	}
	r.bumpWm("fin_opp_eval", max)
	return nil
}

func scanEvals(rows pgx.Rows) ([][]any, time.Time, error) {
	defer rows.Close()
	var out [][]any
	var max time.Time
	for rows.Next() {
		var id int64
		var ts time.Time
		var inst, template, direction, verdict string
		var gross, net *string
		var legs, evidence []byte
		if err := rows.Scan(&id, &ts, &inst, &template, &direction, &gross, &net, &verdict, &legs, &evidence); err != nil {
			return nil, max, err
		}
		out = append(out, []any{id, ts, inst, template, direction, gross, net, verdict, legs, evidence})
		if ts.After(max) {
			max = ts
		}
	}
	return out, max, rows.Err()
}

// projectNewProposals 正向提案投影（id 水印增量——提案 insert-only，ev_proposal
// 仅 INSERT 发）+其引用 eval 行随件（admit join 面）。
func (r *Runner) projectNewProposals(ctx context.Context) error {
	ids, err := r.newProposalIDs(ctx, r.getPropWm())
	if err != nil {
		return err
	}
	return r.projectProposalsByID(ctx, ids)
}

func (r *Runner) newProposalIDs(ctx context.Context, after int64) ([]int64, error) {
	rows, err := r.md.Query(ctx, "SELECT id FROM fin_proposal WHERE id>$1 ORDER BY id", after)
	if err != nil {
		return nil, fmt.Errorf("提案增量 id: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Runner) projectProposalsByID(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.md.Query(ctx, qProposalByID, ids)
	if err != nil {
		return fmt.Errorf("提案读取: %w", err)
	}
	defer rows.Close()
	var batch [][]any
	var evalIDs []int64
	var maxID int64
	for rows.Next() {
		var id, evalID int64
		var inst, direction string
		var netApr *string
		var created time.Time
		if err := rows.Scan(&id, &evalID, &inst, &direction, &netApr, &created); err != nil {
			return err
		}
		batch = append(batch, []any{id, evalID, inst, direction, netApr, created})
		evalIDs = append(evalIDs, evalID)
		if id > maxID {
			maxID = id
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// eval 先行（SG fin_proposal.eval_id FK 引用序——同批内先 eval 后提案）。
	if len(evalIDs) > 0 {
		erows, err := r.md.Query(ctx, qEvalByID, evalIDs)
		if err != nil {
			return fmt.Errorf("引用 eval 读取: %w", err)
		}
		ebatch, _, err := scanEvals(erows)
		if err != nil {
			return err
		}
		if err := batchExec(ctx, r.sg, uEval, ebatch); err != nil {
			return fmt.Errorf("引用 eval 投影: %w", err)
		}
	}
	if err := batchExec(ctx, r.sg, uProposal, batch); err != nil {
		return fmt.Errorf("提案投影: %w", err)
	}
	r.bumpPropWm(maxID) // 批成后推进——批败行留窗（活性对账 max-id 径兜住）
	return nil
}

// projectMacro 宏观日历全表 diff（~30 行 event_date 键——无 ts 全列覆写语义）。
func (r *Runner) projectMacro(ctx context.Context) error {
	mdRows, err := r.md.Query(ctx, qMacroAll)
	if err != nil {
		return fmt.Errorf("macro md: %w", err)
	}
	type ev struct {
		date   time.Time
		kind   string
		brief  string
		source string
	}
	want := map[time.Time]ev{}
	defer mdRows.Close()
	for mdRows.Next() {
		var e ev
		if err := mdRows.Scan(&e.date, &e.kind, &e.brief, &e.source); err != nil {
			return err
		}
		want[e.date] = e
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	have := map[time.Time]ev{}
	sgRows, err := r.sg.Query(ctx, `SELECT event_date,kind,title,source FROM md_macro_event`)
	if err != nil {
		return fmt.Errorf("macro sg: %w", err)
	}
	defer sgRows.Close()
	for sgRows.Next() {
		var e ev
		if err := sgRows.Scan(&e.date, &e.kind, &e.brief, &e.source); err != nil {
			return err
		}
		have[e.date] = e
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for d, e := range want {
		if h, ok := have[d]; !ok || h != e {
			batch = append(batch, []any{e.date, e.kind, e.brief, e.source})
		}
	}
	return batchExec(ctx, r.sg, uMacro, batch)
}

// repairProposalStates 反向回写（SG state 列组=正本——D-0030 写权矩阵）：
// SG 活跃/近期退役行对拍 PG-md，三列差即回写。前向投影不写 state 列组=
// 不触发 SG ev_proposal_sg（UPDATE OF state 按 SET 清单）——零回环构造性成立。
func (r *Runner) repairProposalStates(ctx context.Context) error {
	sgRows, err := r.sg.Query(ctx, `SELECT id,state,retire_reason,retired_at FROM fin_proposal
		WHERE state IN ('fin_proposed','fin_armed') OR retired_at>now()-interval '24 hours'`)
	if err != nil {
		return fmt.Errorf("回写面查询: %w", err)
	}
	type st struct {
		state  string
		reason *string
		at     *time.Time
	}
	sg := map[int64]st{}
	defer sgRows.Close()
	for sgRows.Next() {
		var id int64
		var s st
		if err := sgRows.Scan(&id, &s.state, &s.reason, &s.at); err != nil {
			return err
		}
		sg[id] = s
	}
	if err := sgRows.Err(); err != nil {
		return err
	}
	if len(sg) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(sg))
	for id := range sg {
		ids = append(ids, id)
	}
	mdRows, err := r.md.Query(ctx, `SELECT id,state,retire_reason,retired_at FROM fin_proposal WHERE id=ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("回写对拍: %w", err)
	}
	md := map[int64]st{}
	defer mdRows.Close()
	for mdRows.Next() {
		var id int64
		var s st
		if err := mdRows.Scan(&id, &s.state, &s.reason, &s.at); err != nil {
			return err
		}
		md[id] = s
	}
	if err := mdRows.Err(); err != nil {
		return err
	}
	var batch [][]any
	for id, s := range sg {
		if m, ok := md[id]; !ok || m.state != s.state || !reasonEq(m.reason, s.reason) || !timeEq(m.at, s.at) {
			batch = append(batch, []any{id, s.state, s.reason, s.at})
		}
	}
	return batchExec(ctx, r.md, uProposalReverse, batch)
}

func reasonEq(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func timeEq(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
