// ops_incremental.go —— eval/proposal 增量投影执行面（ops.go >300 棘轮拆分）：
// ts/id 水位回读重叠窗（O-30635 Marten 型跳读根修——分配序≠提交序，
// 低位行未提交期间高位行先入水位=永跳；重叠窗内幂等 upsert 兜回）。
package projector

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// projectEvals 评估桥投影（serve 谓词=fin_proposal 引用行∪在册策略自然键
// 行——D-0030 §三+O-36961 dirEntryGate 陈行根修；
// dualwrite=全量旁写）。拍尾顺带刷新 eval 脉冲（T-0485——谓词承重墙零触，
// 活性派生面走正本聚合）。
func (r *Runner) projectEvals(ctx context.Context) error {
	// ts 水位回读重叠窗：md 侧 research 各臂 goroutine 并发写——id/ts 分配序
	// ≠提交序（Marten 型：低 ts 行未提交期间高 ts 行先入水位=低位行永跳；
	// 内存水位重启才重对账）。upsert 幂等——重叠窗内重读零代价，窗宽对齐
	// replayTsTables 10min 惯例，晚提交落窗内由下拍兜回。
	wm := r.getWm("fin_opp_eval").Add(-evalOverlap)
	q := qEvalFresh
	if r.dualwrite {
		q = qEvalDual
	}
	rows, err := r.md.Query(ctx, q, wm)
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
	return r.refreshEvalPulse(ctx)
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
	// id 水位回读重叠窗（同 eval 面 Marten 型根修）：窗内已投影行重读由
	// uProposal 幂等 upsert 兜零代价；窗宽 500 id≈提案产出数月量级——
	// 晚提交行只要落在窗内即由下拍兜回，不再永久跳读。
	ids, err := r.newProposalIDs(ctx, r.getPropWm()-propIDOverlap)
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
	var mdEval map[int64][3]string // md eval_id→自然键（T-0211 译码底稿）
	if len(evalIDs) > 0 {
		erows, err := r.md.Query(ctx, qEvalByID, evalIDs)
		if err != nil {
			return fmt.Errorf("引用 eval 读取: %w", err)
		}
		ebatch, _, err := scanEvals(erows)
		if err != nil {
			return err
		}
		mdEval = map[int64][3]string{}
		for _, e := range ebatch {
			mdEval[e[0].(int64)] = [3]string{e[2].(string), e[3].(string), e[4].(string)}
		}
		if err := batchExec(ctx, r.sg, uEval, ebatch); err != nil {
			return fmt.Errorf("引用 eval 投影: %w", err)
		}
	}
	// T-0211 eval_id 译码：md 发号≠SG 发号（同键异 id 镜像后 SG 行持自有 id）——
	// 提案 eval_id 按 eval 自然键译 SG id（fin_0053 重指钦定径：提案行自带
	// inst/direction/net_apr，eval 引用=溯源面非载荷）。
	if len(mdEval) > 0 {
		var err error
		if batch, err = r.remapEvalIDs(ctx, mdEval, batch); err != nil {
			return err
		}
	}
	if err := batchExec(ctx, r.sg, uProposal, batch); err != nil {
		return fmt.Errorf("提案投影: %w", err)
	}
	r.bumpPropWm(maxID) // 批成后推进——批败行留窗（活性对账 max-id 径兜住）
	return nil
}
